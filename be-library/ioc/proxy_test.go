package ioc

import (
	"context"
	"testing"

	libraryconf "github.com/asynccnu/ccnubox-be/be-library/conf"
	baseconf "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
)

func TestInitProxyClientsInDirectMode(t *testing.T) {
	cfg := &libraryconf.InfraConf{InfraConf: &baseconf.InfraConf{
		Proxy: &baseconf.ProxyConf{Mode: "direct"},
	}}
	if client := InitProxyClient(nil, cfg); client != nil {
		t.Fatal("direct mode must not initialize a be-proxy gRPC client")
	}

	client := InitHttpProxyClient(nil, cfg, zapx.NewZapLogger(zap.NewNop()))
	getter, ok := client.(interface {
		GetProxyAddr(context.Context, int) []string
	})
	if !ok {
		t.Fatal("proxy client does not expose proxy addresses")
	}
	addresses := getter.GetProxyAddr(context.Background(), 2)
	if len(addresses) != 2 || addresses[0] != "" || addresses[1] != "" {
		t.Fatalf("direct mode returned proxy addresses: %#v", addresses)
	}
}
