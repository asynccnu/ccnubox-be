package conf

import "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"

const (
	ServerEnv = "CCNUBOX_CLASSLIST_NACOS_DSN"
)

type InfraConf struct {
	*conf.InfraConf `mapstructure:",squash"` // 为了能够正常解析需要对其进行拍平
}

type ServerConf struct {
	conf.BaseServerConf `mapstructure:",squash"`
	ShenLongConf        *ShenLongConf  `yaml:"shenLongConf"`
	ClassListConf       *ClassListConf `yaml:"classListConf"`
}

type ShenLongConf struct {
	API      string `yaml:"api"`
	Interval int    `yaml:"interval"`
	Retry    int    `yaml:"retry"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type ClassListConf struct {
	WaitCrawTime        int32 `yaml:"waitCrawTime,omitempty"`        // 等待爬虫抓取数据的时间,单位ms
	ClassExpiration     int32 `yaml:"classExpiration,omitempty"`     // 课程过期时间,单位s
	RecycleExpiration   int32 `yaml:"recycleExpiration,omitempty"`   // 回收站课程过期时间,单位s
	BlackListExpiration int32 `yaml:"blackListExpiration,omitempty"` // 黑名单过期时间,如果要查询的课程在数据库不存在,列入黑名单,单位s
	WaitUserSvcTime     int32 `yaml:"waitUserSvcTime,omitempty"`     // 等待用户服务的时间,单位ms
	RefreshInterval     int32 `yaml:"refreshInterval,omitempty"`     // 刷新间隔时间,单位s
}

func InitServerConf() *ServerConf {
	return conf.InitConfig[ServerConf](ServerEnv)
}

func InitInfraConfig() *InfraConf {
	return &InfraConf{conf.InitInfraConfig()}
}
