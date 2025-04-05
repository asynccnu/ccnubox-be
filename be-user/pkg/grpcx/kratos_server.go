package grpcx

import (
	"github.com/asynccnu/ccnubox-be/be-user/pkg/logger"
	etcd "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"strconv"
	"time"
)

type KratosServer struct {
	*grpc.Server
	Name       string
	Weight     int
	EtcdTTL    time.Duration
	EtcdClient *etcdv3.Client
	stop       func() error
	L          logger.Logger
}

// Serve 启动服务器并且阻塞
func (s *KratosServer) Serve() error {
	r := etcd.New(s.EtcdClient, etcd.RegisterTTL(s.EtcdTTL))
	app := kratos.New(
		kratos.Metadata(map[string]string{
			"weight": strconv.Itoa(s.Weight),
		}),
		kratos.Name(s.Name),
		kratos.Server(
			s.Server,
		),
		kratos.Registrar(r),
	)
	s.stop = app.Stop
	return app.Run()
}

func (s *KratosServer) Close() error {
	err := s.EtcdClient.Close()
	if err != nil {
		return err
	}
	return s.stop()
}
