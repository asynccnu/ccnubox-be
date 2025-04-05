package biz

import (
	"context"
	"fmt"
	"github.com/asynccnu/ccnubox-be/classService/internal/client"
	"github.com/asynccnu/ccnubox-be/classService/internal/conf"
	"github.com/asynccnu/ccnubox-be/classService/internal/data"
	"github.com/asynccnu/ccnubox-be/classService/internal/registry"
	"testing"
)

func initCS() *ClassSerivceUserCase {
	cli, err := data.NewEsClient(&conf.Data{Es: &conf.Data_ES{
		Url:      "http://127.0.0.1:9200",
		Setsniff: false,
		Username: "elastic",
		Password: "12345678",
	}})
	if err != nil {
		panic(fmt.Sprintf("failed to create elasticsearch client: %v", err))
	}
	dt, _, _ := data.NewClassData(cli)
	etcdRegistry := registry.NewRegistrarServer(&conf.Registry{
		Etcd: &conf.Etcd{
			Addr:     "127.0.0.1:2379",
			Username: "",
			Password: "",
		},
	})
	classListService, err := client.NewClassListService(etcdRegistry)
	if err != nil {
		panic(err)
	}

	cs := NewClassSerivceUserCase(dt, classListService)
	return cs
}

func TestClassSerivceUserCase_AddClassInfosToES(t *testing.T) {
	cs := initCS()
	cs.AddClassInfosToES(context.Background(), "2024", "1")
}
