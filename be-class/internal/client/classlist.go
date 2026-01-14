package client

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-class/internal/conf"
	clog "github.com/asynccnu/ccnubox-be/be-class/internal/log"
	"github.com/asynccnu/ccnubox-be/be-class/internal/model"
	classlist "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	b_conf "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
)

type ClassListService struct {
	cs classlist.ClasserClient
}

func NewClassListService(r *etcd.Registry, cf *conf.Registry, env *b_conf.Env) (*ClassListService, error) {
	clog.LogPrinter.Infof("init classlist grpc client, target service address: %s", cf.Classlistsvc)
	conn, err := InitClient(r, cf.Classlistsvc, env)
	if err != nil {
		clog.LogPrinter.Errorw("kind", "grpc-client", "reason", "GRPC_CLIENT_INIT_ERROR", "err", err)
		return nil, err
	}
	cs := classlist.NewClasserClient(conn)
	clog.LogPrinter.Infof("init classlist grpc client success, address: %s", cf.Classlistsvc)
	return &ClassListService{cs: cs}, nil
}

func (c *ClassListService) GetAllSchoolClassInfos(ctx context.Context, xnm, xqm, cursor string) ([]model.ClassInfo, string, error) {
	resp, err := c.cs.GetAllClassInfo(ctx, &classlist.GetAllClassInfoRequest{
		Year:     xnm,
		Semester: xqm,
		Cursor:   cursor,
	})
	if err != nil {
		clog.LogPrinter.Errorf("send request for service[%v] to get all classInfos[xnm:%v xqm:%v] failed: %v", "classlist service", xnm, xqm, err)
		return nil, "", err
	}
	var classInfos = make([]model.ClassInfo, 0, len(resp.ClassInfos))
	for _, info := range resp.ClassInfos {
		classInfo := model.ClassInfo{
			ID:           info.Id,
			Day:          info.Day,
			Teacher:      info.Teacher,
			Where:        info.Where,
			ClassWhen:    info.ClassWhen,
			WeekDuration: info.WeekDuration,
			Classname:    info.Classname,
			Credit:       info.Credit,
			Weeks:        info.Weeks,
			Semester:     info.Semester,
			Year:         info.Year,
		}
		classInfos = append(classInfos, classInfo)
	}
	return classInfos, resp.LastTime, nil
}

func (c *ClassListService) AddClassInfoToClassListService(ctx context.Context, req *classlist.AddClassRequest) (*classlist.AddClassResponse, error) {
	resp, err := c.cs.AddClass(ctx, req)
	if err != nil {
		clog.LogPrinter.Errorf("send request for service[%v] to add  classInfos[%v] failed: %v", "classlist service", req, err)
		return nil, err
	}
	return resp, nil
}

func (c *ClassListService) GetSchoolDay(ctx context.Context) (string, error) {
	resp, err := c.cs.GetSchoolDay(ctx, &classlist.GetSchoolDayReq{})
	if err != nil {
		return "", err
	}
	return resp.SchoolTime, nil
}
