package model

import (
	"testing"
)

func TestWrapClassInfo_ConvertToClass(t *testing.T) {
	classinfos := []*ClassInfo{
		&ClassInfo{
			JxbId:    "123",
			Year:     "2024",
			Semester: "1",
			Weeks:    32,
		},
	}
	wc := WrapClassInfo(classinfos)
	class, jxb := wc.ConvertToClass(6)
	t.Log(class, jxb)
}
