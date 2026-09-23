package cron

import "context"

// StopCronTasks 有界地等待定时任务停机：任务可能正卡在一次群发上，
// 超时后不再阻塞进程退出（未删除的发布计划会在下次启动后重试）。返回 false 表示等待超时。
func StopCronTasks(ctx context.Context, crons []Cron) bool {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, c := range crons {
			c.StopCronTask()
		}
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}
