package infra

import (
	"log"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitEtcdClient(cfg *conf.EtcdConf) *clientv3.Client {

	client, err := clientv3.New(clientv3.Config{
		Endpoints: cfg.Endpoints,
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		log.Fatal("连接 etcd 失败", err)
	}
	return client
}
