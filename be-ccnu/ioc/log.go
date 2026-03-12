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

	return logger.NewFilterLogger(res,
		logger.FilterFunc(func(level logger.Level, key, val string) (string, bool) {
			// 只要包含 password 关键字，不论在哪个字段，都尝试脱敏
			if !strings.Contains(val, "password") {
				return val, false
			}

			// 处理结构化/JSON 风格: password:"xxx"
			masked := passwordReg.ReplaceAllString(val, `$1***$3`)

			// 处理 SQL 风格: `password` = 'xxx'
			masked = passwordSQLReg.ReplaceAllString(masked, `$1***$3`)

			return masked, true
		}),
	)
}
