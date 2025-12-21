package ioc

import (
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitEtcdClient() *clientv3.Client {
	var cfg clientv3.Config
	err := viper.UnmarshalKey("etcd", &cfg)
	if err != nil {
		panic(err)
	}
	client, err := clientv3.New(cfg)
	if err != nil {
		panic(err)
	}
	return client
}
