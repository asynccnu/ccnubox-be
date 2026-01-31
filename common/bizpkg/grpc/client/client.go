package client

import (
	"context"
	"fmt"
	"log"
	"time"

	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	classv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classService/v1"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	elecpricev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/elecprice/v1"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	libraryv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	b_grpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	k_grpc "github.com/go-kratos/kratos/v2/transport/grpc"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// --- 核心泛型工具函数 ---

func NewGrpcClient(ecli *clientv3.Client, cfg *conf.GrpcConf) *grpc.ClientConn {
	r := etcd.New(ecli)
	cc, err := k_grpc.DialInsecure(context.Background(),
		k_grpc.WithEndpoint(fmt.Sprintf("discovery:///%s", cfg.Name)),
		k_grpc.WithDiscovery(r),
		k_grpc.WithTimeout(time.Duration(cfg.ClientTimeout)*time.Second),
		k_grpc.WithMiddleware(tracing.Client()),
	)
	if err != nil {
		panic(fmt.Sprintf("连接服务 %s 失败: %v", cfg.Name, err))
	}
	return cc
}

func InitClient[T any](ecli *clientv3.Client, cfg *conf.GrpcConf, env *conf.Env, fn func(cc grpc.ClientConnInterface) T) T {
	newCfg := *cfg
	newCfg.Name = b_grpc.GetNamePrefix(env, newCfg.Name)
	return fn(NewGrpcClient(ecli, &newCfg))
}

// 内部辅助：安全获取配置
func getConf(cfg *conf.GrpcConfs, key string) *conf.GrpcConf {
	c, ok := (*cfg)[key]
	if !ok {
		log.Fatalf("配置中缺失服务定义: %s", key)
	}
	return c
}

// --- 各服务初始化函数 ---

func InitClassList(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) classlistv1.ClasserClient {
	return InitClient(ecli, getConf(cfg, b_grpc.CLASSLIST), env, classlistv1.NewClasserClient)
}

func InitUser(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) userv1.UserServiceClient {
	return InitClient(ecli, getConf(cfg, b_grpc.USER), env, userv1.NewUserServiceClient)
}

func InitGrade(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) gradev1.GradeServiceClient {
	return InitClient(ecli, getConf(cfg, b_grpc.GRADE), env, gradev1.NewGradeServiceClient)
}

func InitLibrary(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) libraryv1.LibraryClient {
	return InitClient(ecli, getConf(cfg, b_grpc.LIBRARY), env, libraryv1.NewLibraryClient)
}

func InitContent(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) contentv1.ContentServiceClient {
	return InitClient(ecli, getConf(cfg, b_grpc.CONTENT), env, contentv1.NewContentServiceClient)
}

func InitElecprice(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) elecpricev1.ElecpriceServiceClient {
	return InitClient(ecli, getConf(cfg, b_grpc.ELECPRICE), env, elecpricev1.NewElecpriceServiceClient)
}

func InitCCNU(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) ccnuv1.CCNUServiceClient {
	return InitClient(ecli, getConf(cfg, b_grpc.CCNU), env, ccnuv1.NewCCNUServiceClient)
}

func InitCounter(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) counterv1.CounterServiceClient {
	return InitClient(ecli, getConf(cfg, b_grpc.COUNTER), env, counterv1.NewCounterServiceClient)
}

func InitFeed(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) feedv1.FeedServiceClient {
	return InitClient(ecli, getConf(cfg, b_grpc.FEED), env, feedv1.NewFeedServiceClient)
}

func InitProxy(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) proxyv1.ProxyClient {
	return InitClient(ecli, getConf(cfg, b_grpc.PROXY), env, proxyv1.NewProxyClient)
}

func InitClass(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) classv1.ClassServiceClient {
	return InitClient(ecli, getConf(cfg, b_grpc.CLASSS), env, classv1.NewClassServiceClient)
}

func InitClassRoom(ecli *clientv3.Client, cfg *conf.GrpcConfs, env *conf.Env) classv1.FreeClassroomSvcClient {
	return InitClient(ecli, getConf(cfg, b_grpc.CLASSS), env, classv1.NewFreeClassroomSvcClient)
}
