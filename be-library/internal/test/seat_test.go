package test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/client"
	"github.com/asynccnu/ccnubox-be/be-library/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-library/internal/crawler"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/cache"
	"github.com/asynccnu/ccnubox-be/be-library/internal/registry"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
)

var confPath string

func init() {
	flag.StringVar(&confPath, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

// 全局 repo
var repo biz.SeatRepo
var bizz biz.LibraryBiz
var ccnuServiceProxy biz.CCNUServiceProxy

// TestMain 在所有测试前初始化依赖
func TestMain(m *testing.M) {
	flag.Parse()
	c := config.New(
		config.WithSource(file.NewSource(confPath)),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}
	rdb := data.NewRedisDB(bc.Data, log.NewStdLogger(os.Stdout))
	confServer := bc.Server
	confRegistry := bc.Registry
	logger := log.NewStdLogger(os.Stdout)
	cookiePool := client.NewCookiePoolProvider()
	etcdRegistry := registry.NewRegistrarServer(confRegistry, logger)
	userServiceClient, err := client.NewClient(etcdRegistry, confRegistry, logger)
	if err != nil {
		panic(err)
	}
	ccnuServiceProxy = client.NewCCNUServiceProxy(userServiceClient)
	duration := biz.NewWaitTime(confServer)
	libraryCrawler := crawler.NewLibraryCrawler(logger, cookiePool, ccnuServiceProxy, duration)
	seatCache := cache.NewSeatCache(rdb, logger)
	assembler := data.NewAssembler()
	group := data.NewSingleflight()
	repo = data.NewSeatRepo(seatCache, bc.Data, logger, assembler, group, libraryCrawler)
	bizz = biz.NewLibraryBiz(libraryCrawler, log.NewStdLogger(os.Stdout), repo, nil, nil)

	// 执行测试
	m.Run()
}

// 10s -> 8s
func TestSaveRoomSeatsInRedis(t *testing.T) {
	stuID := ""
	ctx := context.Background()

	err := repo.SaveRoomSeatsInRedis(ctx, stuID, biz.RoomIDs)
	if err != nil {
		panic(err)
	}
}

func TestGetGetLibraryCookie(t *testing.T) {
	stuID := ""
	ctx := context.Background()
	cookie, err := ccnuServiceProxy.GetLibraryCookie(ctx, stuID)
	if err != nil {
		panic(err)
	}
	fmt.Println(cookie)
}

func TestGetSeat(t *testing.T) {
	ctx := context.Background()
	stuID := ""

	seats, err := repo.GetSeatInfos(ctx, stuID, biz.RoomIDs)
	if err != nil {
		panic(err)
	}

	for _, seatLine := range seats {
		for _, seat := range seatLine {
			fmt.Println(seat)
		}
	}
}

func TestFindFirstAvailbleSeat(t *testing.T) {
	roomIDs := []string{"100455824"}
	ctx := context.Background()
	devid, _, err := repo.FindFirstAvailableSeat(ctx, 2000, 2100, roomIDs)
	if err != nil {
		panic(err)
	}
	fmt.Println(devid)
}

func TestReserveSeatRandomly(t *testing.T) {
	id := ""
	start := "2025-10-16 20:00"
	end := "2025-10-16 21:00"
	ctx := context.Background()
	msg, err := bizz.ReserveSeatRandomly(ctx, id, start, end, biz.RoomIDs)
	if err != nil {
		panic(err)
	}
	fmt.Println(msg)
}
