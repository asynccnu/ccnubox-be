package conf

import "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"

const (
	ServerEnv = "CCNUBOX_CLASSLIST_NACOS_DSN"
)

type InfraConf struct {
	*conf.InfraConf `mapstructure:",squash"` //为了能够正常解析需要对其进行拍平
}

type ServerConf struct {
	conf.BaseServerConf `mapstructure:",squash"`
	ShenLongConf        *ShenLongConf `yaml:"shenLongConf"`
}

type ShenLongConf struct {
	API      string `yaml:"api"`
	Interval int    `yaml:"interval"`
	Retry    int    `yaml:"retry"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func InitServerConf() *ServerConf {
	return conf.InitConfig[ServerConf](ServerEnv)
}

func InitInfraConfig() *InfraConf {
	return &InfraConf{conf.InitInfraConfig()}
}