package ioc

import "github.com/asynccnu/ccnubox-be/be-ccnu/conf"

func ProvideLibrarySecret(cfg *conf.ServerConf) string {
	return cfg.Crypto.Secret
}
