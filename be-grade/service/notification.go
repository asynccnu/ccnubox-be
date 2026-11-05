package service

import (
	"context"
	"errors"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NotificationPublisher struct {
	dao      dao.NotificationDAO
	feed     feedv1.FeedServiceClient
	l        logger.Logger
	interval time.Duration
	timeout  time.Duration
	batch    int
}

func NewNotificationPublisher(d dao.NotificationDAO, feed feedv1.FeedServiceClient, l logger.Logger, cfg *conf.ServerConf) *NotificationPublisher {
	p := &NotificationPublisher{dao: d, feed: feed, l: l, interval: 5 * time.Second, timeout: 3 * time.Second, batch: 50}
	if cfg != nil && cfg.NotificationConf != nil {
		c := cfg.NotificationConf
		if c.PollSeconds > 0 {
			p.interval = time.Duration(c.PollSeconds) * time.Second
		}
		if c.RPCSeconds > 0 {
			p.timeout = time.Duration(c.RPCSeconds) * time.Second
		}
		if c.BatchSize > 0 {
			p.batch = c.BatchSize
		}
	}
	return p
}

// Run 由 App 持有生命周期；单实例、单协程轮询，不增加租约或 DLQ。
func (p *NotificationPublisher) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		p.publishDue(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *NotificationPublisher) publishDue(ctx context.Context) {
	// 查询与状态回写也有独立超时，停机取消能传递到数据库操作。
	queryCtx, cancel := context.WithTimeout(ctx, p.timeout)
	rows, err := p.dao.FindDue(queryCtx, time.Now().Unix(), p.batch)
	cancel()
	if err != nil {
		p.l.WithContext(ctx).Error("成绩通知查询失败", logger.Error(err))
		return
	}
	for _, row := range rows {
		if ctx.Err() != nil {
			return
		}
		event, err := domain.ParseGradeNotification(row)
		invalid := err != nil
		if err == nil {
			rpcCtx, cancel := context.WithTimeout(ctx, p.timeout)
			resp, rpcErr := p.feed.PublicFeedEvent(rpcCtx, &feedv1.PublicFeedEventReq{StudentId: row.StudentID, Event: event})
			cancel()
			err = rpcErr
			invalid = status.Code(err) == codes.InvalidArgument
			if err == nil && (resp == nil || (resp.Status != feedv1.PublishStatus_ACCEPTED && resp.Status != feedv1.PublishStatus_DUPLICATE)) {
				err = errors.New("unexpected grade notification publish status")
			}
			if err == nil {
				p.l.WithContext(ctx).Info("成绩通知发布结果", logger.Int64("id", row.ID), logger.String("status", resp.Status.String()))
			}
		}
		if ctx.Err() != nil {
			// 退出时不改变任务，下次启动继续使用原 key 和 payload。
			return
		}
		writeCtx, cancel := context.WithTimeout(ctx, p.timeout)
		if err == nil {
			err = p.dao.MarkPublished(writeCtx, row.ID)
		} else {
			// 只存受控摘要，不把下游错误中的潜在敏感响应直接持久化。
			summary := "feed publish failed: " + status.Code(err).String()
			if invalid {
				summary = "grade notification validation failed"
			}
			p.l.WithContext(ctx).Warn("成绩通知未发布，已保留原事件", logger.Int64("id", row.ID), logger.String("reason", summary))
			next := time.Now().Add(notificationBackoff(row.Attempts + 1)).Unix()
			err = p.dao.MarkFailed(writeCtx, row.ID, row.Attempts+1, next, invalid, summary)
		}
		cancel()
		if err != nil {
			p.l.WithContext(ctx).Error("成绩通知状态回写失败，将使用原 key 重试", logger.Int64("id", row.ID), logger.Error(err))
		}
	}
	statsCtx, cancel := context.WithTimeout(ctx, p.timeout)
	stats, err := p.dao.Stats(statsCtx)
	cancel()
	if err != nil {
		p.l.WithContext(ctx).Warn("成绩通知积压统计失败", logger.Error(err))
	} else if stats.Pending > 0 || stats.Invalid > 0 {
		p.l.WithContext(ctx).Warn("成绩通知积压", logger.Int64("pending", stats.Pending), logger.Int64("invalid", stats.Invalid), logger.Int64("oldest_pending_at", stats.Oldest))
	}
}

func notificationBackoff(attempts int64) time.Duration {
	delay := 5 * time.Second
	for i := int64(1); i < attempts && delay < 5*time.Minute; i++ {
		delay *= 2
	}
	if delay > 5*time.Minute {
		return 5 * time.Minute
	}
	return delay
}
