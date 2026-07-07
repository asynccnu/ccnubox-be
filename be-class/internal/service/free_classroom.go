package service

import (
	"context"
	"encoding/json"

	"github.com/asynccnu/ccnubox-be/be-class/internal/errcode"
	pb "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classService/v1"
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
	searcher          FreeClassroomSearcher
	classroomProvider ClassroomJSONProvider
}

func NewFreeClassroomSvc(searcher FreeClassroomSearcher, classroomProvider ClassroomJSONProvider) *FreeClassroomSvc {
	return &FreeClassroomSvc{
		searcher:          searcher,
		classroomProvider: classroomProvider,
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

func (s *FreeClassroomSvc) GetClassrooms(ctx context.Context, req *pb.GetClassroomsReq) (*pb.GetClassroomsResp, error) {
	var data struct {
		ClassRooms []string `json:"class_rooms"`
	}
	if err := json.Unmarshal(s.classroomProvider.ClassroomJSON(), &data); err != nil {
		return &pb.GetClassroomsResp{}, errcode.Err_FreeClassroomSearch
	}

	return &pb.GetClassroomsResp{
		ClassRooms: data.ClassRooms,
	}, nil
}
