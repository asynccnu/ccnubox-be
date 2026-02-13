package ioc

import (
	"regexp"
	"strings"

	"github.com/asynccnu/ccnubox-be/be-ccnu/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/log"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

var passwordReg = regexp.MustCompile(`(password:")([^"]*)(")`)

var passwordSQLReg = regexp.MustCompile(
	"(`password`\\s*=\\s*')([^']*)(')",
)

func InitLogger(cfg *conf.ServerConf) logger.Logger {
	res := log.InitLogger(cfg.Log, 4)
	// 过滤敏感信息
	return logger.NewFilterLogger(res, logger.FilterKey("password"), logger.FilterFunc(func(level logger.Level, key, val string) (string, bool) {
		if level < logger.INFO || key != "request" {
			return val, false
		}

		if !strings.Contains(val, "password:") {
			return val, false
		}

		masked := passwordReg.ReplaceAllString(val, `$1***$3`)
		return masked, true
	}), logger.FilterFunc(func(level logger.Level, key, val string) (string, bool) {
		if key != "args" {
			return val, false
		}

		if !strings.Contains(val, "password") {
			return val, false
		}

		masked := passwordSQLReg.ReplaceAllString(val, `$1***$3`)
		return masked, true
	}))
}
