package conf

import (
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(InitInfraConfig, InitServerConf)

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
	WaitCrawTime        int32 `yaml:"waitCrawTime"`        // 等待爬虫抓取数据的时间,单位ms
	ClassExpiration     int32 `yaml:"classExpiration"`     // 课程过期时间,单位ms
	RecycleExpiration   int32 `yaml:"recycleExpiration"`   // 回收站课程过期时间,单位ms
	BlackListExpiration int32 `yaml:"blackListExpiration"` // 黑名单过期时间,如果要查询的课程在数据库不存在,列入黑名单,单位ms
	WaitUserSvcTime     int32 `yaml:"waitUserSvcTime"`     // 等待用户服务的时间,单位ms
	RefreshInterval     int32 `yaml:"refreshInterval"`     // 刷新间隔时间,单位ms

	HolidayTime string `yaml:"holidayTime"` // 放假日期(正式放假的第一天)
	SchoolTime  string `yaml:"schoolTime"`  // 上学日期(正式上学的第一天)
}

func InitServerConf() *ServerConf {
	return conf.InitConfig[ServerConf](ServerEnv)
}

func InitInfraConfig() *InfraConf {
	return &InfraConf{conf.InitInfraConfig()}
}
