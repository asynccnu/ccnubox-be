package ioc

import (
	"context"
	"errors"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
)

// otelShutdownTimeout 为 OTel 关闭提供独立超时。
// 需覆盖 OTLP exporter 的 5 秒发送超时，给 BatchSpanProcessor 留出完成 flush 的时间。
const otelShutdownTimeout = 10 * time.Second

func InitShutdown(otelShutdown OTelShutdownFunc, db *gorm.DB, etcd *clientv3.Client) func(context.Context) error {
	return func(ctx context.Context) error {
		// 即使 ctx 已取消，也要继续执行关闭操作，避免留下未关闭的资源
		var results []error
		if err := ctx.Err(); err != nil {
			results = append(results, err)
		}
		if sqlDB, err := db.DB(); err != nil {
			results = append(results, err)
		} else if err := sqlDB.Close(); err != nil {
			results = append(results, err)
		}
		if etcd != nil {
			if err := etcd.Close(); err != nil {
				results = append(results, err)
			}
		}
		// OTel 最后关闭，确保任务停止后待导出的 Span 能完成 flush。
		// 使用独立的带超时关闭上下文：调用方 ctx 可能已取消，而 SDK 的
		// TracerProvider.Shutdown 会先标记 provider 已关闭，再因 ctx.Done() 提前返回，
		// 导致 BatchSpanProcessor.Shutdown 不执行、待导出 Span 丢失。
		if otelShutdown != nil {
			otelCtx, cancel := context.WithTimeout(context.Background(), otelShutdownTimeout)
			defer cancel()
			if err := otelShutdown(otelCtx); err != nil {
				results = append(results, err)
			}
		}
		return errors.Join(results...)
	}
}
