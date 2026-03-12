package tool_test

import(
	"github.com/asynccnu/ccnubox-be/common/tool"
	"testing"
	"time"
)

func TestGetCurrentAcademicYearAndSemester(t *testing.T) {
	y,s := tool.GetCurrentAcademicYearAndSemester(time.Now())
	t.Logf("当前学年：%d, 当前学期：%d", y, s)
}
