package service

import (
	"context"

	pb "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classService/v1"
	"github.com/asynccnu/ccnubox-be/be-class/internal/errcode"
)

type FreeClassroomSearcher interface {
	SearchAvailableClassroom(ctx context.Context, year, semester, stuID string, week, day int, sections []int, wherePrefix string) ([]AvailableClassroomStat, error)
}

type AvailableClassroomStat struct {
	Classroom     string
	AvailableStat []bool
}

type FreeClassroomSvc struct {
	pb.UnimplementedFreeClassroomSvcServer
	searcher FreeClassroomSearcher
}

func NewFreeClassroomSvc(searcher FreeClassroomSearcher) *FreeClassroomSvc {
	return &FreeClassroomSvc{
		searcher: searcher,
	}
}

func (s *FreeClassroomSvc) QueryFreeClassroom(ctx context.Context, req *pb.QueryFreeClassroomReq) (*pb.QueryFreeClassroomResp, error) {
	intSections := make([]int, len(req.Sections))
	for i, section := range req.Sections {
		intSections[i] = int(section)
	}
	stats, err := s.searcher.SearchAvailableClassroom(ctx, req.Year, req.Semester, req.StuID, int(req.Week), int(req.Day), intSections, req.WherePrefix)
	if err != nil {
		return &pb.QueryFreeClassroomResp{}, errcode.Err_FreeClassroomSearch
	}

	var res = make([]*pb.ClassroomAvailableStat, 0, len(stats))
	for _, stat := range stats {
		res = append(res, &pb.ClassroomAvailableStat{
			Classroom:     stat.Classroom,
			AvailableStat: stat.AvailableStat,
		})
	}
	return &pb.QueryFreeClassroomResp{
		Stat: res,
	}, nil
}
