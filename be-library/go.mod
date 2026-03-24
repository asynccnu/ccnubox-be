module github.com/asynccnu/ccnubox-be/be-library

go 1.24.0

replace github.com/asynccnu/ccnubox-be/common => ./../common

require (
	github.com/asynccnu/ccnubox-be/common v0.0.0-00010101000000-000000000000
	github.com/joho/godotenv v1.5.1
	github.com/redis/go-redis/v9 v9.17.3
	github.com/tidwall/gjson v1.18.0
	go.etcd.io/etcd/client/v3 v3.6.7
	google.golang.org/grpc v1.77.0
	gorm.io/driver/mysql v1.6.0
	gorm.io/gorm v1.31.1
)
