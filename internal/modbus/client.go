package modbus

import "context"

// Connection 抽象 Modbus 传输，多 device 可共享同一底层连接池。
// 三种传输（tcp / rtu / rtu_over_tcp）各提供一个实现。
type Connection interface {
	// Read 组 Modbus 请求帧并解析响应，返回原始字节（按 tag 的 data_type 决定如何切片）。
	// unitID 为设备从站地址，连接池可被多个 device 共享。
	Read(ctx context.Context, unitID uint8, area Area, addr, count int) ([]byte, error)

	// Write 写操作，v1 不调用，接口预留。
	Write(ctx context.Context, unitID uint8, area Area, addr int, data []byte) error

	// IsConnected 查询连接池健康状态；service 据此决定 device 是否进入 offline。
	IsConnected() bool

	// Close 释放连接池内所有物理连接。
	Close() error
}
