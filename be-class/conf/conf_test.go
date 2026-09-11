package conf

import (
	"bytes"
	"os"
	"testing"

	baseconf "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/spf13/viper"
)

func TestExampleConfigMatchesUnifiedSchema(t *testing.T) {
	serverData, err := os.ReadFile("../config/config-example.yaml")
	if err != nil {
		t.Fatal(err)
	}

	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(serverData)); err != nil {
		t.Fatal(err)
	}

	var server ServerConf
	if err := v.Unmarshal(&server); err != nil {
		t.Fatal(err)
	}
	if server.Class == nil || server.Class.HTTP == nil || server.Class.Elasticsearch == nil {
		t.Fatal("class server configuration is incomplete")
	}
	if server.Class.HTTP.Addr != ":18000" || !server.Class.Elasticsearch.KeepDataAfterRestart {
		t.Fatal("class server configuration contains unexpected values")
	}

	infraData, err := os.ReadFile("../../config-infra-example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	v = viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(infraData)); err != nil {
		t.Fatal(err)
	}

	var infra InfraConf
	infra.InfraConf = &baseconf.InfraConf{}
	if err := v.Unmarshal(infra.InfraConf); err != nil {
		t.Fatal(err)
	}
	if infra.Grpc == nil || (*infra.Grpc)["class"] == nil || infra.Redis == nil || infra.Mysql == nil || infra.Etcd == nil || infra.Elasticsearch == nil {
		t.Fatal("infrastructure configuration is incomplete")
	}
	if (*infra.Grpc)["class"].Addr != ":20001" || len(infra.Elasticsearch.URLs) == 0 {
		t.Fatal("class gRPC or Elasticsearch infrastructure configuration contains unexpected values")
	}
}
