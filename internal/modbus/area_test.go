package modbus

import (
	"encoding/json"
	"testing"
)

func TestAreaFrom(t *testing.T) {
	cases := []struct {
		in      string
		want    Area
		wantErr bool
	}{
		{"coil", AreaCoil, false},
		{"discrete_input", AreaDiscreteInput, false},
		{"holding_register", AreaHoldingRegister, false},
		{"input_register", AreaInputRegister, false},
		{"", 0, true},
		{"Holding_Register", 0, true},
		{"coils", 0, true},
	}
	for _, c := range cases {
		got, err := AreaFrom(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("AreaFrom(%q) 期望报错，实际返回 %v", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("AreaFrom(%q) 意外报错: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("AreaFrom(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestAreaRoundTrip(t *testing.T) {
	for _, area := range []Area{AreaCoil, AreaDiscreteInput, AreaHoldingRegister, AreaInputRegister} {
		got, err := AreaFrom(area.String())
		if err != nil {
			t.Errorf("AreaFrom(%q) 意外报错: %v", area.String(), err)
			continue
		}
		if got != area {
			t.Errorf("往返失败: %v → %q → %v", area, area.String(), got)
		}
	}
}

func TestAreaFuncCode(t *testing.T) {
	cases := []struct {
		area Area
		want byte
	}{
		{AreaCoil, 0x01},
		{AreaDiscreteInput, 0x02},
		{AreaHoldingRegister, 0x03},
		{AreaInputRegister, 0x04},
	}
	for _, c := range cases {
		if got := c.area.FuncCode(); got != c.want {
			t.Errorf("%v.FuncCode() = 0x%02X, want 0x%02X", c.area, got, c.want)
		}
	}
	if got := Area(99).FuncCode(); got != 0 {
		t.Errorf("非法 area FuncCode() = 0x%02X, want 0", got)
	}
}

func TestAreaJSONRoundTrip(t *testing.T) {
	for _, area := range []Area{AreaCoil, AreaDiscreteInput, AreaHoldingRegister, AreaInputRegister} {
		data, err := json.Marshal(area)
		if err != nil {
			t.Fatalf("marshal %v: %v", area, err)
		}
		if string(data) != `"`+area.String()+`"` {
			t.Errorf("marshal %v = %s, want %q", area, data, area.String())
		}
		var back Area
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", data, err)
		}
		if back != area {
			t.Errorf("JSON 往返失败: %v → %s → %v", area, data, back)
		}
	}
}
