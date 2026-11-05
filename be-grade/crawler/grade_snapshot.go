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

type GradeSnapshot[T any] struct {
	Grades   []T
	Complete bool
}

// 没有总数的旧响应仍可用于查询，但不能据短页猜测完整性、建立首次通知基线。
func collectGradePages[T any](ctx context.Context, fetch func(int) ([]T, *int64, error), identity func(T) string) (GradeSnapshot[T], error) {
	var result []T
	seen := map[string]bool{}
	seenPages := map[[32]byte]bool{}
	var total int64 = -1
	for page := 1; page <= 1000; page++ {
		if err := ctx.Err(); err != nil {
			return GradeSnapshot[T]{}, err
		}
		items, count, err := fetch(page)
		if err != nil {
			return GradeSnapshot[T]{}, err
		}
		if count == nil {
			if page == 1 {
				return GradeSnapshot[T]{Grades: items}, nil
			}
			return GradeSnapshot[T]{}, fmt.Errorf("crawler: grade snapshot total missing or changed")
		}
		if *count < 0 || (total >= 0 && total != *count) {
			return GradeSnapshot[T]{}, fmt.Errorf("crawler: grade snapshot total missing or changed")
		}
		total = *count
		encoded, err := json.Marshal(items)
		if err != nil {
			return GradeSnapshot[T]{}, err
		}
		pageHash := sha256.Sum256(encoded)
		if len(items) > 0 && seenPages[pageHash] {
			return GradeSnapshot[T]{}, fmt.Errorf("crawler: repeated grade page")
		}
		seenPages[pageHash] = true
		for _, item := range items {
			key := identity(item)
			if key != "" && seen[key] {
				return GradeSnapshot[T]{}, fmt.Errorf("crawler: ambiguous grade identity or repeated page")
			}
			// 缺少身份的成绩交给同步事务隔离，不能拿课程名当作稳定 key。
			if key != "" {
				seen[key] = true
			}
			result = append(result, item)
		}
		if int64(len(result)) == total {
			return GradeSnapshot[T]{Grades: result, Complete: true}, nil
		}
		if len(items) == 0 || int64(len(result)) > total {
			return GradeSnapshot[T]{}, fmt.Errorf("crawler: grade pagination incomplete or inconsistent")
		}
	}
	return GradeSnapshot[T]{}, fmt.Errorf("crawler: grade pagination limit exceeded")
}
