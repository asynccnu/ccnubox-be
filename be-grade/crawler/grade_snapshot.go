package crawler

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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

// 没有可靠总数时不能证明完整性，禁止据短页猜测完整快照并建立首次基线。
// total/totalCount 字段及跨学期查询参数仍需上线前用脱敏真实响应核验。
func collectGradePages[T any](ctx context.Context, fetch func(int) ([]T, *int64, error), identity func(T) string) ([]T, error) {
	var result []T
	seen := map[string]bool{}
	seenPages := map[[32]byte]bool{}
	var total int64 = -1
	for page := 1; page <= 1000; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		items, count, err := fetch(page)
		if err != nil {
			return nil, err
		}
		if count == nil || *count < 0 || (total >= 0 && total != *count) {
			return nil, fmt.Errorf("crawler: grade snapshot total missing or changed")
		}
		total = *count
		encoded, err := json.Marshal(items)
		if err != nil {
			return nil, err
		}
		pageHash := sha256.Sum256(encoded)
		if len(items) > 0 && seenPages[pageHash] {
			return nil, fmt.Errorf("crawler: repeated grade page")
		}
		seenPages[pageHash] = true
		for _, item := range items {
			key := identity(item)
			if key != "" && seen[key] {
				return nil, fmt.Errorf("crawler: ambiguous grade identity or repeated page")
			}
			// 缺少身份的成绩交给同步事务隔离，不能拿课程名当作稳定 key。
			if key != "" {
				seen[key] = true
			}
			result = append(result, item)
		}
		if int64(len(result)) == total {
			return result, nil
		}
		if len(items) == 0 || int64(len(result)) > total {
			return nil, fmt.Errorf("crawler: grade pagination incomplete or inconsistent")
		}
	}
	return nil, fmt.Errorf("crawler: grade pagination limit exceeded")
}
