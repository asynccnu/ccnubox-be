package jpush

import (
	"testing"
)

func TestPush(t *testing.T) {
	c := NewJPushClient(&JPushConfig{})
	for i := 0; i < 5; i++ {
		err := c.Push([]string{""}, PushData{
			ContentType: "测试",
			Extras:      nil,
			MsgContent:  "测试",
			Title:       "cqh哥哥好帅",
		})
		t.Log(err)
	}

}
