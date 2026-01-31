package conf

import "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"

const (
	ServerEnv = "CCNUBOX_CONTENT_NACOS_DSN"
)

// InfraConf 通用配置
type InfraConf struct {
	*conf.InfraConf `mapstructure:",squash"` //为了能够正常解析需要对其进行拍平
}

// ServerConf 服务配置
type ServerConf struct {
	conf.BaseServerConf `mapstructure:",squash"`
	Qiniu               *QiniuConfig              `yaml:"qiniu"`
	CalendarController  *CalendarControllerConfig `yaml:"calendarController"`
	Version             *UpdateVersionConfig      `yaml:"version"`
}

type QiniuConfig struct {
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	Bucket    string `yaml:"bucket"`
	Domain    string `yaml:"domain"`
	BaseName  string `yaml:"baseName"`
}

type CalendarControllerConfig struct {
	DurationTime int `yaml:"durationTime"` // 单位：小时
}

type UpdateVersionConfig struct {
	Version string `yaml:"version"`
}

func InitServerConf() *ServerConf {
	return conf.InitConfig[ServerConf](ServerEnv)
}

func InitInfraConfig() *InfraConf {
	return &InfraConf{conf.InitInfraConfig()}
}
