package service

import (
	"context"
	pb "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classService/v1"
	v1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classlist/v1"
	"github.com/asynccnu/ccnubox-be/be-class/internal/model"
)

type ClassInfoProxy interface {
	AddClassInfoToClassListService(ctx context.Context, request *v1.AddClassRequest) (*v1.AddClassResponse, error)
	SearchClassInfo(ctx context.Context, keyWords string, xnm, xqm string) ([]model.ClassInfo, error)
}

type ClassServiceService struct {
	pb.UnimplementedClassServiceServer
	cp ClassInfoProxy
}

func NewClassServiceService(cp ClassInfoProxy) *ClassServiceService {
	return &ClassServiceService{
		cp: cp,
	}
}

func (s *ClassServiceService) SearchClass(ctx context.Context, req *pb.SearchRequest) (*pb.SearchReply, error) {
	classInfos, err := s.cp.SearchClassInfo(ctx, req.GetSearchKeyWords(), req.GetYear(), req.GetSemester())
	if err != nil {
		return &pb.SearchReply{}, err
	}
	var pClassInfos = make([]*pb.ClassInfo, 0)
	for _, classInfo := range classInfos {
		info := HandleClassInfo(classInfo)
		pClassInfos = append(pClassInfos, info)
	}
	return &pb.SearchReply{
		ClassInfos: pClassInfos,
	}, nil
}

func (s *ClassServiceService) AddClass(ctx context.Context, req *pb.AddClassRequest) (*pb.AddClassReply, error) {
	preq := &v1.AddClassRequest{
		StuId:    req.GetStuId(),
		Name:     req.GetName(),
		DurClass: req.GetDurClass(),
		Where:    req.GetWhere(),
		Teacher:  req.GetTeacher(),
		Weeks:    req.GetWeeks(),
		Semester: req.GetSemester(),
		Year:     req.GetYear(),
		Day:      req.GetDay(),
	}
	if req.Credit != nil {
		var credit = req.GetCredit()
		preq.Credit = &credit
	}
	resp, err := s.cp.AddClassInfoToClassListService(ctx, preq)
	if err != nil {
		return &pb.AddClassReply{}, err
	}
	return &pb.AddClassReply{
		Id:  resp.Id,
		Msg: resp.Msg,
	}, nil
}
func HandleClassInfo(classInfo model.ClassInfo) *pb.ClassInfo {
	return &pb.ClassInfo{
		Day:          classInfo.Day,
		Teacher:      classInfo.Teacher,
		Where:        classInfo.Where,
		ClassWhen:    classInfo.ClassWhen,
		WeekDuration: classInfo.WeekDuration,
		Classname:    classInfo.Classname,
		Credit:       classInfo.Credit,
		Weeks:        classInfo.Weeks,
		Semester:     classInfo.Semester,
		Year:         classInfo.Year,
		Id:           classInfo.ID,
	}
}
