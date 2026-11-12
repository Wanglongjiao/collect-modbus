package modbus

import (
	"net"
	"strconv"
	"time"

	smodbus "github.com/simonvetter/modbus"
)

// TCPConn 基于 TCP 的 Modbus 连接，按 pool_size 池化，多 device 共用。
type TCPConn struct {
	*conn
}

var _ Connection = (*TCPConn)(nil)

// NewTCPConn 建立 poolSize 条 TCP 连接并注入池。
// 任一条建连失败都会关闭已建连接并返回错误（fail-fast），
// 退避重连由 M2.11 在连接状态机中负责。
func NewTCPConn(host string, port int, timeout time.Duration, poolSize int) (*TCPConn, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	c, err := newConn(addr, poolSize, func() (rawClient, error) {
		return dialTCP(addr, timeout)
	})
	if err != nil {
		return nil, err
	}
	return &TCPConn{conn: c}, nil
}

func dialTCP(addr string, timeout time.Duration) (rawClient, error) {
	c, err := smodbus.NewClient(&smodbus.ClientConfiguration{
		URL:     "tcp://" + addr,
		Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	if err := c.Open(); err != nil {
		return nil, err
	}
	return c, nil
}
