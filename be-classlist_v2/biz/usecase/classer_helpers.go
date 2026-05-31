package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
	classTool "github.com/asynccnu/ccnubox-be/be-classlist_v2/pkg/tool"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/tool"
)

// 统一本地查询逻辑 GetClassesFromLocal + GetLastRefreshTime
func (cluc *ClassUsecase) loadLocal(ctx context.Context, stuID, year, semester string) (classes []*model.ClassInfoBO, lastRefresh *time.Time, err error) {
	logh := cluc.log.WithContext(ctx)
	// 这里处理的是除了 获取的数据在数据库不存在 以外的错误，获取的数据在数据库不存在时 lastRefresh 返回为 nil
	lastRefresh, err = cluc.refreshLogRepo.GetLastRefreshTime(ctx, stuID, year, semester, model.Ready, time.Now())
	if err != nil {
		logh.Errorf("GetLastRefreshTime failed: %+v", err)
	}

	classes, err = cluc.classRepo.GetClassesFromLocal(ctx, stuID, year, semester)
	if err != nil {
		logh.Errorf("GetClassesFromLocal failed: %+v", err)
	}

	return classes, lastRefresh, err
}

// 把是否刷新/是否pending/是否最近已刷新 这些判断集中在一起
// 决定课程数据来源的状态机
func (cluc *ClassUsecase) decideRefreshAction(ctx context.Context, stuID, year, semester string, refresh bool, localErr error, refreshInterval, waitCrawTime time.Duration) (action model.RefreshAction, refreshLog *model.ClassRefreshLogBO, waitBudget time.Duration) {
	logh := cluc.log.WithContext(ctx)
	now := time.Now()

	// 不要求更新且本地获取没有错误 则从本地获取课程
	if !refresh && localErr == nil {
		return model.ActionReturnLocal, nil, 0
	}

	// 获取最新的课程刷新 Log，若没有 Log 说明没保存过课程，则执行爬虫
	latestLog, err := cluc.refreshLogRepo.SearchNewestRefreshLog(ctx, stuID, year, semester, now)
	if err != nil || latestLog == nil {
		logh.Infof("first refresh or fetch log: %+v", err)
		return model.ActionStartCrawl, nil, 0
	}

	// 若上一次的刷新操作的时间还没过时间间隔（最近刷新过）
	// 则检查刷新操作的状态
	if latestLog.UpdatedAt.After(now.Add(-refreshInterval)) {

		// 刷新已完成
		// 从本地拿课程
		if latestLog.IsReady() {
			return model.ActionReturnLocal, latestLog, 0
		}

		// 刷新还在执行
		// 等待刷新
		if latestLog.IsPending() {
			return model.ActionWaitPending, latestLog, waitCrawTime / 2
		}
	}

	// 超过刷新时间间隔了喵
	return model.ActionStartCrawl, latestLog, 0
}

// 轮询 pending 状态 直到 ready 或 超时
func (cluc *ClassUsecase) waitPending(ctx context.Context, refreshLogID uint64, waitBudget time.Duration) (classLog *model.ClassRefreshLogBO, waited time.Duration) {
	start := time.Now()
	for {
		// 若请求取消或超时，直接返回
		if ctx.Err() != nil {
			return classLog, time.Since(start)
		}

		// 若超时，返回 classLog（大概率为空）
		if time.Since(start) >= waitBudget {
			return classLog, time.Since(start)
		}

		// 只要状态不再是 pending，就返回
		// 可能是 ready，也可能是 failed
		classLog, _ = cluc.refreshLogRepo.GetRefreshLogByID(ctx, refreshLogID)
		if classLog == nil || !classLog.IsPending() {
			return classLog, time.Since(start)
		}

		select {
		case <-ctx.Done():
			return classLog, time.Since(start)
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func (cluc *ClassUsecase) hasScheduleConflict(ctx context.Context, stuID string, info *model.ClassInfoBO) (bool, error) {
	logh := cluc.log.WithContext(ctx)

	// 判断这个课程是否存在，存在代表冲突
	if cluc.classRepo.AddedCourseExists(ctx, stuID, info.Year, info.Semester, info.ID) {
		logh.Error("class already exists",
			logger.String("stu_id", stuID),
			logger.String("year", info.Year),
			logger.String("semester", info.Semester),
			logger.String("class_id", info.ID),
		)
		return true, errcode.ErrClassIsExist
	}

	// 拉取本地课表检查是否有冲突
	classes, err := cluc.classRepo.GetClassesFromLocal(ctx, stuID, info.Year, info.Semester)
	if err != nil {
		if errors.Is(err, errcode.ErrClassNotFound) {
			return false, nil
		}
		return false, err
	}

	return cluc.hasScheduleConflictWithClasses(ctx, info, classes)
}

// 检查是否与现有课程有时间上的冲突
// 包装一层 addclass 和 updateclass 大家就可以一起使用底层函数了喵
func (cluc *ClassUsecase) hasScheduleConflictWithClasses(ctx context.Context, info *model.ClassInfoBO, classes []*model.ClassInfoBO) (bool, error) {
	return cluc.hasScheduleConflictWithClassesExcept(ctx, info, classes, "")
}

// 课程冲突检测函数
// 旧课还在旧课表里，需要排除 oldClassID
func (cluc *ClassUsecase) hasScheduleConflictWithClassesExcept(ctx context.Context, info *model.ClassInfoBO, classes []*model.ClassInfoBO, ignoredClassID string) (bool, error) {
	logh := cluc.log.WithContext(ctx)

	// 解析节次
	newSections, err := classTool.ParseClassSections(info.ClassWhen)
	if err != nil {
		return false, errcode.ErrParam
	}

	for _, classInfo := range classes {
		// 粗筛
		if classInfo == nil || classInfo.ID == ignoredClassID || classInfo.Day != info.Day || classInfo.Weeks&info.Weeks == 0 {
			continue
		}

		sections, err := classTool.ParseClassSections(classInfo.ClassWhen)
		if err != nil {
			logh.Warn("skip invalid existing class section",
				logger.String("class_id", classInfo.ID),
				logger.String("class_when", classInfo.ClassWhen),
			)
			continue
		}
		if sections&newSections != 0 {
			return true, nil
		}
	}
	return false, nil
}

// 筛选出与官方课程有冲突的自写课程id（若与官方课程有冲突的话会删除自选课程）
// 输入 官方课程 自写课程 输出 有冲突的自写课程id
// 在 getclass 的 merge 自写阶段使用
func (cluc *ClassUsecase) filterAddedClassesConflictingWithOfficial(ctx context.Context, officialClasses, addedClasses []*model.ClassInfoBO) ([]*model.ClassInfoBO, []string) {
	logh := cluc.log.WithContext(ctx)
	kept := make([]*model.ClassInfoBO, 0, len(addedClasses))
	conflictIDs := make([]string, 0)

	for _, added := range addedClasses {
		if added == nil {
			continue
		}
		conflict, err := cluc.hasScheduleConflictWithClasses(ctx, added, officialClasses)
		if err != nil {
			logh.Warn("skip invalid added class during official refresh conflict cleanup",
				logger.String("class_id", added.ID),
				logger.String("class_when", added.ClassWhen),
				logger.Error(err),
			)
			kept = append(kept, added)
			continue
		}
		if conflict {
			logh.Warn("delete added class because official class conflicts",
				logger.String("class_id", added.ID),
				logger.String("class_when", added.ClassWhen),
				logger.Int64("day", added.Day),
				logger.Int64("weeks", added.Weeks),
			)
			conflictIDs = append(conflictIDs, added.ID)
			continue
		}
		kept = append(kept, added)
	}

	return kept, conflictIDs
}

// 包一层 singleflight + crawClass 调用 + 超时处理
func (cluc *ClassUsecase) doCrawlWithSingleflight(ctx context.Context, key string, stuID, year, semester string, local []*model.ClassInfoBO, logTime time.Time) ([]*model.ClassInfoBO, error) {
	v, err, _ := cluc.sfGroup.Do(key, func() (interface{}, error) {
		res, err := cluc.crawMergedClass(ctx, stuID, year, semester, logTime, local, true)
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, fmt.Errorf("crawler returned empty result")
		}
		return res, nil
	})

	if err != nil {
		return nil, err
	}

	res, ok := v.([]*model.ClassInfoBO)
	if !ok {
		return nil, fmt.Errorf("crawler returned unexpected result type")
	}
	return res, nil
}

// 爬取课表并合并自写课程
func (cluc *ClassUsecase) crawMergedClass(ctx context.Context, stuID, year, semester string, logTime time.Time, localClassInfo []*model.ClassInfoBO, mergeAdd bool) ([]*model.ClassInfoBO, error) {
	logh := cluc.log.WithContext(ctx)

	metaMap := make(map[string]model.ClassMetaDataBO, len(localClassInfo))
	for _, lc := range localClassInfo {
		metaMap[lc.ID] = lc.MetaData
	}

	// 插入刷新日志
	logID, err := cluc.refreshLogRepo.InsertRefreshLog(ctx, stuID, year, semester, model.Pending, logTime)
	if err != nil {
		return nil, err
	}

	// 执行爬虫
	crawClassInfos, crawScs, _, err := cluc.getCourseFromCrawler(ctx, stuID, year, semester)
	if err != nil {
		_ = cluc.refreshLogRepo.UpdateRefreshLogStatus(ctx, logID, model.Failed)
		// 重试
		_ = cluc.sendRetryMsg(ctx, stuID, year, semester)
		return nil, err
	}

	// 标记官方课程和标记备注
	for _, ci := range crawClassInfos {
		if ci == nil {
			continue
		}
		ci.MetaData.IsOfficial = true
		if meta, ok := metaMap[ci.ID]; ok {
			ci.MetaData.Note = meta.Note
		}
	}

	// 添加自添加课程的备注
	for _, sc := range crawScs {
		if sc == nil {
			continue
		}
		if meta, ok := metaMap[sc.ClaID]; ok {
			sc.Note = meta.Note
		}
	}

	jxbIDs := extractJxb(crawClassInfos)
	err = cluc.classRepo.SaveClass(ctx, stuID, year, semester, crawClassInfos, crawScs)
	if err != nil {
		_ = cluc.refreshLogRepo.UpdateRefreshLogStatus(ctx, logID, model.Failed)
		_ = cluc.sendRetryMsg(ctx, stuID, year, semester)
		return nil, err
	}

	if !mergeAdd {
		_ = cluc.refreshLogRepo.UpdateRefreshLogStatus(ctx, logID, model.Ready)
		_ = cluc.jxbRepo.SaveJxb(ctx, stuID, jxbIDs)
		return crawClassInfos, nil
	}

	addedInfos, err := cluc.classRepo.GetAddedClasses(ctx, stuID, year, semester)
	if err != nil {
		// 因为这里是非关键路径，失败了也不影响主流程，所以这里可以就地打日志而不是从上一层返回
		logh.Warn("failed to find added class in the database")
	}

	addedInfos, conflictAddedIDs := cluc.filterAddedClassesConflictingWithOfficial(ctx, crawClassInfos, addedInfos)
	if len(conflictAddedIDs) > 0 {
		err := cluc.classRepo.DeleteAddedClasses(ctx, stuID, year, semester, conflictAddedIDs)
		if err != nil {
			_ = cluc.refreshLogRepo.UpdateRefreshLogStatus(ctx, logID, model.Failed)
			_ = cluc.sendRetryMsg(ctx, stuID, year, semester)
			return nil, err
		}
	}

	_ = cluc.refreshLogRepo.UpdateRefreshLogStatus(ctx, logID, model.Ready)
	_ = cluc.jxbRepo.SaveJxb(ctx, stuID, jxbIDs)

	crawClassInfos = append(crawClassInfos, addedInfos...)
	return crawClassInfos, nil
}

func (cluc *ClassUsecase) getCourseFromCrawler(ctx context.Context, stuID string, year string, semester string) ([]*model.ClassInfoBO, []*model.StudentCourseBO, int, error) {
	logh := cluc.log.WithContext(ctx)
	crawSuccess := true
	defer func(currentTime time.Time) {
		logh.Info(fmt.Sprintf("[%v %v %v] getCourseFromCrawler(success:%v) took %v", stuID, year, semester, crawSuccess, time.Since(currentTime)))
	}(time.Now())

	cookie, err := func() (string, error) {
		cookieSuccess := true
		defer func(currentTime time.Time) {
			logh.Info(fmt.Sprintf("Get cookie (stu_id:%v,success:%v) from other service,cost %v", stuID, cookieSuccess, time.Since(currentTime)))
		}(time.Now())

		cookie, err := cluc.ccnu.GetCookie(ctx, stuID)
		if err != nil {
			cookieSuccess = false
			logh.Errorf("get cookie from ccnu failed stu_id=%s: %+v", stuID, err)
		}
		return cookie, err
	}()
	if err != nil {
		crawSuccess = false
		return nil, nil, -1, err
	}

	if len(cookie) == 0 {
		crawSuccess = false
		logh.Error(fmt.Sprintf("the cookie from other service is empty for stu_id:%v", stuID))
		return nil, nil, -1, fmt.Errorf("the cookie from other service is empty for stu_id:%v", stuID)
	}

	var stu biz.Student

	sType := tool.ParseStudentType(stuID)
	switch sType {
	case tool.UnderGraduate:
		stu = &biz.Undergraduate{}
	case tool.PostGraduate:
		stu = &biz.GraduateStudent{}
	default:
		return nil, nil, -1, fmt.Errorf("the type of student isn't undergraduate")
	}

	ci, sc, sum, err := func() ([]*model.ClassInfoBO, []*model.StudentCourseBO, int, error) {
		defer func(currentTime time.Time) {
			logh.Info(fmt.Sprintf("craw class [%v,%v,%v] cost %v", stuID, year, semester, time.Since(currentTime)))
		}(time.Now())

		classinfos, scs, sum, err := stu.GetClass(ctx, stuID, year, semester, cookie, cluc.crawler)
		if err != nil {
			logh.Errorf("craw classlist stu_id=%s year=%s semester=%s failed: %+v", stuID, year, semester, err)
			return nil, nil, -1, err
		}
		if len(classinfos) == 0 || len(scs) == 0 {
			return nil, nil, -1, errors.New("no classinfos or scs found")
		}
		return classinfos, scs, sum, nil
	}()
	if err != nil {
		crawSuccess = false
		return nil, nil, -1, err
	}
	return ci, sc, sum, nil
}

// 发送重试消息
func (cluc *ClassUsecase) sendRetryMsg(ctx context.Context, stuID, year, semester string) error {
	logh := cluc.log.WithContext(ctx)

	retryInfo := map[string]string{
		"stu_id":   stuID,
		"year":     year,
		"semester": semester,
	}
	key := fmt.Sprintf("be-classlist-refresh-retry-%d", time.Now().UnixMilli())
	val, err := json.Marshal(&retryInfo)
	if err != nil {
		return err
	}
	err = cluc.delayQue.Send(ctx, []byte(key), val)
	if err != nil {
		logh.Errorf("delayQue.Send retry msg failed: %+v", err)
	}
	return err
}

func extractJxb(infos []*model.ClassInfoBO) []string {
	if len(infos) == 0 {
		return nil
	}
	Jxbmp := make(map[string]struct{})
	for _, classInfo := range infos {
		if classInfo.JxbId != "" {
			Jxbmp[classInfo.JxbId] = struct{}{}
		}
	}
	jxbIDs := make([]string, 0, len(Jxbmp))
	for k := range Jxbmp {
		jxbIDs = append(jxbIDs, k)
	}
	return jxbIDs
}
