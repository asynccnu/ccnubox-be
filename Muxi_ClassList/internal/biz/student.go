package biz

import (
	"context"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/model"
)

type Student interface {
	GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*model.ClassInfo, []*model.StudentCourse, error)
}
type Undergraduate struct{}

func (u *Undergraduate) GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*model.ClassInfo, []*model.StudentCourse, error) {
	var (
		classInfos = make([]*model.ClassInfo, 0)
		scs        = make([]*model.StudentCourse, 0)
	)
	resp, err := craw.GetClassInfosForUndergraduate(ctx, model.GetClassInfosForUndergraduateReq{
		StuID:    stuID,
		Year:     year,
		Semester: semester,
		Cookie:   cookie,
	})
	if err != nil {
		return nil, nil, err
	}
	if resp != nil {
		if resp.ClassInfos != nil {
			classInfos = resp.ClassInfos
		}
		if resp.StudentCourses != nil {
			scs = resp.StudentCourses
		}
	}
	return classInfos, scs, nil
}

type GraduateStudent struct{}

func (g *GraduateStudent) GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*model.ClassInfo, []*model.StudentCourse, error) {
	var (
		classInfos = make([]*model.ClassInfo, 0)
		scs        = make([]*model.StudentCourse, 0)
	)
	resp2, err := craw.GetClassInfoForGraduateStudent(ctx, model.GetClassInfoForGraduateStudentReq{
		StuID:    stuID,
		Year:     year,
		Semester: semester,
		Cookie:   cookie,
	})
	if err != nil {
		return nil, nil, err
	}
	if resp2.ClassInfos != nil {
		classInfos = resp2.ClassInfos
	}
	if resp2.StudentCourses != nil {
		scs = resp2.StudentCourses
	}
	return classInfos, scs, nil
}
