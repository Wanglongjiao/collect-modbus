package model

import (
	"encoding/json"
	"testing"
	"time"

	"collect-modbus/internal/modbus"
)

func TestQualityString(t *testing.T) {
	if got := QualityGood.String(); got != "good" {
		t.Errorf("QualityGood.String() = %q, want %q", got, "good")
	}
	if got := QualityBad.String(); got != "bad" {
		t.Errorf("QualityBad.String() = %q, want %q", got, "bad")
	}
}

func TestTagValueJSON(t *testing.T) {
	ts := time.Date(2026, 11, 12, 10, 30, 0, 0, time.UTC)
	v := TagValue{
		DeviceID:  "dev-1",
		TagID:     "tag-1",
		TagName:   "入口温度",
		Area:      modbus.AreaHoldingRegister,
		Address:   100,
		Timestamp: ts,
		Raw:       []byte{0x41, 0x48, 0x00, 0x00},
		Value:     12.5,
		Quality:   QualityGood,
		Unit:      "℃",
		TraceID:   "0123456789abcdef",
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal TagValue: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal 为 map: %v", err)
	}
	if m["area"] != "holding_register" {
		t.Errorf("area = %v, want holding_register", m["area"])
	}
	if m["quality"] != "good" {
		t.Errorf("quality = %v, want good", m["quality"])
	}
	if m["timestamp"] != "2026-11-12T10:30:00Z" {
		t.Errorf("timestamp = %v, want 2026-11-12T10:30:00Z", m["timestamp"])
	}
	if m["device_id"] != "dev-1" || m["tag_id"] != "tag-1" || m["trace_id"] != "0123456789abcdef" {
		t.Errorf("标识字段序列化不符: %v", m)
	}
	if m["value"] != 12.5 {
		t.Errorf("value = %v, want 12.5", m["value"])
	}
}

func TestTagValueJSONOmitsEmpty(t *testing.T) {
	data, err := json.Marshal(TagValue{DeviceID: "d", TagID: "t"})
	if err != nil {
		t.Fatalf("marshal TagValue: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal 为 map: %v", err)
	}
	for _, key := range []string{"raw", "unit", "trace_id"} {
		if _, ok := m[key]; ok {
			t.Errorf("空值字段 %q 不应出现在 JSON 中: %s", key, data)
		}
	}
}
