package client

import (
	"context"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-class/internal/conf"
	"time"

	"github.com/asynccnu/ccnubox-be/be-class/internal/errcode"
	clog "github.com/asynccnu/ccnubox-be/be-class/internal/log"
	"github.com/asynccnu/ccnubox-be/be-class/internal/model"
	classlist "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	user "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewClassListService, NewCookieSvc, InitProxyClient)

type ClassListService struct {
	cs classlist.ClasserClient
}

func NewClassListService(r *etcd.Registry, cf *conf.Registry) (*ClassListService, error) {
	clog.LogPrinter.Infof("init classlist grpc client, target service address: %s", cf.Classlistsvc)
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(cf.Classlistsvc), // 需要发现的服务，如果是k8s部署可以直接用服务器本地地址:9001，9001端口是需要调用的服务的端口
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			tracing.Client(),
			recovery.Recovery(),
		),
	)
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

type CookieSvc struct {
	usc user.UserServiceClient
}

func NewCookieSvc(r *etcd.Registry, cf *conf.Registry) (*CookieSvc, error) {
	clog.LogPrinter.Infof("init user grpc client, target service address: %s", cf.Usersvc)
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(cf.Usersvc),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			tracing.Client(),
			recovery.Recovery(),
		),
		grpc.WithTimeout(120*time.Second),
	)
	if err != nil {
		clog.LogPrinter.Errorw("kind", "grpc-client", "reason", "GRPC_CLIENT_INIT_ERROR", "err", err)
		return nil, err
	}
	usc := user.NewUserServiceClient(conn)
	clog.LogPrinter.Infof("init user grpc client success, address: %s", cf.Classlistsvc)
	return &CookieSvc{usc: usc}, nil
}

func (c *CookieSvc) GetCookie(ctx context.Context, stuID string, tpe ...string) (string, error) {

	req := &user.GetCookieRequest{
		StudentId: stuID,
	}
	if len(tpe) != 0 {
		req.Type = tpe[0]
	}
	resp, err := c.usc.GetCookie(ctx, req)
	if err != nil {
		return "", fmt.Errorf("%s:%v", errcode.ErrCCNULogin, err)
	}
	cookie := resp.Cookie
	if len(cookie) == 0 {
		return "", fmt.Errorf("cookie is empty")
	}
	return cookie, nil
}
