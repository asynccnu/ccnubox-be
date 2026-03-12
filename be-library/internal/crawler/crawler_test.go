package crawler

import (
	"context"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/internal/client"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestCrawler(t *testing.T) {
	etcdClient, err := clientv3.New(clientv3.Config{Endpoints: []string{"localhost:2379"}})
	if err != nil {
		panic(err)
	}

	r := etcd.New(etcdClient)

	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///prod/user"),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(20*time.Second),
		grpc.WithMiddleware(
			tracing.Client(),
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}

	userClient := userv1.NewUserServiceClient(conn)

	ccnuService := client.NewCCNUServiceProxy(userClient)

	httpClient := client.NewHttpClient()

	crawler1 := NewLibraryCrawler(log.With(log.DefaultLogger), ccnuService, 5*time.Second, httpClient)

	result, err := crawler1.GetSeatInfos(context.Background(), "", []string{""})
	if err != nil {
		panic(err)
	}
	t.Log(result)

	//token, err := crawler1.GetLibraryDiscussionToken("")
	//if err != nil {
	//	return
	//}
	//
	//t.Log(token)

	//msg, err := crawler1.ReserveSeat(context.Background(), "", "", "", "")
	//if err != nil {
	//	panic(err)
	//}
	//t.Log(msg)

	//records, err := crawler1.GetHistory(context.Background(), "")
	//if err != nil {
	//	panic(err)
	//}
	//t.Log(records)

	//freeList, err := crawler1.GetFreeList(context.Background(), "", "")
	//if err != nil {
	//	panic(err)
	//}
	//t.Log(freeList)

	//res, err := crawler1.GetDiscussion(context.Background(), "", "", "", "")
	//if err != nil {
	//	panic(err)
	//}
	//t.Log(res)

}
