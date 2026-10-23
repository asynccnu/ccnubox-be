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
	FeedWhiteList       *FeedWhiteListConf `yaml:"feedWhiteList"`
	conf.BaseServerConf `mapstructure:",squash"`
	ServiceName         string         `yaml:"serviceName"`
	ElecpriceController *ElecPriceConf `yaml:"elecpriceController"`
	Http                *HttpConf      `yaml:"http"`
	Administrators      []string       `yaml:"administrators"`
	JWT                 *JWTConf       `yaml:"jwt"`
	Oss                 *OssConf       `yaml:"oss"`
	//Prometheus          *PrometheusConf `yaml:"prometheus"`
	BasicAuth     *BasicAuthConf     `yaml:"basicAuth"`
	ClientMetrics *ClientMetricsConf `yaml:"clientMetrics"`
	Tiered        *TieredConf        `yaml:"tiered"`
}
type FeedWhiteListConf struct {
	WindowDays    int      `yaml:"windowDays"`
	MinActiveDays int      `yaml:"minActiveDays"`
	ScriptCommand []string `yaml:"scriptCommand"`
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

//type PrometheusConf struct {
//	Namespace string `yaml:"namespace"` //项目名称
//
//	RouterCounter struct {
//		Name string `yaml:"name"`
//		Help string `yaml:"help"`
//	} `yaml:"routerCounter"`
//
//	ActiveConnections struct {
//		Name string `yaml:"name"`
//		Help string `yaml:"help"`
//	} `yaml:"activeConnections"`
//
//	DurationTime struct {
//		Name string `yaml:"name"`
//		Help string `yaml:"help"`
//	} `yaml:"durationTime"`
//
//	ActiveUsers24h struct {
//		Name string `yaml:"name"`
//		Help string `yaml:"help"`
//	} `yaml:"activeUsers24h"`
//}

type BasicAuthConf struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type ClientMetricsConf struct {
	ClientKey      string `yaml:"clientKey"`
	MaxBatchSize   int    `yaml:"maxBatchSize"`
	MaxBodyBytes   int64  `yaml:"maxBodyBytes"`
	MaxAppVersions int    `yaml:"maxAppVersions"`
}

type TieredConf struct {
	Low    int `yaml:"low"`
	Middle int `yaml:"middle"`
	High   int `yaml:"high"`
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
