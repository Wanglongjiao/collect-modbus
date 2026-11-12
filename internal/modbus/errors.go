package modbus

import "errors"

// 传输层错误哨兵，调用方通过 errors.Is 判定错误类别。
var (
	// ErrNotImplemented 功能尚未实现（如 v1 的 Write）。
	ErrNotImplemented = errors.New("modbus: not implemented")
	// ErrProtocolLimit 请求数量超出协议上限（读寄存器 125 / 读线圈 2000）。
	ErrProtocolLimit = errors.New("modbus: request exceeds protocol limit")
	// ErrNotConnected 连接未建立或已断开。
	ErrNotConnected = errors.New("modbus: not connected")
)
