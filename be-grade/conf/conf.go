package conf

import (
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
)

const (
	ServerEnv = "CCNUBOX_GRADE_NACOS_DSN"
)

// InfraConf 通用配置
type InfraConf struct {
	*conf.InfraConf `mapstructure:",squash"` //为了能够正常解析需要对其进行拍平
}

// ServerConf 服务配置
type ServerConf struct {
	conf.BaseServerConf `mapstructure:",squash"`
	GradeConf           *GradeConf   `yaml:"gradeConf"`
	ConsumeConf         *ConsumeConf `yaml:"consumeConf"`
}

type ConsumeConf struct {
	ConsumeTime int `yaml:"consumeTime"`
	ConsumeNum  int `yaml:"consumeNum"`
}

type GradeConf struct {
	High   int `yaml:"high"`
	Middle int `yaml:"middle"`
	Low    int `yaml:"low"`
}

func InitServerConf() *ServerConf {
	return conf.InitConfig[ServerConf](ServerEnv)
}

func InitInfraConfig() *InfraConf {
	return &InfraConf{conf.InitInfraConfig()}
}
