package registry

import (
	"time"

	"github.com/asynccnu/ccnubox-be/be-class/internal/conf"
	clog "github.com/asynccnu/ccnubox-be/be-class/internal/log"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/google/wire"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

var ProviderSet = wire.NewSet(NewRegistrarServer)

func NewRegistrarServer(c *conf.Registry) *etcd.Registry {
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
		panic(err)
	}
	//fmt.Println("connect successfully")

	clog.LogPrinter.Info("connect to etcd successfully")

	// 创建服务注册 registrar

	registrar := etcd.New(client)
	return registrar
}
