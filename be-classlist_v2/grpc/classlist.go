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
