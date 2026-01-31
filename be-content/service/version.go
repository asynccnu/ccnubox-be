package service

import (
	"context"
	"bytes"
	"fmt"
	"os"

	"github.com/asynccnu/ccnubox-be/be-content/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/nacosx"
	"github.com/spf13/viper"
)

type VersionService interface {
	GetVersion(ctx context.Context) string
	Refresh(ctx context.Context) error
}

type versionService struct {
	version string
	l       logger.Logger
}

func NewVersionService(cfg *conf.ServerConf, l logger.Logger) VersionService {
	return &versionService{
		version: cfg.Version.Version,
		l:       l,
	}
}

func (s *versionService) GetVersion(ctx context.Context) string {
	return s.version
}

func (s *versionService) Refresh(ctx context.Context) error {
	content, err := nacosx.GetConfigFromNacos(conf.ServerEnv)
	if err != nil || content == "" {
		paths := []string{
			"./config/config.yaml",
			"./config.yaml",
		}
		for _, p := range paths {
			data, e := os.ReadFile(p)
			if e == nil {
				content = string(data)
				break
			}
			err = e
		}
		if content == "" {
			s.l.Error("更新版本失败", logger.Error(err))
			return fmt.Errorf("刷新版本失败: %v", err)
		}
	}

	v := viper.New()
	v.SetConfigType("yaml")
	if e := v.ReadConfig(bytes.NewBufferString(content)); e != nil {
		s.l.Error("解析配置失败", logger.Error(e))
		return e
	}
	ver := v.GetString("version.version")
	if ver == "" {
		return fmt.Errorf("配置缺少版本字段")
	}
	s.version = ver
	s.l.Info("版本已更新", logger.String("version", ver))
	return nil
}
