package biz

import (
	"context"
	"testing"
)

type MockCookieClient struct {
}

func (m *MockCookieClient) GetCookie(ctx context.Context, stuID string, tpe ...string) (string, error) {
	return "JSESSIONID=ACB2FEEF93678BF837955F63E088D85B", nil
}

func TestFreeClassroomBiz_crawFreeClassroom(t *testing.T) {
	cli := new(MockCookieClient)
	fcb := NewFreeClassroomBiz(nil, nil, cli, nil, nil, nil)
	res, err := fcb.getFreeClassrooms(context.Background(), "2024", "2", "testID", 6, 2, []int{1, 2}, "71")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(res)
}
