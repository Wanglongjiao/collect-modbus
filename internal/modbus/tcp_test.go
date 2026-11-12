package modbus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	smodbus "github.com/simonvetter/modbus"
)

// localSlave 内存版从站：寄存器值 = UnitId*1000 + 地址 + 下标，线圈值按地址奇偶翻转。
type localSlave struct{}

func (localSlave) HandleCoils(req *smodbus.CoilsRequest) ([]bool, error) {
	if req.IsWrite {
		return nil, nil
	}
	out := make([]bool, req.Quantity)
	for i := range out {
		out[i] = (req.Addr+uint16(i))%2 == 1
	}
	return out, nil
}

func (localSlave) HandleDiscreteInputs(req *smodbus.DiscreteInputsRequest) ([]bool, error) {
	out := make([]bool, req.Quantity)
	for i := range out {
		out[i] = (req.Addr+uint16(i))%2 == 1
	}
	return out, nil
}

func (localSlave) HandleHoldingRegisters(req *smodbus.HoldingRegistersRequest) ([]uint16, error) {
	if req.IsWrite {
		return nil, nil
	}
	out := make([]uint16, req.Quantity)
	for i := range out {
		out[i] = uint16(req.UnitId)*1000 + req.Addr + uint16(i)
	}
	return out, nil
}

func (localSlave) HandleInputRegisters(req *smodbus.InputRegistersRequest) ([]uint16, error) {
	out := make([]uint16, req.Quantity)
	for i := range out {
		out[i] = uint16(req.UnitId)*1000 + req.Addr + uint16(i)
	}
	return out, nil
}

// startSlave 起一个本地 Modbus TCP 从站，返回 host/port。
func startSlave(t *testing.T) (string, int) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("申请空闲端口失败: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	srv, err := smodbus.NewServer(&smodbus.ServerConfiguration{
		URL:        fmt.Sprintf("tcp://127.0.0.1:%d", port),
		Timeout:    2 * time.Second,
		MaxClients: 8,
		Logger:     log.New(io.Discard, "", 0),
	}, localSlave{})
	if err != nil {
		t.Fatalf("创建模拟从站失败: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("启动模拟从站失败: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop() })
	return "127.0.0.1", port
}

func mustTCPConn(t *testing.T, host string, port, poolSize int) *TCPConn {
	t.Helper()
	c, err := NewTCPConn(host, port, 2*time.Second, poolSize)
	if err != nil {
		t.Fatalf("NewTCPConn: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestTCPConnReadHoldingRegisters(t *testing.T) {
	host, port := startSlave(t)
	c := mustTCPConn(t, host, port, 1)

	raw, err := c.Read(context.Background(), 1, AreaHoldingRegister, 100, 3)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	// 1100/1101/1102 的大端编码
	want := []byte{0x04, 0x4C, 0x04, 0x4D, 0x04, 0x4E}
	if !bytes.Equal(raw, want) {
		t.Errorf("raw = % X, want % X", raw, want)
	}
}

func TestTCPConnReadInputRegisters(t *testing.T) {
	host, port := startSlave(t)
	c := mustTCPConn(t, host, port, 1)

	raw, err := c.Read(context.Background(), 2, AreaInputRegister, 10, 2)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	// 2010/2011 的大端编码
	want := []byte{0x07, 0xDA, 0x07, 0xDB}
	if !bytes.Equal(raw, want) {
		t.Errorf("raw = % X, want % X", raw, want)
	}
}

func TestTCPConnReadCoils(t *testing.T) {
	host, port := startSlave(t)
	c := mustTCPConn(t, host, port, 1)

	raw, err := c.Read(context.Background(), 1, AreaCoil, 0, 4)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := []byte{0, 1, 0, 1}
	if !bytes.Equal(raw, want) {
		t.Errorf("raw = %v, want %v", raw, want)
	}
}

func TestTCPConnReadDiscreteInputs(t *testing.T) {
	host, port := startSlave(t)
	c := mustTCPConn(t, host, port, 1)

	raw, err := c.Read(context.Background(), 1, AreaDiscreteInput, 1, 2)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := []byte{1, 0}
	if !bytes.Equal(raw, want) {
		t.Errorf("raw = %v, want %v", raw, want)
	}
}

// 同一连接池服务不同从站地址（unitID 逐请求切换）。
func TestTCPConnUnitIDPerRequest(t *testing.T) {
	host, port := startSlave(t)
	c := mustTCPConn(t, host, port, 2)

	for _, unit := range []uint8{1, 7, 1} {
		raw, err := c.Read(context.Background(), unit, AreaHoldingRegister, 5, 1)
		if err != nil {
			t.Fatalf("unit %d Read: %v", unit, err)
		}
		want := registersToBytes([]uint16{uint16(unit)*1000 + 5})
		if !bytes.Equal(raw, want) {
			t.Errorf("unit %d raw = % X, want % X", unit, raw, want)
		}
	}
}

func TestTCPConnConcurrentRead(t *testing.T) {
	host, port := startSlave(t)
	c := mustTCPConn(t, host, port, 3)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(unit uint8) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				raw, err := c.Read(context.Background(), unit, AreaHoldingRegister, 20, 2)
				if err != nil {
					t.Errorf("unit %d Read: %v", unit, err)
					return
				}
				want := registersToBytes([]uint16{uint16(unit)*1000 + 20, uint16(unit)*1000 + 21})
				if !bytes.Equal(raw, want) {
					t.Errorf("unit %d raw = % X, want % X", unit, raw, want)
					return
				}
			}
		}(uint8(i%5 + 1))
	}
	wg.Wait()
}

func TestNewTCPConnDialError(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("申请空闲端口失败: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close() // 端口关闭后建连应失败

	if c, err := NewTCPConn("127.0.0.1", port, 200*time.Millisecond, 1); err == nil {
		_ = c.Close()
		t.Fatal("建连失败时应返回错误")
	}
}

func TestTCPConnClose(t *testing.T) {
	host, port := startSlave(t)
	c := mustTCPConn(t, host, port, 1)

	if !c.IsConnected() {
		t.Fatal("新建连接应处于可用状态")
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if c.IsConnected() {
		t.Error("Close 后 IsConnected 应返回 false")
	}
	if _, err := c.Read(context.Background(), 1, AreaHoldingRegister, 0, 1); !errors.Is(err, ErrNotConnected) {
		t.Errorf("Close 后 Read 应返回 ErrNotConnected，实际: %v", err)
	}
	if err := c.Write(context.Background(), 1, AreaHoldingRegister, 0, []byte{0, 1}); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("Write 应返回 ErrNotImplemented，实际: %v", err)
	}
}

func TestTCPConnReadValidation(t *testing.T) {
	host, port := startSlave(t)
	c := mustTCPConn(t, host, port, 1)
	ctx := context.Background()

	if _, err := c.Read(ctx, 1, AreaHoldingRegister, 0, MaxReadRegisters+1); !errors.Is(err, ErrProtocolLimit) {
		t.Errorf("超上限应返回 ErrProtocolLimit，实际: %v", err)
	}
	if _, err := c.Read(ctx, 1, AreaCoil, 0, MaxReadCoils+1); !errors.Is(err, ErrProtocolLimit) {
		t.Errorf("超上限应返回 ErrProtocolLimit，实际: %v", err)
	}
	if _, err := c.Read(ctx, 1, AreaHoldingRegister, 0, 0); err == nil {
		t.Error("count 为 0 应返回错误")
	}
	if _, err := c.Read(ctx, 1, Area(99), 0, 1); err == nil {
		t.Error("非法 area 应返回错误")
	}
}
