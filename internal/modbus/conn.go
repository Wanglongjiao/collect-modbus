package modbus

import (
	"context"
	"encoding/binary"
	"fmt"

	smodbus "github.com/simonvetter/modbus"
)

// conn 是各传输共用的请求执行层。
// 池化传输（tcp / rtu_over_tcp）按 pool_size 借还物理连接；
// 非池化传输（rtu 串口天然单链路）用容量 1 的池，借还退化为互斥串行，
// 保证同一时刻只有一个请求在串口上收发。
type conn struct {
	addr string
	pool *pool
}

// newConn 建 size 条连接注入池；任一条失败即关闭已建连接并返回错误。
func newConn(addr string, size int, dial func() (rawClient, error)) (*conn, error) {
	p := newPool(size)
	for i := 0; i < p.size; i++ {
		c, err := dial()
		if err != nil {
			_ = p.close()
			return nil, fmt.Errorf("modbus: dial %s: %w", addr, err)
		}
		p.offer(c)
	}
	return &conn{addr: addr, pool: p}, nil
}

// Read 借出一条连接完成一次读请求，返回原始字节：
// 寄存器按大端展开为 2 字节/个；线圈与离散输入展开为 1 字节/点（0x00/0x01）。
// unitID 即设备从站地址，同一连接池可服务不同从站。
func (c *conn) Read(ctx context.Context, unitID uint8, area Area, addr, count int) ([]byte, error) {
	if err := validateRead(area, count); err != nil {
		return nil, err
	}
	rc, err := c.pool.borrow(ctx)
	if err != nil {
		return nil, err
	}
	defer c.pool.release(rc)

	if err := rc.SetUnitId(unitID); err != nil {
		return nil, err
	}

	switch area {
	case AreaHoldingRegister, AreaInputRegister:
		regType := smodbus.HOLDING_REGISTER
		if area == AreaInputRegister {
			regType = smodbus.INPUT_REGISTER
		}
		regs, err := rc.ReadRegisters(uint16(addr), uint16(count), regType)
		if err != nil {
			return nil, err
		}
		return registersToBytes(regs), nil
	case AreaCoil, AreaDiscreteInput:
		var bits []bool
		if area == AreaCoil {
			bits, err = rc.ReadCoils(uint16(addr), uint16(count))
		} else {
			bits, err = rc.ReadDiscreteInputs(uint16(addr), uint16(count))
		}
		if err != nil {
			return nil, err
		}
		return boolsToBytes(bits), nil
	default:
		return nil, fmt.Errorf("modbus: invalid area %v", area)
	}
}

// Write v1 不实现，返回 ErrNotImplemented。
func (c *conn) Write(_ context.Context, _ uint8, _ Area, _ int, _ []byte) error {
	return ErrNotImplemented
}

// IsConnected 池未关闭即视为连接可用。
func (c *conn) IsConnected() bool {
	return !c.pool.isClosed()
}

// Close 关闭池内所有物理连接。
func (c *conn) Close() error {
	return c.pool.close()
}

func registersToBytes(regs []uint16) []byte {
	out := make([]byte, len(regs)*2)
	for i, r := range regs {
		binary.BigEndian.PutUint16(out[i*2:], r)
	}
	return out
}

func boolsToBytes(bits []bool) []byte {
	out := make([]byte, len(bits))
	for i, b := range bits {
		if b {
			out[i] = 1
		}
	}
	return out
}
