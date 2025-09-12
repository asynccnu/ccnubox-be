package crawler

import (
	"context"
	"io/ioutil"
	"testing"
)

func Test_extractCourseInfo(t *testing.T) {

	path := "./test.html"

	// 读取文件内容
	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	t.Logf("length: %v", len(content))

	c := NewClassCrawler2()

	classes, err := c.extractCourses(context.Background(), "2025", "1", string(content))
	if err != nil {
		t.Fatalf("failed to extract classes: %v", err)
	}
	for _, class := range classes {
		t.Log(class)
	}
}

func Test_Crawler2(t *testing.T) {
	c := NewClassCrawler2()
	test_cookie := "bzb_jsxsd=CD6739EC4A67BB85312FF0388BD82311"
	a, b, err := c.GetClassInfosForUndergraduate(context.Background(), "2023214414", "2025", "1", test_cookie)
	if err != nil {
		t.Fatalf("failed to crawl: %v", err)
	}
	for _, info := range a {
		t.Log(info)
	}
	for _, info := range b {
		t.Log(info)
	}

}
