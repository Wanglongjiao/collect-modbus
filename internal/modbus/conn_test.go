package modbus

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	smodbus "github.com/simonvetter/modbus"
)

// exclusiveClient 记录自身是否被并发复用，并按 value+addr 返回寄存器值。
type exclusiveClient struct {
	value    uint16
	inUse    atomic.Int32
	reused   atomic.Bool
	failOnce atomic.Bool
}

func (e *exclusiveClient) ReadRegisters(addr, quantity uint16, _ smodbus.RegType) ([]uint16, error) {
	if e.inUse.Add(1) > 1 {
		e.reused.Store(true)
	}
	defer e.inUse.Add(-1)

	time.Sleep(50 * time.Microsecond)

	if e.failOnce.CompareAndSwap(true, false) {
		return nil, errors.New("fake read failed")
	}
	out := make([]uint16, quantity)
	for i := range out {
		out[i] = e.value + addr + uint16(i)
	}
	return out, nil
}

func (e *exclusiveClient) ReadCoils(_, quantity uint16) ([]bool, error) {
	return make([]bool, quantity), nil
}

func (e *exclusiveClient) ReadDiscreteInputs(_, quantity uint16) ([]bool, error) {
	return make([]bool, quantity), nil
}

func (e *exclusiveClient) SetUnitId(uint8) error { return nil }
func (e *exclusiveClient) Close() error          { return nil }

// 借出的连接独占使用：并发读时任一物理连接不会被同时复用，且返回值不错位。
func TestConnNoConcurrentReuse(t *testing.T) {
	const (
		size       = 3
		goroutines = 10
		loops      = 20
	)

	var clients []*exclusiveClient
	c, err := newConn("fake://local", size, func() (rawClient, error) {
		ec := &exclusiveClient{value: uint16(len(clients)+1) * 100}
		clients = append(clients, ec)
		return ec, nil
	})
	if err != nil {
		t.Fatalf("newConn: %v", err)
	}
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				raw, err := c.Read(context.Background(), 1, AreaHoldingRegister, 10, 2)
				if err != nil {
					t.Errorf("Read: %v", err)
					return
				}
				if !oneOfRaw(raw, clients, 10) {
					t.Errorf("raw = % X 不属于任一连接应返回的值，存在数据错位", raw)
					return
				}
			}
		}()
	}
	wg.Wait()

	for _, ec := range clients {
		if ec.reused.Load() {
			t.Errorf("连接 value=%d 被并发复用", ec.value)
		}
	}
}

// oneOfRaw 校验返回字节是否等于某个连接在该地址上的预期值（大端）。
func oneOfRaw(raw []byte, clients []*exclusiveClient, addr uint16) bool {
	for _, ec := range clients {
		want := registersToBytes([]uint16{ec.value + addr, ec.value + addr + 1})
		if bytes.Equal(raw, want) {
			return true
		}
	}
	return false
}

// 读失败时连接必须归还，否则容量 1 的池会被耗尽。
func TestConnReleaseOnReadError(t *testing.T) {
	fc := &exclusiveClient{value: 100}
	fc.failOnce.Store(true)

	c, err := newConn("fake://local", 1, func() (rawClient, error) { return fc, nil })
	if err != nil {
		t.Fatalf("newConn: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, err := c.Read(ctx, 1, AreaHoldingRegister, 0, 1); err == nil {
		t.Fatal("首次读应返回错误")
	}

	raw, err := c.Read(ctx, 1, AreaHoldingRegister, 0, 1)
	if err != nil {
		t.Fatalf("失败后连接未归还，第二次读报错: %v", err)
	}
	want := registersToBytes([]uint16{100})
	if !bytes.Equal(raw, want) {
		t.Errorf("raw = % X, want % X", raw, want)
	}
}
