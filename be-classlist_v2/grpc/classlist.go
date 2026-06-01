package grpc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/service"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	"google.golang.org/grpc"
)

type ClasslistServiceServer struct {
	classlistv1.UnimplementedClasserServer
	svc *service.ClassListService
}

func NewCalendarServiceServer(svc *service.ClassListService) *ClasslistServiceServer {
	return &ClasslistServiceServer{
		svc: svc,
	}
}

// 注册为grpc服务
func (c *ClasslistServiceServer) Register(server grpc.ServiceRegistrar) {
	classlistv1.RegisterClasserServer(server, c)
}

// GetClass 获取课表
// grpc 层只负责协议适配：pb 解包 → 调 service → BO 装箱回 pb
func (c *ClasslistServiceServer) GetClass(ctx context.Context, req *classlistv1.GetClassRequest) (*classlistv1.GetClassResponse, error) {
	classInfos, lastTime, err := c.svc.GetClass(ctx, req.GetStuId(), req.GetYear(), req.GetSemester(), req.GetRefresh())
	if err != nil {
		return &classlistv1.GetClassResponse{}, err
	}

	// BO → pb 装箱
	pbClasses := make([]*classlistv1.Class, 0, len(classInfos))
	for _, ci := range classInfos {
		pbClasses = append(pbClasses, &classlistv1.Class{
			Info: classInfoBOToPb(ci),
		})
	}

	// lastTime 为 nil 时使用哨兵值（pb int64 无法表达 nil）
	var lastTimeStamp int64
	if lastTime != nil {
		lastTimeStamp = toShanghaiTimeStamp(*lastTime)
	} else {
		lastTimeStamp = time.Date(1949, 10, 1, 0, 0, 0, 0, time.Local).Unix()
	}

	return &classlistv1.GetClassResponse{
		Classes:  pbClasses,
		LastTime: lastTimeStamp,
	}, nil
}

// 增加自编课程
func (c *ClasslistServiceServer) AddClass(ctx context.Context, req *classlistv1.AddClassRequest) (*classlistv1.AddClassResponse, error) {
	id, msg, err := c.svc.AddClass(ctx,
		req.GetStuId(),
		req.GetName(),
		req.GetDurClass(),
		req.GetWhere(),
		req.GetTeacher(),
		req.GetWeeks(),
		req.GetSemester(),
		req.GetYear(),
		req.GetDay(),
		req.Credit,
	)
	if err != nil {
		return &classlistv1.AddClassResponse{}, err
	}
	return &classlistv1.AddClassResponse{
		Id:  id,
		Msg: msg,
	}, nil
}

func (c *ClasslistServiceServer) DeleteClass(ctx context.Context, req *classlistv1.DeleteClassRequest) (*classlistv1.DeleteClassResponse, error) {
	msg, err := c.svc.DeleteClass(ctx,
		req.GetStuId(),
		req.GetYear(),
		req.GetSemester(),
		req.GetId(),
	)
	if err != nil {
		return &classlistv1.DeleteClassResponse{
			Msg: msg,
		}, err
	}
	return &classlistv1.DeleteClassResponse{
		Msg: msg,
	}, nil
}

func (c *ClasslistServiceServer) UpdateClass(ctx context.Context, req *classlistv1.UpdateClassRequest) (*classlistv1.UpdateClassResponse, error) {
	classID, msg, err := c.svc.UpdateClass(ctx,
		req.GetStuId(),
		req.GetYear(),
		req.GetSemester(),
		req.GetClassId(),
		req.Name,
		req.DurClass,
		req.Where,
		req.Teacher,
		req.Weeks,
		req.Day,
		req.Credit,
	)
	if err != nil {
		return &classlistv1.UpdateClassResponse{
			Msg: msg,
		}, err
	}
	return &classlistv1.UpdateClassResponse{
		Msg:     msg,
		ClassId: classID,
	}, nil
}

func (c *ClasslistServiceServer) GetSchoolDay(ctx context.Context, _ *classlistv1.GetSchoolDayReq) (*classlistv1.GetSchoolDayResp, error) {
	holidayTime, schoolTime, err := c.svc.GetSchoolDay(ctx)
	if err != nil {
		return &classlistv1.GetSchoolDayResp{}, err
	}
	return &classlistv1.GetSchoolDayResp{
		HolidayTime: holidayTime,
		SchoolTime:  schoolTime,
	}, nil
}

func (c *ClasslistServiceServer) UpdateClassNote(ctx context.Context, req *classlistv1.UpdateClassNoteReq) (*classlistv1.UpdateClassNoteResp, error) {
	msg, err := c.svc.UpdateClassNote(ctx,
		req.GetStuId(),
		req.GetYear(),
		req.GetSemester(),
		req.GetClassId(),
		req.GetNote(),
	)
	if err != nil {
		return &classlistv1.UpdateClassNoteResp{
			Msg: msg,
		}, err
	}
	return &classlistv1.UpdateClassNoteResp{
		Msg: msg,
	}, nil
}

func (c *ClasslistServiceServer) DeleteClassNote(ctx context.Context, req *classlistv1.DeleteClassNoteReq) (*classlistv1.DeleteClassNoteResp, error) {
	msg, err := c.svc.DeleteClassNote(ctx,
		req.GetStuId(),
		req.GetYear(),
		req.GetSemester(),
		req.GetClassId(),
	)
	if err != nil {
		return &classlistv1.DeleteClassNoteResp{
			Msg: msg,
		}, err
	}
	return &classlistv1.DeleteClassNoteResp{
		Msg: msg,
	}, nil
}

func (c *ClasslistServiceServer) GetStuIdByJxbId(ctx context.Context, req *classlistv1.GetStuIdByJxbIdRequest) (*classlistv1.GetStuIdByJxbIdResponse, error) {
	stuIDs, err := c.svc.GetStuIdsByJxbId(ctx, req.GetJxbId())
	if err != nil {
		return &classlistv1.GetStuIdByJxbIdResponse{}, err
	}
	return &classlistv1.GetStuIdByJxbIdResponse{
		StuId: stuIDs,
	}, nil
}

func (c *ClasslistServiceServer) GetClassNatures(ctx context.Context, req *classlistv1.GetClassNaturesReq) (*classlistv1.GetClassNaturesResp, error) {
	natures, err := c.svc.GetClassNatures(ctx, req.GetStuId())
	if err != nil {
		return &classlistv1.GetClassNaturesResp{}, err
	}
	return &classlistv1.GetClassNaturesResp{
		ClassNatures: natures,
	}, nil
}
