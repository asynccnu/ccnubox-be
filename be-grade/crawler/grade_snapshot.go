package crawler

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseTotalScore 不把缺失、文本状态或溢出回退为有效零分，也不推测等级制映射。
func ParseTotalScore(value string) (float32, error) {
	value = strings.TrimSpace(value)
	number, err := strconv.ParseFloat(value, 32)
	if value == "" || err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("crawler: invalid or unpublished total score")
	}
	return float32(number), nil
}
