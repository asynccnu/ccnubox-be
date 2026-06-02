package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/conf"
	classTool "github.com/asynccnu/ccnubox-be/be-classlist_v2/pkg/tool"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
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
	cluc := &ClassUsecase{
		conf:           conf,
		log:            l,
		classRepo:      cla,
		refreshLogRepo: re,
		jxbRepo:        jxb,
		ccnu:           ccnu,
		crawler:        crawler,
		delayQue:       queue,
	}
	cluc.startRetryConsumer()
	return cluc
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
		return errorx.Errorf("usecase.class.AddClass: class schedule conflict: %w", biz.ErrClassScheduleConflict)
	}

	return cluc.classRepo.AddClass(ctx, stuID, info.Year, info.Semester, info, sc)
}

func (cluc *ClassUsecase) DeleteClass(ctx context.Context, stuID, year, semester, classID string) error {
	logh := cluc.log.WithContext(ctx)

	classes, err := cluc.classRepo.GetClassesFromLocal(ctx, stuID, year, semester)
	if err != nil {
		return err
	}

	var target *model.ClassInfoBO
	for _, classInfo := range classes {
		if classInfo != nil && classInfo.ID == classID {
			target = classInfo
			break
		}
	}
	if target == nil {
		return errorx.Errorf("usecase.class.DeleteClass: classID=%s: %w", classID, biz.ErrStudentCourseNotFound)
	}
	if target.MetaData.IsOfficial {
		logh.Warn("reject deleting official class",
			logger.String("stu_id", stuID),
			logger.String("year", year),
			logger.String("semester", semester),
			logger.String("class_id", classID),
		)
		return errorx.Errorf("usecase.class.DeleteClass: reject official classID=%s: %w", classID, biz.ErrClassDeleteRejected)
	}

	if err := cluc.classRepo.DeleteAddedClasses(ctx, stuID, year, semester, []string{classID}); err != nil {
		logh.Error("delete added class failed",
			logger.String("stu_id", stuID),
			logger.String("year", year),
			logger.String("semester", semester),
			logger.String("class_id", classID),
			logger.Error(err),
		)
		return err
	}
	return nil
}

func (cluc *ClassUsecase) UpdateClass(ctx context.Context, stuID, year, semester, oldClassID string, name, durClass, where, teacher *string, weeks, day *int64, credit *float64) (string, error) {
	logh := cluc.log.WithContext(ctx)

	// 拿数据库数据
	classes, err := cluc.classRepo.GetClassesFromLocal(ctx, stuID, year, semester)
	if err != nil {
		return "", err
	}

	var oldInfo *model.ClassInfoBO
	for _, classInfo := range classes {
		if classInfo != nil && classInfo.ID == oldClassID {
			oldInfo = classInfo
			break
		}
	}
	if oldInfo == nil {
		return "", errorx.Errorf("usecase.class.UpdateClass: oldClassID=%s: %w", oldClassID, biz.ErrStudentCourseNotFound)
	}
	// 防御一下，正常情况不可能升级官方课程的，前端不会提供入口
	if oldInfo.MetaData.IsOfficial {
		logh.Warn("reject updating official class",
			logger.String("stu_id", stuID),
			logger.String("year", year),
			logger.String("semester", semester),
			logger.String("class_id", oldClassID),
		)
		return "", errorx.Errorf("usecase.class.UpdateClass: reject official classID=%s: %w", oldClassID, biz.ErrClassUpdateRejected)
	}

	newInfo := *oldInfo
	if name != nil {
		newInfo.Classname = *name
	}
	if durClass != nil {
		newInfo.ClassWhen = *durClass
	}
	if where != nil {
		newInfo.Where = *where
	}
	if teacher != nil {
		newInfo.Teacher = *teacher
	}
	if weeks != nil {
		newInfo.Weeks = *weeks
		newInfo.WeekDuration = classTool.FormatWeeks(classTool.ParseWeeks(*weeks))
	}
	if day != nil {
		newInfo.Day = *day
	}
	if credit != nil {
		newInfo.Credit = *credit
	}
	newInfo.UpdateID()

	if newInfo.ID != oldClassID && cluc.classRepo.AddedCourseExists(ctx, stuID, year, semester, newInfo.ID) {
		logh.Error("class already exists",
			logger.String("stu_id", stuID),
			logger.String("year", year),
			logger.String("semester", semester),
			logger.String("class_id", newInfo.ID),
		)
		return "", errorx.Errorf("usecase.class.UpdateClass: classID=%s: %w", newInfo.ID, biz.ErrClassAlreadyExists)
	}

	// 判定冲突
	conflict, err := cluc.hasScheduleConflictWithClassesExcept(ctx, &newInfo, classes, oldClassID)
	if err != nil {
		return "", err
	}
	if conflict {
		logh.Error("class schedule conflict",
			logger.String("stu_id", stuID),
			logger.String("year", year),
			logger.String("semester", semester),
			logger.String("old_class_id", oldClassID),
			logger.String("new_class_id", newInfo.ID),
			logger.Int64("day", newInfo.Day),
			logger.String("class_when", newInfo.ClassWhen),
			logger.Int64("weeks", newInfo.Weeks),
		)
		return "", errorx.Errorf("usecase.class.UpdateClass: class schedule conflict oldClassID=%s newClassID=%s: %w",
			oldClassID, newInfo.ID, biz.ErrClassScheduleConflict)
	}

	sc := &model.StudentCourseBO{
		StuID:           stuID,
		ClaID:           newInfo.ID,
		Year:            year,
		Semester:        semester,
		IsManuallyAdded: true,
		Note:            oldInfo.MetaData.Note,
	}
	if err := cluc.classRepo.UpdateAddedClass(ctx, stuID, year, semester, oldClassID, &newInfo, sc); err != nil {
		logh.Error("update added class failed",
			logger.String("stu_id", stuID),
			logger.String("year", year),
			logger.String("semester", semester),
			logger.String("old_class_id", oldClassID),
			logger.String("new_class_id", newInfo.ID),
			logger.Error(err),
		)
		return "", err
	}
	return newInfo.ID, nil
}

func (cluc *ClassUsecase) UpdateClassNote(ctx context.Context, stuID, year, semester, classID, note string) error {
	logh := cluc.log.WithContext(ctx)

	classes, err := cluc.classRepo.GetClassesFromLocal(ctx, stuID, year, semester)
	if err != nil {
		return err
	}

	var found bool
	for _, classInfo := range classes {
		if classInfo != nil && classInfo.ID == classID {
			found = true
			break
		}
	}
	if !found {
		return errorx.Errorf("usecase.class.UpdateClassNote: classID=%s: %w", classID, biz.ErrStudentCourseNotFound)
	}

	if err := cluc.classRepo.UpdateClassNote(ctx, stuID, year, semester, classID, note); err != nil {
		logh.Error("update class note failed",
			logger.String("stu_id", stuID),
			logger.String("year", year),
			logger.String("semester", semester),
			logger.String("class_id", classID),
			logger.Error(err),
		)
		return err
	}
	return nil
}

func (cluc *ClassUsecase) GetStuIdsByJxbId(ctx context.Context, jxbID string) ([]string, error) {
	stuIDs, err := cluc.jxbRepo.FindStuIdsByJxbId(ctx, jxbID)
	if err != nil {
		return []string{}, errorx.Errorf("usecase.class.GetStuIdsByJxbId: jxbID=%s: %w", jxbID, err)
	}
	if len(stuIDs) == 0 {
		return []string{}, errorx.Errorf("usecase.class.GetStuIdsByJxbId: jxbID=%s: %w", jxbID, biz.ErrGetStuIDsByJxbID)
	}
	return stuIDs, nil
}

func (cluc *ClassUsecase) GetClassNatures(ctx context.Context, stuID string) ([]string, error) {
	return cluc.classRepo.GetClassNatures(ctx, stuID)
}
