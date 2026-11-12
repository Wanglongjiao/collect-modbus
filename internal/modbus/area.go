package modbus

import (
	"encoding/json"
	"fmt"
)

// Area Modbus 数据区，与配置中的 area 字符串一一对应。
type Area int

const (
	AreaCoil            Area = iota // coil 线圈
	AreaDiscreteInput               // discrete_input 离散输入
	AreaHoldingRegister             // holding_register 保持寄存器
	AreaInputRegister               // input_register 输入寄存器
)

var areaNames = map[Area]string{
	AreaCoil:            "coil",
	AreaDiscreteInput:   "discrete_input",
	AreaHoldingRegister: "holding_register",
	AreaInputRegister:   "input_register",
}

// areaFuncCodes 仅包含读路径功能码：01 读线圈、02 读离散输入、03 读保持寄存器、04 读输入寄存器。
// 写功能码（05/06/0F/10 等）v1 不实现，待写操作接入时再扩展。
var areaFuncCodes = map[Area]byte{
	AreaCoil:            0x01,
	AreaDiscreteInput:   0x02,
	AreaHoldingRegister: 0x03,
	AreaInputRegister:   0x04,
}

// AreaFrom 将配置字符串解析为 Area，非法值返回错误。
func AreaFrom(s string) (Area, error) {
	for area, name := range areaNames {
		if name == s {
			return area, nil
		}
	}
	return 0, fmt.Errorf("invalid area %q, want coil|discrete_input|holding_register|input_register", s)
}

// String 返回配置中使用的 area 字符串。
func (a Area) String() string {
	if name, ok := areaNames[a]; ok {
		return name
	}
	return fmt.Sprintf("Area(%d)", int(a))
}

// 单次读请求的协议上限。
const (
	MaxReadRegisters = 125  // 03/04 功能码单次最多读 125 个寄存器
	MaxReadCoils     = 2000 // 01/02 功能码单次最多读 2000 个线圈
)

// FuncCode 返回该 area 对应的读功能码（01/02/03/04），非法 area 返回 0。
func (a Area) FuncCode() byte {
	return areaFuncCodes[a]
}

// MaxReadCount 返回该 area 单次读取的数量上限，非法 area 返回 0。
func (a Area) MaxReadCount() int {
	switch a {
	case AreaCoil, AreaDiscreteInput:
		return MaxReadCoils
	case AreaHoldingRegister, AreaInputRegister:
		return MaxReadRegisters
	default:
		return 0
	}
}

// MarshalJSON 以配置字符串形式序列化，保证上行文件可读。
func (a Area) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON 解析配置字符串形式的 area。
func (a *Area) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	area, err := AreaFrom(s)
	if err != nil {
		return err
	}
	*a = area
	return nil
}
