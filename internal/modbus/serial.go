package modbus

import (
	"fmt"
	"strconv"
	"strings"

	smodbus "github.com/simonvetter/modbus"
)

// SerialConfig 串口参数，字段取值与 simonvetter/modbus 的 ClientConfiguration 对齐。
type SerialConfig struct {
	Baud     uint // 波特率
	DataBits uint // 数据位 5~8
	Parity   uint // 校验位：smodbus.PARITY_NONE / EVEN / ODD
	StopBits uint // 停止位 1~2
}

// ParseSerial 解析 "波特率|数据位+校验位+停止位"，如 "9600|8N1"。
// 规则与 config.validateSerial 保持一致：波特率 > 0，数据位 5~8，校验位 N/E/O，停止位 1/2。
func ParseSerial(s string) (SerialConfig, error) {
	parts := strings.Split(s, "|")
	if len(parts) != 2 {
		return SerialConfig{}, fmt.Errorf("modbus: serial %q 格式应为 波特率|数据位+校验位+停止位，如 9600|8N1", s)
	}

	baud, err := strconv.Atoi(parts[0])
	if err != nil || baud <= 0 {
		return SerialConfig{}, fmt.Errorf("modbus: serial %q 波特率非法", s)
	}

	frame := parts[1]
	if len(frame) != 3 {
		return SerialConfig{}, fmt.Errorf("modbus: serial %q 数据位/校验位/停止位应为 3 个字符，如 8N1", s)
	}

	cfg := SerialConfig{Baud: uint(baud)}

	if frame[0] < '5' || frame[0] > '8' {
		return SerialConfig{}, fmt.Errorf("modbus: serial %q 数据位非法，应为 5~8", s)
	}
	cfg.DataBits = uint(frame[0] - '0')

	switch frame[1] {
	case 'N':
		cfg.Parity = smodbus.PARITY_NONE
	case 'E':
		cfg.Parity = smodbus.PARITY_EVEN
	case 'O':
		cfg.Parity = smodbus.PARITY_ODD
	default:
		return SerialConfig{}, fmt.Errorf("modbus: serial %q 校验位非法，应为 N|E|O", s)
	}

	if frame[2] != '1' && frame[2] != '2' {
		return SerialConfig{}, fmt.Errorf("modbus: serial %q 停止位非法，应为 1|2", s)
	}
	cfg.StopBits = uint(frame[2] - '0')

	return cfg, nil
}
