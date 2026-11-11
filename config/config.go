package config

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Config 对应 config.toml 的顶层结构
type Config struct {
	Version     int          `mapstructure:"version"`
	Defaults    Defaults     `mapstructure:"defaults"`
	Log         Log          `mapstructure:"log"`
	Connections []Connection `mapstructure:"connections"`
	Devices     []Device     `mapstructure:"devices"`
}

// Defaults 全局默认采集参数
type Defaults struct {
	PollInterval         int `mapstructure:"poll_interval"`          // 采集周期，毫秒
	OfflineAfterFailures int `mapstructure:"offline_after_failures"` // 失败次数超过阈值后标记离线
	MaxGap               int `mapstructure:"max_gap"`                // 连续寄存器合并读的最大空洞，超过则拆请求
}

// Log 日志配置
type Log struct {
	Level      string `mapstructure:"level"` // debug|info|warn|error
	Console    bool   `mapstructure:"console"`
	Path       string `mapstructure:"path"`
	MaxSize    int    `mapstructure:"max_size"` // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // 天
}

// Connection 传输连接配置，id 不含传输方式
type Connection struct {
	ID         string `mapstructure:"id"`
	Transport  string `mapstructure:"transport"`   // tcp|rtu|rtu_over_tcp
	Host       string `mapstructure:"host"`        // tcp / rtu_over_tcp 使用
	Port       int    `mapstructure:"port"`        // tcp / rtu_over_tcp 使用
	SerialPort string `mapstructure:"serial_port"` // rtu 使用，如 COM3 / /dev/ttyUSB0
	Serial     string `mapstructure:"serial"`      // 波特率|数据位+校验位+停止位，如 9600|8N1
	Timeout    int    `mapstructure:"timeout"`     // 毫秒
}

// Device 采集设备
type Device struct {
	ID            string `mapstructure:"id"`
	Name          string `mapstructure:"name"`
	Connection    string `mapstructure:"connection"` // 引用 Connection.ID
	DeviceAddress int    `mapstructure:"device_address"`
	Endian        string `mapstructure:"endian"` // 32/64 位字节+字序：ABCD|CDAB|BADC|DCBA，tag 可覆盖
	Tags          []Tag  `mapstructure:"tags"`
}

// Tag 采集点位
type Tag struct {
	ID         string  `mapstructure:"id"`
	Name       string  `mapstructure:"name"`
	Area       string  `mapstructure:"area"` // coil|discrete_input|holding_register|input_register
	Address    int     `mapstructure:"address"`
	DataType   string  `mapstructure:"data_type"`
	Permission string  `mapstructure:"permission"` // r|rw
	Endian     string  `mapstructure:"endian"`     // 为空时继承 Device.Endian
	Scale      float64 `mapstructure:"scale"`      // 最终值 = raw*Scale + Offset，未配置时默认为 1
	Offset     float64 `mapstructure:"offset"`
	Precision  int     `mapstructure:"precision"`
	Deadband   float64 `mapstructure:"deadband"` // 变化超过阈值才上报
	Unit       string  `mapstructure:"unit"`
	Bits       []Bit   `mapstructure:"bits"` // 位拆分定义
}

// Bit 寄存器位定义
type Bit struct {
	ID    string `mapstructure:"id"`
	Name  string `mapstructure:"name"`
	Index int    `mapstructure:"index"` // 0~15
}

var (
	global  *Config
	once    sync.Once
	loadErr error
)

// Load 解析 path 指定的 TOML 配置文件并初始化全局配置单例。
// 并发安全，仅首次调用真正加载，后续调用返回首次的结果。
func Load(path string) (*Config, error) {
	once.Do(func() {
		v := viper.New()
		v.SetConfigFile(path)
		v.SetConfigType("toml")

		if err := v.ReadInConfig(); err != nil {
			loadErr = fmt.Errorf("load config %q: %w", path, err)
			return
		}
		cfg := &Config{}
		if err := v.Unmarshal(cfg); err != nil {
			loadErr = fmt.Errorf("unmarshal config %q: %w", path, err)
			return
		}
		cfg.setDefaults()
		if err := cfg.validate(); err != nil {
			loadErr = fmt.Errorf("invalid config %q: %w", path, err)
			return
		}
		global = cfg
	})
	return global, loadErr
}

// Get 返回全局配置单例，需先调用 Load；未加载时返回 nil。
func Get() *Config {
	return global
}

// setDefaults 填充解析后的默认值，0 值/空字符串视为未配置
func (c *Config) setDefaults() {
	d := &c.Defaults
	if d.PollInterval == 0 {
		d.PollInterval = 10000
	}
	if d.OfflineAfterFailures == 0 {
		d.OfflineAfterFailures = 3
	}
	if d.MaxGap == 0 {
		d.MaxGap = 8
	}

	l := &c.Log
	if l.Level == "" {
		l.Level = "info"
	}
	if l.Path == "" {
		l.Path = "./log/modbus.log"
	}
	if l.MaxSize == 0 {
		l.MaxSize = 50
	}
	if l.MaxBackups == 0 {
		l.MaxBackups = 10
	}
	if l.MaxAge == 0 {
		l.MaxAge = 30
	}

	for i := range c.Connections {
		conn := &c.Connections[i]
		if conn.Transport == "" {
			conn.Transport = "tcp"
		}
		if conn.Timeout == 0 {
			conn.Timeout = 3000
		}
		switch conn.Transport {
		case "tcp", "rtu_over_tcp":
			if conn.Port == 0 {
				conn.Port = 502
			}
		case "rtu":
			if conn.Serial == "" {
				conn.Serial = "9600|8N1"
			}
		}
	}

	for i := range c.Devices {
		dev := &c.Devices[i]
		if dev.Endian == "" {
			dev.Endian = "ABCD"
		}
		for j := range dev.Tags {
			t := &dev.Tags[j]
			if t.Permission == "" {
				t.Permission = "r"
			}
			if t.Endian == "" {
				t.Endian = dev.Endian // 继承设备字节序
			}
			if t.Scale == 0 {
				t.Scale = 1
			}
		}
	}
}

// validate 校验配置合法性，收集所有错误并通过 errors.Join 一次返回（错误信息以换行分隔）
func (c *Config) validate() error {
	var errs []error
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	// if c.Version <= 0 {
	// 	add("version 必须为正整数")
	// }
	if c.Defaults.PollInterval <= 0 {
		add("defaults.poll_interval 必须大于 0")
	}
	if c.Defaults.OfflineAfterFailures <= 0 {
		add("defaults.offline_after_failures 必须大于 0")
	}
	if c.Defaults.MaxGap < 0 {
		add("defaults.max_gap 不能为负数")
	}
	if !oneOf(c.Log.Level, "debug", "info", "warn", "error") {
		add("log.level %q 非法，可选 debug|info|warn|error", c.Log.Level)
	}
	if c.Log.MaxSize <= 0 {
		add("log.max_size 必须大于 0")
	}
	if c.Log.MaxBackups < 0 || c.Log.MaxAge < 0 {
		add("log.max_backups / log.max_age 不能为负数")
	}

	connIDs := map[string]bool{}
	for i := range c.Connections {
		conn := &c.Connections[i]
		if conn.ID == "" {
			add("connections[%d]: id 不能为空", i)
		} else {
			if connIDs[conn.ID] {
				add("connection %q: id 重复", conn.ID)
			}
			connIDs[conn.ID] = true
		}
		if !oneOf(conn.Transport, "tcp", "rtu", "rtu_over_tcp") {
			add("connection %q: transport %q 非法，可选 tcp|rtu|rtu_over_tcp", conn.ID, conn.Transport)
		}
		if conn.Timeout <= 0 {
			add("connection %q: timeout 必须大于 0", conn.ID)
		}
		switch conn.Transport {
		case "tcp", "rtu_over_tcp":
			if conn.Host == "" {
				add("connection %q: host 不能为空", conn.ID)
			}
			if conn.Port <= 0 || conn.Port > 65535 {
				add("connection %q: port %d 非法，范围 1~65535", conn.ID, conn.Port)
			}
		case "rtu":
			if conn.SerialPort == "" {
				add("connection %q: serial_port 不能为空", conn.ID)
			}
			if err := validateSerial(conn.Serial); err != nil {
				errs = append(errs, fmt.Errorf("connection %q: %w", conn.ID, err))
			}
		}
	}

	devIDs := map[string]bool{}
	for i := range c.Devices {
		dev := &c.Devices[i]
		if dev.ID == "" {
			add("devices[%d]: id 不能为空", i)
		} else {
			if devIDs[dev.ID] {
				add("device %q: id 重复", dev.ID)
			}
			devIDs[dev.ID] = true
		}
		if !connIDs[dev.Connection] {
			add("device %q: connection %q 未定义", dev.ID, dev.Connection)
		}
		if dev.DeviceAddress < 1 || dev.DeviceAddress > 247 {
			add("device %q: device_address %d 非法，Modbus 站号范围 1~247", dev.ID, dev.DeviceAddress)
		}
		if !oneOf(dev.Endian, "ABCD", "CDAB", "BADC", "DCBA") {
			add("device %q: endian %q 非法，可选 ABCD|CDAB|BADC|DCBA", dev.ID, dev.Endian)
		}

		tagIDs := map[string]bool{}
		for j := range dev.Tags {
			t := &dev.Tags[j]
			if t.ID == "" {
				add("device %q: tags[%d]: id 不能为空", dev.ID, j)
			} else {
				if tagIDs[t.ID] {
					add("device %q tag %q: id 重复", dev.ID, t.ID)
				}
				tagIDs[t.ID] = true
			}
			if !oneOf(t.Area, "coil", "discrete_input", "holding_register", "input_register") {
				add("device %q tag %q: area %q 非法，可选 coil|discrete_input|holding_register|input_register", dev.ID, t.ID, t.Area)
			}
			if t.Address < 0 || t.Address > 65535 {
				add("device %q tag %q: address %d 非法，范围 0~65535", dev.ID, t.ID, t.Address)
			}
			dataTypeOK := oneOf(t.DataType, "bool", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float32", "float64")
			if !dataTypeOK {
				add("device %q tag %q: data_type %q 非法，可选 bool|int16|uint16|int32|uint32|int64|uint64|float32|float64", dev.ID, t.ID, t.DataType)
			}
			if dataTypeOK && (t.Area == "coil" || t.Area == "discrete_input") && t.DataType != "bool" {
				add("device %q tag %q: %s 只支持 data_type = bool", dev.ID, t.ID, t.Area)
			}
			if !oneOf(t.Permission, "r", "rw") {
				add("device %q tag %q: permission %q 非法，可选 r|rw", dev.ID, t.ID, t.Permission)
			}
			if !oneOf(t.Endian, "ABCD", "CDAB", "BADC", "DCBA") {
				add("device %q tag %q: endian %q 非法，可选 ABCD|CDAB|BADC|DCBA", dev.ID, t.ID, t.Endian)
			}
			if t.Precision < 0 {
				add("device %q tag %q: precision 不能为负数", dev.ID, t.ID)
			}
			if t.Deadband < 0 {
				add("device %q tag %q: deadband 不能为负数", dev.ID, t.ID)
			}
			if dataTypeOK && len(t.Bits) > 0 && !oneOf(t.DataType, "uint16", "int16") {
				add("device %q tag %q: bits 仅支持 uint16|int16 类型", dev.ID, t.ID)
			}

			bitIDs := map[string]bool{}
			for k := range t.Bits {
				b := &t.Bits[k]
				if b.ID == "" {
					add("device %q tag %q: bits[%d]: id 不能为空", dev.ID, t.ID, k)
				} else {
					if bitIDs[b.ID] {
						add("device %q tag %q: bit id %q 重复", dev.ID, t.ID, b.ID)
					}
					bitIDs[b.ID] = true
				}
				if b.Index < 0 || b.Index > 15 {
					add("device %q tag %q bit %q: index %d 非法，范围 0~15", dev.ID, t.ID, b.ID, b.Index)
				}
			}
		}
	}
	return errors.Join(errs...)
}

// serialPattern 串口参数格式：数据位 5~8 + 校验位 N/E/O + 停止位 1/2，如 8N1
var serialPattern = regexp.MustCompile(`^[5-8][NEO][12]$`)

// validateSerial 校验串口配置，格式为 波特率|数据位+校验位+停止位，如 9600|8N1
func validateSerial(s string) error {
	parts := strings.Split(s, "|")
	if len(parts) != 2 {
		return fmt.Errorf("serial %q 格式应为 波特率|数据位+校验位+停止位，如 9600|8N1", s)
	}
	if baud, err := strconv.Atoi(parts[0]); err != nil || baud <= 0 {
		return fmt.Errorf("serial %q 波特率非法", s)
	}
	if !serialPattern.MatchString(parts[1]) {
		return fmt.Errorf("serial %q 数据位/校验位/停止位非法，应为 [5-8][N|E|O][1|2]，如 8N1", s)
	}
	return nil
}

func oneOf(s string, vals ...string) bool {
	for _, v := range vals {
		if s == v {
			return true
		}
	}
	return false
}
