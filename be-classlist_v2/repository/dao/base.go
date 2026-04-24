package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/pkg/transaction"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/gorm"
)

type BaseDAO struct {
	db  *gorm.DB
	log logger.Logger
}

func NewBaseDAO(db *gorm.DB, l logger.Logger) (BaseDAO, func(), error) {
	cleanup := func() {
		l.Info("closing mysql resources")
	}

	return BaseDAO{
		db:  db,
		log: l,
	}, cleanup, nil
}

// mysql 写成私有字段，使得取数据库使用时必须得使用该逻辑，保证事务 db 一定能被取出
func (dao *BaseDAO) GetDB(ctx context.Context) *gorm.DB {
	tx, ok := ctx.Value(transaction.ContextTxKey{}).(*gorm.DB)
	if ok && tx != nil {
		return tx
	}

	return dao.db.WithContext(ctx)
}
