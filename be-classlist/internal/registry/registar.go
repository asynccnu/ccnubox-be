package registry

import (
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"time"
)

func NewRegistrarServer(c *conf.Registry, logger log.Logger) *etcd.Registry {
	// ETCD源地址
	endpoints := []string{c.Etcd.Addr}

	// ETCD配置信息
	etcdCfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Second,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
		Username:    c.Etcd.Username,
		Password:    c.Etcd.Password,
	}

	// 创建ETCD客户端
	client, err := clientv3.New(etcdCfg)
	if err != nil {
		logger.Log(log.LevelError, "msg", "failed to connect etcd", "err", err)
		panic(err)
	}
	logger.Log(log.LevelInfo, "msg", "service registry successfully")
	//fmt.Println("connect successfully")
	// 创建服务注册 registrar
	registrar := etcd.New(client)
	return registrar
}
