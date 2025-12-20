module github.com/asynccnu/ccnubox-be/be-grade

go 1.24.0

replace github.com/asynccnu/ccnubox-be/common/be-api => ./../common/be-api

replace github.com/asynccnu/ccnubox-be/common/pkg => ./../common/pkg

require (
	github.com/asynccnu/ccnubox-be/common/be-api v0.0.0-00010101000000-000000000000
	github.com/asynccnu/ccnubox-be/common/pkg v0.0.0-00010101000000-000000000000
	github.com/avast/retry-go/v4 v4.7.0
	github.com/go-kratos/kratos/contrib/registry/etcd/v2 v2.0.0-20251205160234-b9fab9a5a5ab
	github.com/go-kratos/kratos/v2 v2.9.2
	github.com/go-redsync/redsync/v4 v4.14.1
	github.com/google/wire v0.6.0
	github.com/redis/go-redis/v9 v9.17.2
	github.com/robfig/cron/v3 v3.0.1
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	go.etcd.io/etcd/client/v3 v3.6.6
	go.uber.org/zap v1.27.1
	golang.org/x/sync v0.19.0
	google.golang.org/grpc v1.77.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gorm.io/driver/mysql v1.5.7
	gorm.io/gorm v1.25.12
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-kratos/aegis v0.2.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.etcd.io/etcd/api/v3 v3.6.6 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.6 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
