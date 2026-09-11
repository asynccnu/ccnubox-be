package ioc

import "github.com/asynccnu/ccnubox-be/be-library/conf"

func InitSecret(cfg *conf.ServerConf) string {
	if cfg == nil || cfg.Crypto == nil {
		return ""
	}
	return cfg.Crypto.Secret
}
