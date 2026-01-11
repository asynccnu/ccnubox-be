package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-website/conf"
	"github.com/jinzhu/copier"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitEtcdClient(cfg *conf.InfraConf) *clientv3.Client {
	var c clientv3.Config
	copier.Copy(&c, cfg)
	client, err := clientv3.New(c)
	if err != nil {
		panic(err)
	}
	return client
}
