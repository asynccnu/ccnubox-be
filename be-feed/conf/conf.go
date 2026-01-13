package conf

import (
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
)

const (
	ServerEnv = "CCNUBOX_FEED_NACOS_DSN"
)

// InfraConf 通用配置
type InfraConf struct {
	*conf.InfraConf `mapstructure:",squash"` //为了能够正常解析需要对其进行拍平
}

// ServerConf 服务配置
type ServerConf struct {
	conf.BaseServerConf `mapstructure:",squash"`
	MuxiController      *MuxiConf                `yaml:"muxiController"`
	Consume             *ConsumeConf             `yaml:"consume"`
	JPush               *JPushConf               `yaml:"jpush"`
	HolidayController   *HolidayControllerConfig `yaml:"holidayController"`
}

type MuxiConf struct {
	DurationTime int `yaml:"durationTime"`
}

type ConsumeConf struct {
	ConsumeTime int `yaml:"consumeTime"`
	ConsumeNum  int `yaml:"consumeNum"`
}

type JPushConf struct {
	AppKey       string `yaml:"appKey"`
	MasterSecret string `yaml:"masterSecret"`
}
type HolidayControllerConfig struct {
	DurationTime int64 `yaml:"durationTime"`
	AdvanceDay   int64 `yaml:"advanceDay"`
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
