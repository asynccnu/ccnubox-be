package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"golang.org/x/sync/singleflight"
)

type ClassUsecase struct {
	conf *conf.ServerConf

	classInfoRepo  biz.ClassInfoRepo
	refreshLogRepo biz.RefreshLogRepo
	jxbRepo        biz.JxbRepo
	ccnu           biz.CCNUService
	crawler        biz.ClassCrawler

	delayQue biz.DelayQueue
	sfGroup  singleflight.Group
}

// 将实现暴露出的主函数与流程里使用到的工具函数分开放在两个文件里提高可读性
func (cluc *ClassUsecase) GetClasses(ctx context.Context, stuID, year, semester string, refresh bool) ([]*model.ClassInfoBO, *time.Time, error) {
	// 能返回到最上层的错误，就统一在最上层打错误日志
	logh := logger.From(ctx).With(
		logger.String("stu_id", stuID),
		logger.String("year", year),
		logger.String("semester", semester),
		logger.Any("refresh", refresh),
	)
	ctx = logger.WithLogger(ctx, logh) // 把当前带额外字段的 logger 写入上下文

	currentTime := time.Now()

	waitCrawTime := time.Duration(cluc.conf.ClassListConf.WaitCrawTime) * time.Millisecond
	refreshInterval := time.Duration(cluc.conf.ClassListConf.RefreshInterval) * time.Millisecond

	// 1. 本地查询阶段
	localClasses, localLastRefreshTime, localErr := cluc.loadLocal(ctx, stuID, year, semester)

	// 希望首次爬虫时间更长
	if localLastRefreshTime == nil {
		waitCrawTime = max(waitCrawTime, 15*time.Second)
	}

	// 2. 状态检查，决定从哪里获取课程数据
	action, refreshLog, waitBudget := cluc.decideRefreshAction(ctx, stuID, year, semester, refresh, localErr, refreshInterval, waitCrawTime)

	if action == model.ActionReturnLocal {
		logh.Infof("return local classes, last_refresh=%v", localLastRefreshTime)
		return localClasses, localLastRefreshTime, nil
	}

	if action == model.ActionWaitPending && refreshLog != nil {
		// waited 返回的是 等待时间 若超过设定值，waited就是超时值
		logh.Infof("wait pending refresh log id=%d, budget=%v", refreshLog.ID, waitBudget)
		readyLog, waited := cluc.waitPending(ctx, refreshLog.ID, waitBudget)

		if readyLog != nil && readyLog.IsReady() {
			// 刷新的课表保存到本地了，从本地拿就好了
			newLocalClassInfo, err := cluc.classInfoRepo.GetClassesFromLocal(ctx, stuID, year, semester)
			if err != nil {
				logh.Errorf("fetch class from local error=%v", err)
				return localClasses, localLastRefreshTime, nil
			}
			return newLocalClassInfo, &readyLog.UpdatedAt, nil
		}

		// 刷新课表失败
		// 如果等的时间不长（小于一秒），可以发起爬虫，阻塞等待新爬虫返回数据，消耗的时间代价不多
		// 反之就得返回了
		if waited >= 1*time.Second {
			logh.Warnf("pending wait timeout, waited=%v, fallback to local", waited)
			return localClasses, localLastRefreshTime, nil
		}

		// 若时间超过一秒或获取爬虫失败
	}
	logh.Infof("start crawl, waitCrawTime=%v", waitCrawTime)
	requestKey := fmt.Sprintf("craw:%s:%s:%s", stuID, year, semester)

	res, err := cluc.doCrawlWithSingleflight(ctx, requestKey, stuID, year, semester, localClasses, currentTime)
	if err == nil && res != nil {
		return res, &currentTime, nil
	}
	if err != nil {
		logh.Errorf("crawl error err=%v", err)
	}

	return localClasses, localLastRefreshTime, nil
}
