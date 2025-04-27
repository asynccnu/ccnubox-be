package biz

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/classLog"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/data"
	model2 "github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"github.com/asynccnu/ccnubox-be/be-classlist/test"
	"testing"
)

var classRepo *ClassRepo

var testStuID = "123"

var cla1 = &model2.ClassInfo{
	JxbId:        "jxb1",
	Day:          1,
	Teacher:      "cc",
	Where:        "somewhere1",
	ClassWhen:    "1-2",
	WeekDuration: "1-16周",
	Classname:    "haha",
	Credit:       1,
	Weeks:        65535,
	Semester:     "1",
	Year:         "2024",
}
var cla2 = &model2.ClassInfo{
	JxbId:        "jxb2",
	Day:          1,
	Teacher:      "cc",
	Where:        "somewhere2",
	ClassWhen:    "1-2",
	WeekDuration: "1-16周",
	Classname:    "haha",
	Credit:       1,
	Weeks:        65535,
	Semester:     "1",
	Year:         "2024",
}
var cla3 = &model2.ClassInfo{
	JxbId:        "jxb3",
	Day:          1,
	Teacher:      "cc",
	Where:        "somewhere3",
	ClassWhen:    "1-2",
	WeekDuration: "1-16周",
	Classname:    "haha",
	Credit:       1,
	Weeks:        65535,
	Semester:     "1",
	Year:         "2024",
}

func TestMain(m *testing.M) {
	db := test.NewDB("root:12345678@tcp(127.0.0.1:13306)/MuxiClass?charset=utf8mb4&parseTime=True&loc=Local")
	cli := test.NewRedisDB("127.0.0.1:16379", "")
	testdata := &data.Data{Mysql: db}
	logger := test.NewLogger()
	helper := classLog.NewClogger(logger)
	classInfoDBRepo := data.NewClassInfoDBRepo(testdata, helper)
	classInfoCacheRepo := data.NewClassInfoCacheRepo(cli, helper)
	classInfoRepo := NewClassInfoRepo(classInfoDBRepo, classInfoCacheRepo)
	studentAndCourseDBRepo := data.NewStudentAndCourseDBRepo(testdata, helper)
	studentAndCourseCacheRepo := data.NewStudentAndCourseCacheRepo(cli, helper)
	studentAndCourseRepo := NewStudentAndCourseRepo(studentAndCourseDBRepo, studentAndCourseCacheRepo)
	classRepo = NewClassRepo(classInfoRepo, testdata, studentAndCourseRepo, logger)
	cla1.UpdateID()
	cla2.UpdateID()
	cla3.UpdateID()
	m.Run()
}

func TestClassRepo_SaveClass(t *testing.T) {
	//循序渐进
	t.Run("模拟一开始没有任何课", func(t *testing.T) {
		classRepo.SaveClass(context.Background(), "123", "2024", "1", []*model2.ClassInfo{cla1}, []*model2.StudentCourse{
			{
				StuID:    testStuID,
				ClaID:    cla1.ID,
				Year:     "2024",
				Semester: "1",
			},
		})
	})
	t.Run("模拟有新增课", func(t *testing.T) {
		classRepo.SaveClass(context.Background(), "123", "2024", "1", []*model2.ClassInfo{cla1, cla2}, []*model2.StudentCourse{
			{
				StuID:    testStuID,
				ClaID:    cla1.ID,
				Year:     "2024",
				Semester: "1",
			},
			{
				StuID:    testStuID,
				ClaID:    cla2.ID,
				Year:     "2024",
				Semester: "1",
			},
		})
	})
	t.Run("模拟有退课行为", func(t *testing.T) {
		classRepo.SaveClass(context.Background(), "123", "2024", "1", []*model2.ClassInfo{cla2, cla3}, []*model2.StudentCourse{
			{
				StuID:    testStuID,
				ClaID:    cla2.ID,
				Year:     "2024",
				Semester: "1",
			},
			{
				StuID:    testStuID,
				ClaID:    cla3.ID,
				Year:     "2024",
				Semester: "1",
			},
		})
	})

}
