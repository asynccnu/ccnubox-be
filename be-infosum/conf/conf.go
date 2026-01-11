package conf

import (
	"bytes"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/spf13/viper"
)

// InfraConf 基础配置
type InfraConf struct {
	Etcd  EtcdConf  `json:"etcd"`
	Redis RedisConf `json:"redis"`
	Mysql MysqlConf `json:"mysql"`
	Log   LogConf   `json:"log"`
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

type LogConf struct {
	Path       string `yaml:"path"`
	MaxSize    int    `yaml:"maxSize"`
	MaxBackups int    `yaml:"maxBackups"`
	MaxAge     int    `yaml:"maxAge"`
	Compress   bool   `yaml:"compress"`
}

// TransConf 服务配置
type TransConf struct {
	Grpc GrpcConf `yaml:"grpc"`
}

type GrpcConf struct {
	Server GrpcSer            `yaml:"server"`
	Client map[string]GrpcCli `yaml:"client"`
}

type GrpcSer struct {
	Name    string `yaml:"name"`
	Weight  int    `yaml:"weight"`
	Addr    string `yaml:"addr"`
	EtcdTTL int    `yaml:"etcdTTL"`
}

type GrpcCli struct {
	Endpoint string `yaml:"endpoint"`
}

const (
	Infra = "CCNUBOX_NACOS_INFRA"
	Trans = "CCNUBOX_NACOS_INFOSUM"
)

func InitInfraConfig() *InfraConf {
	content, err := getConfigFromNacos(Infra)
	if err != nil {
		log.Println(err)

		localPath := "./config/config.yaml"
		fileContent, err := os.ReadFile(localPath)
		if err != nil {
			// 如果本地文件也读取失败，则彻底失败
			log.Fatalf("无法读取本地配置文件 %s，且 Nacos 配置获取失败: %v", localPath, err)
			return nil
		}
		content = string(fileContent)
	}

	v := viper.New()
	v.SetConfigType("yaml")

	if err = v.ReadConfig(bytes.NewBufferString(content)); err != nil {
		log.Fatal("配置文件解析失败:", err)
		return nil
	}

	var infra InfraConf
	err = v.Unmarshal(&infra)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	return &infra
}

func InitTransConfig() *TransConf {
	content, err := getConfigFromNacos(Trans)
	if err != nil {
		log.Println(err)

		localPath := "./config/config.yaml"
		fileContent, err := os.ReadFile(localPath)
		if err != nil {
			// 如果本地文件也读取失败，则彻底失败
			log.Fatalf("无法读取本地配置文件 %s，且 Nacos 配置获取失败: %v", localPath, err)
			return nil
		}
		content = string(fileContent)
	}

	v := viper.New()
	v.SetConfigType("yaml")

	if err := v.ReadConfig(bytes.NewBufferString(content)); err != nil {
		log.Fatal("配置解析失败:", err)
		return nil
	}

	var trans TransConf
	err = v.Unmarshal(&trans)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	return &trans
}

func getConfigFromNacos(env string) (string, error) {
	server, port, namespace, user, pass, group, dataId := parseNacosDSN(env)

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: server,
			Port:   port,
			Scheme: "http",
		},
	}

	clientConfig := constant.ClientConfig{
		NamespaceId:         namespace,
		Username:            user,
		Password:            pass,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		CacheDir:            "./data/configCache",
	}

	configClient, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})
	if err != nil {
		log.Fatal("初始化失败:", err)
	}

	content, err := configClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		log.Fatal("拉取配置失败:", err)
	}
	return content, nil
}

func parseNacosDSN(env string) (server string, port uint64, ns, user, pass, group, dataId string) {
	dsn := os.Getenv(env)
	if dsn == "" {
		log.Fatalf("%s 环境变量未设置", env)
	}

	parts := strings.SplitN(dsn, "?", 2)
	host := parts[0]
	params := url.Values{}

	if len(parts) == 2 {
		params, _ = url.ParseQuery(parts[1])
	}

	hostParts := strings.Split(host, ":")
	server = hostParts[0]
	if len(hostParts) > 1 {
		p, _ := strconv.Atoi(hostParts[1])
		port = uint64(p)
	} else {
		port = 8848
	}

	ns = params.Get("namespace")
	if ns == "" {
		ns = "public"
	}

	user = params.Get("username")
	pass = params.Get("password")
	group = params.Get("group")
	dataId = params.Get("dataId")
	return
}
