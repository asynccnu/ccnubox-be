package conf

import "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"

const (
	ServerEnv = "CCNUBOX_COUNTER_NACOS_DSN"
)

// InfraConf 通用配置
type InfraConf struct {
	*conf.InfraConf `mapstructure:",squash"` //为了能够正常解析需要对其进行拍平
}

// ServerConf 服务配置
type ServerConf struct {
	conf.BaseServerConf `mapstructure:",squash"`
	CountLevel          *CountLevelConfig `yaml:"countLevel"`
}

type CountLevelConfig struct {
	Low    int64 `yaml:"low"`
	Middle int64 `yaml:"middle"`
	High   int64 `yaml:"high"`
	Step   int64 `yaml:"step"`
}

func InitServerConf() *ServerConf {
	return conf.InitConfig[ServerConf](ServerEnv)
}

func InitInfraConfig() *InfraConf {
	return &InfraConf{conf.InitInfraConfig()}
}
