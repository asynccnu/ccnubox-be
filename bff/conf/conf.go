package conf

import (
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
)

const (
	ServerEnv = "CCNUBOX_BFF_NACOS_DSN"
)

// InfraConf 通用配置
type InfraConf struct {
	*conf.InfraConf `mapstructure:",squash"` //为了能够正常解析需要对其进行拍平
}

// ServerConf 服务配置
type ServerConf struct {
	conf.BaseServerConf `mapstructure:",squash"`
	ElecpriceController *ElecPriceConf  `yaml:"elecpriceController"`
	Http                *HttpConf       `yaml:"http"`
	Administrators      []string        `yaml:"administrators"`
	JWT                 *JWTConf        `yaml:"jwt"`
	Oss                 *OssConf        `yaml:"oss"`
	Prometheus          *PrometheusConf `yaml:"prometheus"`
	BasicAuth           *BasicAuthConf  `yaml:"basicAuth"`
}
type HttpConf struct {
	Addr string `yaml:"addr"`
}

type JWTConf struct {
	JwtKey     string `yaml:"jwtKey"`
	RefreshKey string `yaml:"refreshKey"`
	EncKey     string `yaml:"encKey"`
}

type OssConf struct {
	AccessKey  string `yaml:"accessKey"`
	SecretKey  string `yaml:"secretKey"`
	BucketName string `yaml:"bucketName"`
	DomainName string `yaml:"domainName"`
	BaseName   string `yaml:"baseName"`
	FileName   string `yaml:"fileName"`
}

type PrometheusConf struct {
	Namespace string `yaml:"namespace"` //项目名称

	RouterCounter struct {
		Name string `yaml:"name"`
		Help string `yaml:"help"`
	} `yaml:"routerCounter"`

	ActiveConnections struct {
		Name string `yaml:"name"`
		Help string `yaml:"help"`
	} `yaml:"activeConnections"`

	DurationTime struct {
		Name string `yaml:"name"`
		Help string `yaml:"help"`
	} `yaml:"durationTime"`

	DailyActiveUsers struct {
		Name string `yaml:"name"`
		Help string `yaml:"help"`
	} `yaml:"dailyActiveUsers"`
}

type BasicAuthConf struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type ElecPriceConf struct {
	DurationTime int `yaml:"durationTime"`
}

func InitServerConf() *ServerConf {
	return conf.InitConfig[ServerConf](ServerEnv)
}

func InitInfraConfig() *InfraConf {
	return &InfraConf{conf.InitInfraConfig()}
}
