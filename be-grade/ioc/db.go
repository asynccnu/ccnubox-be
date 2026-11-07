package ioc

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/infra"
	"gorm.io/gorm"
)

// migrationTimeout 上线 DDL（INPLACE、LOCK=NONE）在大表上的预算；
// 单实例部署，迁移完成前容器不对外服务，超时则启动失败由部署流程暴露。
const migrationTimeout = 15 * time.Minute

func InitDB(cfg *conf.InfraConf) *gorm.DB {
	db := infra.InitMysql(cfg.Mysql)
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()
	// 启动时执行结构迁移（幂等），使 DDL 随正常 CI/CD 部署自动完成
	err := dao.MigrateGradeNotifications(ctx, db)
	if err != nil {
		panic(err)
	}
	return db
}
