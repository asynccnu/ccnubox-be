package model

import (
	"testing"
)

func TestClassInfo_SearchWeek(t *testing.T) {
	var classInfo ClassInfo
	classInfo.Weeks = 131071
	t.Log(classInfo.SearchWeek(7))
}
