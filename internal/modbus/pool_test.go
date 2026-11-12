package modbus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	smodbus "github.com/simonvetter/modbus"
)

// fakeClient 用于无设备环境下验证池的借还语义。
type fakeClient struct {
	id     int
	closed atomic.Int32
}

func (f *fakeClient) ReadCoils(_, quantity uint16) ([]bool, error) {
	return make([]bool, quantity), nil
}

func (f *fakeClient) ReadDiscreteInputs(_, quantity uint16) ([]bool, error) {
	return make([]bool, quantity), nil
}

func (f *fakeClient) ReadRegisters(_, quantity uint16, _ smodbus.RegType) ([]uint16, error) {
	return make([]uint16, quantity), nil
}

func (f *fakeClient) SetUnitId(uint8) error { return nil }

func (f *fakeClient) Close() error {
	f.closed.Add(1)
	return nil
}

func TestNewPoolSizeClamp(t *testing.T) {
	if got := newPool(0).size; got != 1 {
		t.Errorf("newPool(0).size = %d, want 1", got)
	}
	if got := newPool(-3).size; got != 1 {
		t.Errorf("newPool(-3).size = %d, want 1", got)
	}
	if got := newPool(4).size; got != 4 {
		t.Errorf("newPool(4).size = %d, want 4", got)
	}
}

func TestPoolOfferFull(t *testing.T) {
	p := newPool(1)
	if !p.offer(&fakeClient{id: 1}) {
		t.Fatal("首次 offer 应成功")
	}
	if p.offer(&fakeClient{id: 2}) {
		t.Error("池已满时 offer 应返回 false")
	}
}

func TestPoolOfferWhenClosed(t *testing.T) {
	p := newPool(1)
	_ = p.close()
	if p.offer(&fakeClient{id: 1}) {
		t.Error("池已关闭时 offer 应返回 false")
	}
}

func TestPoolBorrowContextCancel(t *testing.T) {
	p := newPool(1) // 空池，无连接可借
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	if _, err := p.borrow(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("borrow 应返回 context.DeadlineExceeded，实际: %v", err)
	}
}

func TestPoolCloseUnblocksBorrow(t *testing.T) {
	p := newPool(1)
	errc := make(chan error, 1)
	go func() {
		_, err := p.borrow(context.Background())
		errc <- err
	}()

	time.Sleep(20 * time.Millisecond)
	_ = p.close()

	select {
	case err := <-errc:
		if !errors.Is(err, ErrNotConnected) {
			t.Errorf("池关闭后 borrow 应返回 ErrNotConnected，实际: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("关闭池后 borrow 未被唤醒")
	}
}

func TestPoolReleaseAfterClose(t *testing.T) {
	p := newPool(1)
	c := &fakeClient{id: 1}
	p.offer(c)

	got, err := p.borrow(context.Background())
	if err != nil {
		t.Fatalf("borrow: %v", err)
	}
	_ = p.close()
	p.release(got)

	if n := c.closed.Load(); n != 1 {
		t.Errorf("关闭后归还的连接应被关闭，Close 调用次数 = %d, want 1", n)
	}
}

func TestPoolBorrowReleaseConcurrent(t *testing.T) {
	const (
		size       = 3
		goroutines = 10
		loops      = 50
	)
	p := newPool(size)
	clients := make([]*fakeClient, size)
	for i := range clients {
		clients[i] = &fakeClient{id: i}
		p.offer(clients[i])
	}

	ctx := context.Background()
	var inUse, maxInUse atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				c, err := p.borrow(ctx)
				if err != nil {
					t.Errorf("borrow: %v", err)
					return
				}
				n := inUse.Add(1)
				for {
					m := maxInUse.Load()
					if n <= m || maxInUse.CompareAndSwap(m, n) {
						break
					}
				}
				time.Sleep(time.Millisecond)
				inUse.Add(-1)
				p.release(c)
			}
		}()
	}
	wg.Wait()

	if got := maxInUse.Load(); got > size {
		t.Errorf("并发占用峰值 %d 超过池容量 %d", got, size)
	}
	if err := p.close(); err != nil {
		t.Errorf("close: %v", err)
	}

	closed := 0
	for _, c := range clients {
		if n := c.closed.Load(); n != 1 {
			t.Errorf("连接 %d 的 Close 调用次数 = %d, want 1", c.id, n)
		}
		closed += int(c.closed.Load())
	}
	if closed != size {
		t.Errorf("关闭的连接数 = %d, want %d", closed, size)
	}
}

// 容量 1 的池等价于单连接串行：RTU 串口这类不可池化链路依赖该语义。
func TestPoolSingleSerializes(t *testing.T) {
	p := newPool(1)
	p.offer(&fakeClient{id: 1})

	var inUse, maxInUse atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				c, err := p.borrow(context.Background())
				if err != nil {
					t.Errorf("borrow: %v", err)
					return
				}
				n := inUse.Add(1)
				for {
					m := maxInUse.Load()
					if n <= m || maxInUse.CompareAndSwap(m, n) {
						break
					}
				}
				time.Sleep(200 * time.Microsecond)
				inUse.Add(-1)
				p.release(c)
			}
		}()
	}
	wg.Wait()

	if got := maxInUse.Load(); got != 1 {
		t.Errorf("容量 1 的池并发占用峰值 = %d, want 1", got)
	}
	_ = p.close()
}
