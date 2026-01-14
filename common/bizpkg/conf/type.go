package conf

type Env string

const (
	EnvDev  Env = "dev"
	EnvTest Env = "test"
	EnvProd Env = "prod"
	EnvGrey Env = "grey"
)

// String 转换方法
func (e Env) String() string {
	return string(e)
}

// IsProd 判断是否为生产环境
func (e Env) IsProd() bool {
	return e == EnvProd
}

func (e Env) IsDev() bool {
	return e == EnvDev
}

// InfraConf 基础配置
type InfraConf struct {
	Env   *Env       `yaml:"env"`
	Etcd  *EtcdConf  `yaml:"etcd"`
	Redis *RedisConf `yaml:"redis"`
	Mysql *MysqlConf `yaml:"mysql"`
	Kafka *KafkaConf `yaml:"kafka"`
	Grpc  *GrpcConfs `yaml:"grpc"`
}

type EtcdConf struct {
	Endpoints []string `yaml:"endpoints"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
}

type RedisConf struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
}

type MysqlConf struct {
	DSN string `yaml:"dsn"`
}

type KafkaConf struct {
	Addrs []string `yaml:"addrs"`
}

type LogConf struct {
	Path       string `yaml:"path"`
	MaxSize    int    `yaml:"maxSize"`
	MaxBackups int    `yaml:"maxBackups"`
	MaxAge     int    `yaml:"maxAge"`
	Compress   bool   `yaml:"compress"`
}

// BaseServerConf
type BaseServerConf struct {
	Otel *OtelConf `yaml:"otel"`
	Log  *LogConf  `yaml:"log"`
}

// GRPC

// TODO 这个地方逻辑上非常耦合，期望找到更加合理而且好用的方案
// client和server用的同一个配置文件,同一个结构体,主要是为了防止出现服务名称不一致的问题做的权衡
type GrpcConfs map[string]*GrpcConf

type GrpcConf struct {
	Name          string `yaml:"name"`
	Weight        int    `yaml:"weight"`
	Addr          string `yaml:"addr"`
	EtcdTTL       int    `yaml:"etcdTTL"`
	ServerTimeout int    `yaml:"serverTimeout"`
	ClientTimeout int    `yaml:"clientTimeout"`
}

// OTel
type OtelConf struct {
	ServiceName    string `yaml:"serviceName"`
	ServiceVersion string `yaml:"serviceVersion"`
	Endpoint       string `yaml:"endpoint"`
}
