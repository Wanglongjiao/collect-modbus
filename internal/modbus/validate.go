package modbus

import "fmt"

// validateRead 校验读请求的协议约束：area 合法、count 为正、且不超过该 area 的单次上限。
// 超限返回 ErrProtocolLimit，调用方可用 errors.Is 判定。
func validateRead(area Area, count int) error {
	limit := area.MaxReadCount()
	if limit == 0 {
		return fmt.Errorf("modbus: invalid area %v", area)
	}
	if count <= 0 {
		return fmt.Errorf("modbus: read count %d must be positive", count)
	}
	if count > limit {
		return fmt.Errorf("%w: area %s count %d > %d", ErrProtocolLimit, area, count, limit)
	}
	return nil
}
