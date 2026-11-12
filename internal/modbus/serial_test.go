package modbus

import (
	"testing"

	smodbus "github.com/simonvetter/modbus"
)

func TestParseSerial(t *testing.T) {
	cases := []struct {
		in   string
		want SerialConfig
	}{
		{"9600|8N1", SerialConfig{Baud: 9600, DataBits: 8, Parity: smodbus.PARITY_NONE, StopBits: 1}},
		{"19200|7E2", SerialConfig{Baud: 19200, DataBits: 7, Parity: smodbus.PARITY_EVEN, StopBits: 2}},
		{"115200|8O1", SerialConfig{Baud: 115200, DataBits: 8, Parity: smodbus.PARITY_ODD, StopBits: 1}},
		{"1200|5N2", SerialConfig{Baud: 1200, DataBits: 5, Parity: smodbus.PARITY_NONE, StopBits: 2}},
	}
	for _, c := range cases {
		got, err := ParseSerial(c.in)
		if err != nil {
			t.Errorf("ParseSerial(%q) 意外报错: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseSerial(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestParseSerialInvalid(t *testing.T) {
	cases := []string{
		"",               // 空串
		"9600",           // 缺少 |
		"9600|8N1|extra", // 多余分隔
		"|8N1",           // 缺波特率
		"abc|8N1",        // 波特率非数字
		"0|8N1",          // 波特率为 0
		"-9600|8N1",      // 波特率为负
		"9600|8N",        // 帧格式不足 3 位
		"9600|8N11",      // 帧格式超过 3 位
		"9600|9N1",       // 数据位越界
		"9600|4N1",       // 数据位越界
		"9600|8X1",       // 校验位非法
		"9600|8n1",       // 校验位大小写敏感
		"9600|8N3",       // 停止位越界
		"9600|8N0",       // 停止位越界
	}
	for _, in := range cases {
		if _, err := ParseSerial(in); err == nil {
			t.Errorf("ParseSerial(%q) 期望报错，实际成功", in)
		}
	}
}
