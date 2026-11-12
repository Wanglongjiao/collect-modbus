package modbus

import (
	"errors"
	"testing"
)

func TestAreaMaxReadCount(t *testing.T) {
	cases := []struct {
		area Area
		want int
	}{
		{AreaCoil, 2000},
		{AreaDiscreteInput, 2000},
		{AreaHoldingRegister, 125},
		{AreaInputRegister, 125},
		{Area(99), 0},
	}
	for _, c := range cases {
		if got := c.area.MaxReadCount(); got != c.want {
			t.Errorf("%v.MaxReadCount() = %d, want %d", c.area, got, c.want)
		}
	}
}

func TestValidateRead(t *testing.T) {
	cases := []struct {
		name    string
		area    Area
		count   int
		wantErr bool
		wantLim bool // 是否应为 ErrProtocolLimit
	}{
		{"寄存器上限内", AreaHoldingRegister, 125, false, false},
		{"寄存器超 1 个", AreaHoldingRegister, 126, true, true},
		{"输入寄存器上限内", AreaInputRegister, 125, false, false},
		{"输入寄存器超 1 个", AreaInputRegister, 126, true, true},
		{"线圈上限内", AreaCoil, 2000, false, false},
		{"线圈超 1 个", AreaCoil, 2001, true, true},
		{"离散输入上限内", AreaDiscreteInput, 2000, false, false},
		{"离散输入超 1 个", AreaDiscreteInput, 2001, true, true},
		{"单寄存器", AreaHoldingRegister, 1, false, false},
		{"数量为 0", AreaHoldingRegister, 0, true, false},
		{"数量为负", AreaCoil, -1, true, false},
		{"非法 area", Area(99), 1, true, false},
	}
	for _, c := range cases {
		err := validateRead(c.area, c.count)
		if !c.wantErr {
			if err != nil {
				t.Errorf("%s: validateRead(%v, %d) 意外报错: %v", c.name, c.area, c.count, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("%s: validateRead(%v, %d) 期望报错，实际为 nil", c.name, c.area, c.count)
			continue
		}
		if got := errors.Is(err, ErrProtocolLimit); got != c.wantLim {
			t.Errorf("%s: errors.Is(err, ErrProtocolLimit) = %v, want %v (err=%v)", c.name, got, c.wantLim, err)
		}
	}
}
