package transaction

import (
	"context"

	"gorm.io/gorm"
)

type ContextTxKey struct{}

func InTx(mysql *gorm.DB, ctx context.Context, fn func(ctx context.Context) error) error {
	return mysql.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将tx放入到ctx中
		ctx = context.WithValue(ctx, ContextTxKey{}, tx)
		return fn(ctx)
	})
}
