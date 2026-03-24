package ioc

import "github.com/asynccnu/ccnubox-be/be-library/conf"

func InitSecret(cfg *conf.ServerConf) string {
	return cfg.Crypto.Secret
}
