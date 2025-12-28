package user

import (
	"context"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	"github.com/spf13/viper"
	"strconv"
	"time"
)

type ClassListGetter interface {
	PreGetClassList(ctx context.Context, studentId string)
}

type classListGetter struct {
	currentSemester string
	classListSvc    classlistv1.ClasserClient
}

func NewClassListGetter(classListSvc classlistv1.ClasserClient) ClassListGetter {
	return &classListGetter{
		classListSvc:    classListSvc,
		currentSemester: viper.GetString("classlist.currentSemester"),
	}
}

func (g *classListGetter) PreGetClassList(ctx context.Context, studentId string) {
	// 异步获取学生课表,不需要等待结果
	go func() {
		_, _ = g.classListSvc.GetClass(ctx, &classlistv1.GetClassRequest{
			Refresh:  true,
			StuId:    studentId,
			Year:     g.getCurrentYear(),
			Semester: g.currentSemester,
		})
	}()
}

func (g *classListGetter) getCurrentYear() string {
	return strconv.Itoa(time.Now().Year())
}
