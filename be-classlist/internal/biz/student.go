package biz

import "context"

// Student 学生接口
type Student interface {
	GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*ClassInfoBO, []*StudentCourse, int, error)
}
type Undergraduate struct{}

func (u *Undergraduate) GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*ClassInfoBO, []*StudentCourse, int, error) {
	infos, scs, sum, err := craw.GetClassInfosForUndergraduate(ctx, stuID, year, semester, cookie)
	if err != nil {
		return nil, nil, -1, err
	}
	return infos, scs, sum, nil
}

type GraduateStudent struct{}

func (g *GraduateStudent) GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*ClassInfoBO, []*StudentCourse, int, error) {
	infos, scs, sum, err := craw.GetClassInfoForGraduateStudent(ctx, stuID, year, semester, cookie)
	if err != nil {
		return nil, nil, -1, err
	}
	return infos, scs, sum, nil
}
