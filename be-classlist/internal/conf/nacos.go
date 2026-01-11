package conf

import (
	"bytes"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/spf13/viper"
)

const (
	ClassList = "CCNUBOX_NACOS_CLASSLIST"
)

func InitBootstrapFromNacos() *Bootstrap {
	content, err := getConfigFromNacos(ClassList)
	if err != nil {
		log.Println(err)
		localPath := "./configs/config-example.yaml"
		fileContent, ferr := os.ReadFile(localPath)
		if ferr != nil {
			log.Fatalf("读取本地配置失败: %v", ferr)
			return nil
		}
		content = string(fileContent)
	}
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBufferString(content)); err != nil {
		log.Fatal("解析失败:", err)
		return nil
	}
	var bc Bootstrap
	if err := v.Unmarshal(&bc); err != nil {
		log.Fatal(err)
		return nil
	}
	return &bc
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
		log.Fatalf("%s 未设置", env)
	}
	parts := strings.SplitN(dsn, "?", 2)
	host := parts[0]
	params := map[string]string{}
	if len(parts) == 2 {
		for _, kv := range strings.Split(parts[1], "&") {
			if kv == "" {
				continue
			}
			p := strings.SplitN(kv, "=", 2)
			k := p[0]
			v := ""
			if len(p) == 2 {
				v = p[1]
			}
			params[k] = v
		}
	}
	hostParts := strings.Split(host, ":")
	server = hostParts[0]
	if len(hostParts) > 1 {
		p, _ := strconv.Atoi(hostParts[1])
		port = uint64(p)
	} else {
		port = 8848
	}
	ns = params["namespace"]
	if ns == "" {
		ns = "public"
	}
	user = params["username"]
	pass = params["password"]
	group = params["group"]
	dataId = params["dataId"]
	return
}
