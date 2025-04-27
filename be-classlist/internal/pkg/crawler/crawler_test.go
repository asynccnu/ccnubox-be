package crawler

import (
	"context"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"github.com/asynccnu/ccnubox-be/be-classlist/test"
	"testing"
	"time"
)

var cookie = "JSESSIONID=9923654784AEF41BD751198E64AD830B"

func TestCrawler_GetClassInfosForUndergraduate(t *testing.T) {
	crawler := NewClassCrawler(test.NewLogger())
	start := time.Now()
	res, err := crawler.GetClassInfosForUndergraduate(context.Background(), model.GetClassInfosForUndergraduateReq{
		StuID:    "testID",
		Year:     "2024",
		Semester: "2",
		Cookie:   cookie,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(fmt.Sprintf("一共耗时%v", time.Since(start)))
	t.Log(res)
}

func BenchmarkCrawler_GetClassInfosForUndergraduate(b *testing.B) {
	crawler := NewClassCrawler(test.NewLogger())
	req := model.GetClassInfosForUndergraduateReq{
		StuID:    "testID",
		Year:     "2024",
		Semester: "2",
		Cookie:   cookie,
	}

	ctx := context.Background()

	// 通常第一次调用可以预热缓存等，不纳入统计
	_, _ = crawler.GetClassInfosForUndergraduate(ctx, req)

	b.ResetTimer() // 重置计时器，排除预热时间
	for i := 0; i < b.N; i++ {
		_, err := crawler.GetClassInfosForUndergraduate(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
