package client

import (
	"context"
	classlist "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classlist/v1"
	user "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/be-class/internal/errcode"
	clog "github.com/asynccnu/ccnubox-be/be-class/internal/log"
	"github.com/asynccnu/ccnubox-be/be-class/internal/model"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"
)

const CLASSLISTSERVICE = "discovery:///MuXi_ClassList"

var ProviderSet = wire.NewSet(NewClassListService, NewCookieSvc)

type ClassListService struct {
	cs classlist.ClasserClient
}

func NewClassListService(r *etcd.Registry) (*ClassListService, error) {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(CLASSLISTSERVICE), // 需要发现的服务，如果是k8s部署可以直接用服务器本地地址:9001，9001端口是需要调用的服务的端口
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
	return &ClassListService{cs: cs}, nil
}

func (c *ClassListService) GetAllSchoolClassInfos(ctx context.Context, xnm, xqm, cursor string) ([]model.ClassInfo, string, error) {
	resp, err := c.cs.GetAllClassInfo(ctx, &classlist.GetAllClassInfoRequest{
		Year:     xnm,
		Semester: xqm,
		Cursor:   cursor,
	})
	if err != nil {
		clog.LogPrinter.Errorf("send request for service[%v] to get all classInfos[xnm:%v xqm:%v] failed: %v", CLASSLISTSERVICE, xnm, xqm, err)
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
		clog.LogPrinter.Errorf("send request for service[%v] to add  classInfos[%v] failed: %v", CLASSLISTSERVICE, req, err)
		return nil, err
	}
	return resp, nil

}

type CookieSvc struct {
	usc user.UserServiceClient
}

const (
	USERSERVICE = "discovery:///user"
)

func NewCookieSvc(r *etcd.Registry) (*CookieSvc, error) {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(USERSERVICE),
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
	usc := user.NewUserServiceClient(conn)
	return &CookieSvc{usc: usc}, nil
}

func (c *CookieSvc) GetCookie(ctx context.Context, stuID string) (string, error) {

	resp, err := c.usc.GetCookie(ctx, &user.GetCookieRequest{
		StudentId: stuID,
	})
	if err != nil {
		return "", errcode.ErrCCNULogin
	}
	cookie := resp.Cookie
	return cookie, nil
}
