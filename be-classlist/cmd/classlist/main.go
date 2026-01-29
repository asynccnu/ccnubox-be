package main

import (
	"flag"
	"os"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/classLog"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/metrics"
	b_conf "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	b_grpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Version is the version of the compiled software.
	Version string = "v1"
	// flagconf is the config flag.
	flagconf string
)

func init() {
	// 预加载.env文件,用于本地开发
	_ = godotenv.Load()
	prometheus.MustRegister(metrics.Counter, metrics.Summary)
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(env *b_conf.Env, logger log.Logger, gs *grpc.Server, r *etcd.Registry, server *conf.Server) *kratos.App {
	return kratos.New(
		kratos.Name(b_grpc.GetNamePrefix(env, server.Name)),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
		),
		kratos.Registrar(r),
	)
}

func main() {
	flag.Parse()
	var bc conf.Bootstrap
	if os.Getenv(conf.ClassList) != "" {
		bootstrap := conf.InitBootstrap()
		if bootstrap == nil {
			panic("nacos 配置初始化失败")
		}
		bc = *bootstrap
	} else {
		c := config.New(
			config.WithSource(
				file.NewSource(flagconf),
			),
		)
		defer c.Close()
		if err := c.Load(); err != nil {
			panic(err)
		}
		if err := c.Scan(&bc); err != nil {
			panic(err)
		}
	}

	logger := log.With(classLog.Logger(bc.Zaplog), "service.name", bc.Server.Name)
	classLog.InitGlobalLogger(logger)

	// gorm的日志文件
	// 在main函数中声明,程序结束执行Close
	// 防止只有连接数据库的时候，才会将sql语句写入
	logfile := classLog.NewLumberjackLogger(bc.Data.Database.LogPath,
		bc.Data.Database.LogFileName, 6, 5, 30, false)
	defer logfile.Close()

	app, cleanup, err := wireApp(bc.Env, bc.Server, bc.Data, bc.Registry, bc.Schoolday, bc.Defaults, logfile, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}

}
