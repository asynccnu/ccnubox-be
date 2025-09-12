package crawler

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCrawler_GetClassInfosForUndergraduate(t *testing.T) {
	var cookie = "JSESSIONID=98355539BF868E9B0675D58EE1D794A8"
	crawler := NewClassCrawler()
	start := time.Now()
	infos, scs, err := crawler.GetClassInfosForUndergraduate(context.Background(), "testID", "2024", "2", cookie)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(fmt.Sprintf("一共耗时%v", time.Since(start)))

	for _, v := range infos {
		t.Log(*v)
	}
	for _, v := range scs {
		t.Log(*v)
	}
	//t.Log(infos, scs)
}

func BenchmarkCrawler_GetClassInfosForUndergraduate(b *testing.B) {
	var cookie = "JSESSIONID=98355539BF868E9B0675D58EE1D794A8"
	crawler := NewClassCrawler()

	ctx := context.Background()

	// 通常第一次调用可以预热缓存等，不纳入统计
	_, _, _ = crawler.GetClassInfosForUndergraduate(ctx, "testID", "2024", "2", cookie)

	b.ResetTimer() // 重置计时器，排除预热时间
	for i := 0; i < b.N; i++ {
		_, _, err := crawler.GetClassInfosForUndergraduate(ctx, "testID", "2024", "2", cookie)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestCrawler_GetClassInfoForGraduateStudent(t *testing.T) {
	var cookie = "JSESSIONID=9BF9BFAD7E543259A65596CA5DFF4E60;route=f06bbbc827e6ce0f67fc73327c06186a"
	crawler := NewClassCrawler()
	start := time.Now()
	infos, scs, err := crawler.GetClassInfoForGraduateStudent(context.Background(), "testID", "2024", "1", cookie)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(fmt.Sprintf("一共耗时%v", time.Since(start)))

	for _, v := range infos {
		t.Log(*v)
	}
	for _, v := range scs {
		t.Log(*v)
	}
}
