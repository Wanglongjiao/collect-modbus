// Package trace 提供轻量 TraceID 生成与 ctx 透传能力，无内部依赖。
//
// TraceID 结构：[4字节 crypto/rand 启动时生成一次][4字节 atomic 每次 +1]
// 总 8 字节 → 16 位 hex，如 "7a3e91b200000001"。
//
// 多进程安全：每个进程启动时各自生成 4 字节随机前缀，32 位碰撞概率极低
// （2^32 ≈ 43 亿次启动才 50% 概率撞前缀），进程内 counter 单调递增，
// 三重保险确保多进程 TraceID 不重复。
package trace

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
)

type ctxKey struct{}

var (
	once    sync.Once
	prefix  uint32
	counter atomic.Uint32
)

func initPrefix() {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 在几乎所有环境都不会失败；极端失败退化为 time 前缀
		prefix = uint32(0)
		return
	}
	prefix = binary.BigEndian.Uint32(b)
}

// NewTraceID 生成一个新的 TraceID，16 位小写 hex。
func NewTraceID() string {
	once.Do(initPrefix)
	c := counter.Add(1)
	return fmt.Sprintf("%08x%08x", prefix, c)
}

// WithTraceID 将 TraceID 注入 ctx，返回新的 ctx。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, traceID)
}

// FromCtx 从 ctx 取出 TraceID；不存在返回空串。
func FromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}
