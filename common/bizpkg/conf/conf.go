package conf

import (
	"bytes"
	"log"
	"os"

	"github.com/asynccnu/ccnubox-be/common/pkg/nacosx"
	"github.com/spf13/viper"
)

const Infra = "CCNUBOX_INFRA_NACOS_DSN"

// InitInfraConfig 用来快速初始化infra的配置
func InitInfraConfig(paths ...string) *InfraConf {
	if len(paths) == 0 {
		paths = []string{"./config-infra.yaml"}
	}
	return InitConfig[InfraConf](Infra, paths...)
}

func InitConfig[T any](env string, localPaths ...string) *T {
	var content string

	// 1. 先从 nacos 读取
	cfg, err := nacosx.GetConfigFromNacos(env)
	if err == nil && cfg != "" {
		content = cfg
	} else {
		log.Printf("nacos 配置获取失败: %v，尝试读取本地配置\n", err)

		// 2. 本地路径兜底
		paths := localPaths
		if len(paths) == 0 {
			paths = []string{
				"./config/config.yaml",
				"./config.yaml",
			}
		}

		var fileErr error
		for _, path := range paths {
			data, e := os.ReadFile(path)
			if e == nil {
				content = string(data)
				log.Printf("使用本地配置文件: %s\n", path)
				break
			}
			fileErr = e
		}

		if content == "" {
			log.Fatalf("Nacos 失败，且本地配置文件读取失败: %v", fileErr)
		}
	}

	// 3. 解析配置
	v := viper.New()
	v.SetConfigType("yaml")

	if err := v.ReadConfig(bytes.NewBufferString(content)); err != nil {
		log.Fatalf("配置文件解析失败: %v", err)
	}

	var t T
	if err := v.Unmarshal(&t); err != nil {
		log.Fatalf("配置反序列化失败: %v", err)
	}

	return &t
}
