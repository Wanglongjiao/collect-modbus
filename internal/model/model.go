package model

import (
	"encoding/json"
	"fmt"
	"time"

	"collect-modbus/internal/modbus"
)

// Quality 点位质量。
type Quality int

const (
	QualityGood Quality = iota // 采集成功
	QualityBad                 // 采集失败
)

// String 返回小写质量标识。
func (q Quality) String() string {
	switch q {
	case QualityGood:
		return "good"
	case QualityBad:
		return "bad"
	default:
		return fmt.Sprintf("Quality(%d)", int(q))
	}
}

// MarshalJSON 以小写字符串形式序列化。
func (q Quality) MarshalJSON() ([]byte, error) {
	return json.Marshal(q.String())
}

// Bit 寄存器位定义，对应 config.Bit。
type Bit struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Index int    `json:"index"` // 0~15
}

// Tag 精简点位定义，供 ReadPlan 引用，字段与 config.Tag 一一对应。
type Tag struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Area      modbus.Area `json:"area"`
	Address   int         `json:"address"`
	DataType  string      `json:"data_type"`
	Endian    string      `json:"endian"`
	Scale     float64     `json:"scale"`
	Offset    float64     `json:"offset"`
	Precision int         `json:"precision"`
	Deadband  float64     `json:"deadband"`
	Unit      string      `json:"unit"`
	Bits      []Bit       `json:"bits,omitempty"`
}

// ReadPlan 一段连续地址的读计划，覆盖该区间内的所有 tags。
type ReadPlan struct {
	Area      modbus.Area `json:"area"`
	StartAddr int         `json:"start_addr"`
	Count     int         `json:"count"` // 要读的寄存器/线圈数量
	Tags      []Tag       `json:"tags"`
}

// TagValue 单点位采集结果，可直接序列化后上行。
type TagValue struct {
	DeviceID  string      `json:"device_id"`
	TagID     string      `json:"tag_id"`
	TagName   string      `json:"tag_name"`
	Area      modbus.Area `json:"area"`
	Address   int         `json:"address"`
	Timestamp time.Time   `json:"timestamp"`
	Raw       []byte      `json:"raw,omitempty"`
	Value     any         `json:"value"`
	Quality   Quality     `json:"quality"`
	Unit      string      `json:"unit,omitempty"`
	TraceID   string      `json:"trace_id,omitempty"`
}

// DeviceStatus 设备在线状态：连接断开时随 connection 级联离线；
// 连接正常但设备连续采集失败次数达到 offline_after_failures 时，该设备单独离线。
type DeviceStatus struct {
	DeviceID            string    `json:"device_id"`
	Online              bool      `json:"online"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastSuccessAt       time.Time `json:"last_success_at"`
	LastChangeAt        time.Time `json:"last_change_at"`
}
