package modbus

import (
	"context"
	"sync"

	smodbus "github.com/simonvetter/modbus"
)

// rawClient 是连接池管理的物理连接，*smodbus.ModbusClient 实现该接口。
// 抽成接口是为了在无设备环境下用假连接做单元测试。
type rawClient interface {
	ReadCoils(addr, quantity uint16) ([]bool, error)
	ReadDiscreteInputs(addr, quantity uint16) ([]bool, error)
	ReadRegisters(addr, quantity uint16, regType smodbus.RegType) ([]uint16, error)
	SetUnitId(id uint8) error
	Close() error
}

// pool 固定容量的连接池：借出独占、归还阻塞式入池。
// 借出的连接同一时刻只被一个调用方使用，天然串行化同一物理链路上的请求。
type pool struct {
	size      int
	conns     chan rawClient
	closed    chan struct{}
	closeOnce sync.Once
}

func newPool(size int) *pool {
	if size < 1 {
		size = 1
	}
	return &pool{
		size:   size,
		conns:  make(chan rawClient, size),
		closed: make(chan struct{}),
	}
}

// offer 把已建连的物理连接注入池；池已关闭或已满时返回 false，由调用方负责关闭。
func (p *pool) offer(c rawClient) bool {
	select {
	case <-p.closed:
		return false
	default:
	}

	select {
	case <-p.closed:
		return false
	case p.conns <- c:
		return true
	default:
		return false
	}
}

// borrow 借出一条连接；无可用连接时阻塞，直到有连接归还、ctx 取消或池关闭。
func (p *pool) borrow(ctx context.Context) (rawClient, error) {
	select {
	case <-p.closed:
		return nil, ErrNotConnected
	default:
	}

	select {
	case c := <-p.conns:
		return c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.closed:
		return nil, ErrNotConnected
	}
}

// release 归还连接；池已关闭时直接关闭该物理连接，避免泄漏。
func (p *pool) release(c rawClient) {
	select {
	case <-p.closed:
		_ = c.Close()
		return
	default:
	}

	select {
	case p.conns <- c:
	default:
		// 借出数量不可能超过容量，兜底防止连接泄漏
		_ = c.Close()
	}
}

// close 关闭池并释放仍在池内的物理连接；已借出的连接在归还时关闭。
func (p *pool) close() error {
	p.closeOnce.Do(func() {
		close(p.closed)
		for {
			select {
			case c := <-p.conns:
				_ = c.Close()
			default:
				return
			}
		}
	})
	return nil
}

func (p *pool) isClosed() bool {
	select {
	case <-p.closed:
		return true
	default:
		return false
	}
}
