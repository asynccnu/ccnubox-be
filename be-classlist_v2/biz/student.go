package biz

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
)

// Student 学生接口
type Student interface {
	GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*model.ClassInfoBO, []*model.StudentCourseBO, int, error)
}
type Undergraduate struct{}

func (u *Undergraduate) GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*model.ClassInfoBO, []*model.StudentCourseBO, int, error) {
	infos, scs, sum, err := craw.GetClassInfosForUndergraduate(ctx, stuID, year, semester, cookie)
	if err != nil {
		return nil, nil, -1, err
	}
	return infos, scs, sum, nil
}

type GraduateStudent struct{}

func (g *GraduateStudent) GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*model.ClassInfoBO, []*model.StudentCourseBO, int, error) {
	infos, scs, sum, err := craw.GetClassInfoForGraduateStudent(ctx, stuID, year, semester, cookie)
	if err != nil {
		return nil, nil, -1, err
	}
	return infos, scs, sum, nil
}
