package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"golang.org/x/sync/singleflight"
)

type ClassUsecase struct {
	conf *conf.ServerConf
	log  logger.Logger

	classRepo      biz.ClassRepo
	refreshLogRepo biz.RefreshLogRepo
	jxbRepo        biz.JxbRepo
	ccnu           biz.CCNUService
	crawler        biz.ClassCrawler

	delayQue biz.DelayQueue
	sfGroup  singleflight.Group
}

func NewClassUsecase(
	conf *conf.ServerConf,
	cla biz.ClassRepo,
	re biz.RefreshLogRepo,
	jxb biz.JxbRepo,
	ccnu biz.CCNUService,
	crawler biz.ClassCrawler,
	queue biz.DelayQueue,
	l logger.Logger,
) *ClassUsecase {
	return &ClassUsecase{
		conf:           conf,
		log:            l,
		classRepo:      cla,
		refreshLogRepo: re,
		jxbRepo:        jxb,
		ccnu:           ccnu,
		crawler:        crawler,
		delayQue:       queue,
	}
}

// 将实现暴露出的主函数与流程里使用到的工具函数分开放在两个文件里提高可读性
func (cluc *ClassUsecase) GetClasses(ctx context.Context, stuID, year, semester string, refresh bool) ([]*model.ClassInfoBO, *time.Time, error) {
	// 能返回到最上层的错误，就统一在最上层打错误日志
	logh := cluc.log.WithContext(ctx).With(
		logger.String("stu_id", stuID),
		logger.String("year", year),
		logger.String("semester", semester),
		logger.Any("refresh", refresh),
	)

	currentTime := time.Now()

	waitCrawTime := time.Duration(cluc.conf.ClassListConf.WaitCrawTime) * time.Millisecond
	refreshInterval := time.Duration(cluc.conf.ClassListConf.RefreshInterval) * time.Millisecond

	// 1. 本地查询阶段
	localClasses, localLastRefreshTime, localErr := cluc.loadLocal(ctx, stuID, year, semester)
	if localErr != nil {
		logh.Errorf("load local failed: %+v", localErr)
	}

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
			newLocalClassInfo, err := cluc.classRepo.GetClassesFromLocal(ctx, stuID, year, semester)
			if err != nil {
				logh.Errorf("fetch class from local failed: %+v", err)
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
		logh.Errorf("crawl failed: %+v", err)
	}

	return localClasses, localLastRefreshTime, nil
}

func (cluc *ClassUsecase) AddClass(ctx context.Context, stuID string, info *model.ClassInfoBO) error {
	logh := cluc.log.WithContext(ctx)

	sc := &model.StudentCourseBO{
		StuID:           stuID,
		ClaID:           info.ID,
		Year:            info.Year,
		Semester:        info.Semester,
		IsManuallyAdded: !info.MetaData.IsOfficial,
		Note:            info.MetaData.Note,
	}

	// 判断这个课程是否与现成课程发生冲突
	conflict, err := cluc.hasScheduleConflict(ctx, stuID, info)
	if err != nil {
		return err
	}
	if conflict {
		logh.Error("class schedule conflict",
			logger.String("stu_id", stuID),
			logger.String("year", info.Year),
			logger.String("semester", info.Semester),
			logger.String("class_id", info.ID),
			logger.Int64("day", info.Day),
			logger.String("class_when", info.ClassWhen),
			logger.Int64("weeks", info.Weeks),
		)
		return errcode.ErrClassScheduleConflict
	}

	return cluc.classRepo.AddClass(ctx, stuID, info.Year, info.Semester, info, sc)
}
