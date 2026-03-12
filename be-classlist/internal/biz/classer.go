package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/cronx"
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/tool"
)

type ClassUsecase struct {
	classRepo      ClassRepo
	crawler        ClassCrawler
	ccnu           CCNUServiceProxy
	jxbRepo        JxbRepo
	recycleBinRepo RecycleBinRepo
	delayQue       DelayQueue

	refreshLogRepo  RefreshLogRepo
	waitCrawTime    time.Duration
	waitUserSvcTime time.Duration
	refreshInterval time.Duration // 刷新间隔,当前时间距离上次刷新时间超过该值时,需要重新刷新

	sfGroup     singleflight.Group
	cronManager *cronx.Manager
}

func (cluc *ClassUsecase) Close() {
	if cluc.cronManager != nil {
		cluc.cronManager.Stop(context.Background())
	}
}

func NewClassUsecase(classRepo ClassRepo, crawler ClassCrawler,
	JxbRepo JxbRepo, Cs CCNUServiceProxy,
	delayQue DelayQueue, refreshLog RefreshLogRepo,
	recycleBinRepo RecycleBinRepo,
	cf *conf.Server, log logger.Logger,
) (*ClassUsecase, func()) {
	waitCrawTime := 1200 * time.Millisecond
	waitUserSvcTime := 10000 * time.Millisecond
	refreshInterval := 10 * time.Second

	if cf.WaitCrawTime > 0 {
		waitCrawTime = time.Duration(cf.WaitCrawTime) * time.Millisecond
	}
	if cf.WaitUserSvcTime > 0 {
		waitUserSvcTime = time.Duration(cf.WaitUserSvcTime) * time.Millisecond
	}
	if cf.RefreshInterval > 0 {
		refreshInterval = time.Duration(cf.RefreshInterval) * time.Second
	}

	cluc := &ClassUsecase{
		classRepo:       classRepo,
		crawler:         crawler,
		jxbRepo:         JxbRepo,
		delayQue:        delayQue,
		ccnu:            Cs,
		refreshLogRepo:  refreshLog,
		recycleBinRepo:  recycleBinRepo,
		waitCrawTime:    waitCrawTime,
		waitUserSvcTime: waitUserSvcTime,
		cronManager:     cronx.NewManager(log),
		refreshInterval: refreshInterval,
	}

	// 每周日自动为全部学员爬取课表
	err := cluc.cronManager.AddTask("pull_class_list", "0 3 * * 0", cluc.pullClassListTask)
	if err != nil {
		panic(err)
	}

	// 每天凌晨5点设置的课表缓存
	err = cluc.cronManager.AddTask("cache_class_list", "0 5 * * *", cluc.cacheClass)
	if err != nil {
		panic(err)
	}

	// 每个月 1 号 2 点清理回收站中过期的 zset 元素
	err = cluc.cronManager.AddTask("clean_recycle_bin", "0 2 1 * *", cluc.cleanRecycleBinTask)
	if err != nil {
		panic(err)
	}
	// 开启一个协程来处理重试消息
	go func() {
		if err := cluc.delayQue.Consume("be-classlist-refresh-retry", cluc.handleRetryMsg); err != nil {
			logger.GlobalLogger.Error(fmt.Sprintf("Error consuming retry message: %v", err))
		}
	}()

	return cluc, func() {
		cluc.Close()
	}
}

func (cluc *ClassUsecase) GetClasses(ctx context.Context, stuID, year, semester string, refresh bool) ([]*ClassInfoBO, *time.Time, error) {

	currentTime := time.Now() // 当前时间

	logh := logger.GetLoggerFromCtx(ctx)
	noExpireCtx := logger.WithLogger(context.Background(), logh)

	waitCrawTime := cluc.waitCrawTime // 等待爬虫的时间

	// 1. 本地查询阶段
	lastRefreshTime := cluc.refreshLogRepo.GetLastRefreshTime(ctx, stuID, year, semester, Ready, currentTime) // 获取上次刷新成功的时间
	localClassInfo, err := cluc.classRepo.GetClassesFromLocal(ctx, stuID, year, semester)                     // 获取本地课表

	// lastRefreshTime==nil说明这个学生在year-semester并没有爬取过,必须走爬虫
	if lastRefreshTime == nil {
		waitCrawTime = max(waitCrawTime, 15*time.Second)
	} else if !refresh && err == nil {
		return localClassInfo, lastRefreshTime, nil
	}

	// 2. 状态检查与轮询阶段

	// 查询最新的一条log
	refreshLog, err := cluc.refreshLogRepo.SearchNewestRefreshLog(ctx, stuID, year, semester, currentTime)
	// 如果有记录
	if err == nil && refreshLog != nil {
		if refreshLog.UpdatedAt.After(currentTime.Add(-cluc.refreshInterval)) {
			// 不久前已经爬取过,并且已经更新到数据库了,这里直接返回查询数据库的结果即可
			if refreshLog.IsReady() {
				return localClassInfo, lastRefreshTime, nil
			}
			// 如果是pending,说明正在爬取,我们等待一定时间,如果没有结果,则直接返回数据库的结果
			// 如果一段时间后是ready,我们重新走数据库
			if refreshLog.IsPending() {
				pollingTime := 0 * time.Second
				refreshLogID := refreshLog.ID
				// 轮询一段时间，直到当前这个refreshLog退出pending状态
				for pollingTime < waitCrawTime/2 && refreshLog != nil && refreshLog.IsPending() {
					refreshLog, _ = cluc.refreshLogRepo.GetRefreshLogByID(ctx, refreshLogID)
					time.Sleep(200 * time.Millisecond) // 显式休眠
					pollingTime += 200 * time.Millisecond
				}

				// 如果refreshLog是ready的，再走一遍数据库，就可以获取刚刚成功的爬虫的结果，而不用再发起一次爬虫请求
				if refreshLog != nil && refreshLog.IsReady() {
					newLocalClassInfo, err := cluc.classRepo.GetClassesFromLocal(ctx, stuID, year, semester)
					if err != nil {
						return localClassInfo, lastRefreshTime, nil
					}
					return newLocalClassInfo, &refreshLog.UpdatedAt, nil
				}
				// 如果等的时间不长（小于一秒），可以发起爬虫，消耗的时间代价不多
				// 反之就得返回了
				if pollingTime >= 1*time.Second {
					return localClassInfo, lastRefreshTime, nil
				}
			}
		}

	}

	// 3. SingleFlight 爬虫阶段

	requestKey := fmt.Sprintf("craw:%s:%s:%s", stuID, year, semester)

	// 使用 SingleFlight 封装爬取逻辑
	// v 是返回的结果，err 是错误
	v, err, _ := cluc.sfGroup.Do(requestKey, func() (interface{}, error) {

		resChan := make(chan []*ClassInfoBO, 1)
		go func() {
			result := cluc.crawClass(noExpireCtx, stuID, year, semester, currentTime, localClassInfo, true)
			resChan <- result
			close(resChan)
		}()

		select {
		case res := <-resChan:
			if res != nil {
				return res, nil
			}
			return nil, fmt.Errorf("crawler returned empty result")
		case <-time.After(waitCrawTime):
			return nil, fmt.Errorf("crawler timeout")
		}
	})

	// 如果 SingleFlight 成功获取结果
	if err == nil {
		if res, ok := v.([]*ClassInfoBO); ok {
			return res, &currentTime, nil
		}
	}

	// 如果爬取失败或超时，降级返回本地旧数据
	return localClassInfo, lastRefreshTime, nil
}

// 爬取课表并保存
func (cluc *ClassUsecase) crawClass(ctx context.Context, stuID, year, semester string, logTime time.Time, localClassInfo []*ClassInfoBO, mergeAdd bool) []*ClassInfoBO {
	logh := logger.GetLoggerFromCtx(ctx)

	metaMap := make(map[string]ClassMetaDataBO, len(localClassInfo))
	// 构建本地课程 ID -> MetaData 的映射，避免 O(n^2) 比较
	for _, lc := range localClassInfo {
		metaMap[lc.ID] = lc.MetaData
	}

	logID, err := cluc.refreshLogRepo.InsertRefreshLog(ctx, stuID, year, semester, Pending, logTime)
	if err != nil {
		logh.Errorf("failed to insert refresh log,param(%v,%v,%v)", stuID, year, semester)
		return nil
	}

	crawClassInfos, crawScs, _, err := cluc.getCourseFromCrawler(ctx, stuID, year, semester)
	if err != nil {
		_ = cluc.refreshLogRepo.UpdateRefreshLogStatus(ctx, logID, Failed)
		_ = cluc.sendRetryMsg(ctx, stuID, year, semester)
		return nil
	}

	// 爬取课表的note继承本地课表
	// 将本地备注合并到爬虫结果中
	for _, ci := range crawClassInfos {
		if ci == nil {
			continue
		}

		// 设置这个meta，是为了返回的结果的数据完整性
		ci.MetaData.IsOfficial = true
		if meta, ok := metaMap[ci.ID]; ok {
			ci.MetaData.Note = meta.Note
		}
	}

	// 将本地备注合并到学生课程信息中
	for _, sc := range crawScs {
		if sc == nil {
			continue
		}
		// sc.IsManuallyAdded这个在爬虫时已经设置了，这里不用动
		// 只需要把丢失的note设置即可
		if meta, ok := metaMap[sc.ClaID]; ok {
			sc.Note = meta.Note
		}
	}

	// 保存课表

	jxbIDs := extractJxb(crawClassInfos)
	err = cluc.classRepo.SaveClass(ctx, stuID, year, semester, crawClassInfos, crawScs)
	// 更新log状态
	if err != nil {
		_ = cluc.refreshLogRepo.UpdateRefreshLogStatus(ctx, logID, Failed)
		_ = cluc.sendRetryMsg(ctx, stuID, year, semester)
	} else {
		_ = cluc.refreshLogRepo.UpdateRefreshLogStatus(ctx, logID, Ready)
	}
	_ = cluc.jxbRepo.SaveJxb(ctx, stuID, jxbIDs)

	if !mergeAdd {
		return crawClassInfos
	}

	addedInfos, err := cluc.classRepo.GetAddedClasses(ctx, stuID, year, semester)
	if err != nil {
		logh.Warn("failed to find added class in the database")
	}

	crawClassInfos = append(crawClassInfos, addedInfos...)
	return crawClassInfos
}

func (cluc *ClassUsecase) AddClass(ctx context.Context, stuID string, info *ClassInfoBO) error {
	return cluc.addClass(ctx, stuID, info)
}

func (cluc *ClassUsecase) DeleteClass(ctx context.Context, stuID, year, semester, classId string) error {
	logh := logger.GetLoggerFromCtx(ctx).WithContext(ctx)

	classInfo, err := cluc.classRepo.GetSpecificClassInfo(ctx, stuID, year, semester, classId)
	if err != nil || classInfo == nil {
		return errcode.ErrClassFound
	}
	if classInfo.MetaData.IsOfficial {
		return errcode.ErrClassDelete
	}

	// 先添加到回收站
	err = cluc.recycleBinRepo.RecycleClass(ctx, stuID, year, semester, classId, classInfo)
	if err != nil {
		logh.Error(fmt.Sprintf("add class [%v] to recycle bin failed: %v", classId, err))
		return errcode.ErrClassDelete
	}

	// 删除课程
	err = cluc.classRepo.DeleteClass(ctx, stuID, year, semester, classInfo)
	if err != nil {
		logh.Error(fmt.Sprintf("delete classlist [%v] failed", classId))
		return errcode.ErrClassDelete
	}
	return nil
}

func (cluc *ClassUsecase) GetRecycledClassInfos(ctx context.Context, stuID, year, semester string) ([]*ClassInfoBO, error) {
	// 获取回收站的课程ID
	classInfos, err := cluc.recycleBinRepo.ListClasses(ctx, stuID, year, semester)
	if err != nil {
		return nil, err
	}
	return classInfos, nil
}

func (cluc *ClassUsecase) RecoverClassInfo(ctx context.Context, stuID, year, semester, classId string) error {
	classInfo, ok := cluc.recycleBinRepo.GetClass(ctx, stuID, year, semester, classId)
	if !ok {
		return errcode.ErrRecycleBinDoNotHaveIt
	}

	// 恢复数据库中的对应关系
	err := cluc.addClass(ctx, stuID, classInfo)
	if err != nil {
		return errcode.ErrRecover
	}

	// 删除回收站的对应ID
	err = cluc.recycleBinRepo.RemoveClass(ctx, stuID, year, semester, classId)
	if err != nil {
		return errcode.ErrRecover
	}
	return nil
}

func (cluc *ClassUsecase) GetSpecificClassInfo(ctx context.Context, stuID, year, semester, classId string) (*ClassInfoBO, error) {
	info, err := cluc.classRepo.GetSpecificClassInfo(ctx, stuID, year, semester, classId)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (cluc *ClassUsecase) UpdateClass(ctx context.Context, stuID, year, semester string, newClassInfo *ClassInfoBO, newSc *StudentCourse, oldClassId string) error {
	logh := logger.GetLoggerFromCtx(ctx).WithContext(ctx)
	// 检查下要更新的课程是否是官方课程，如果是，不让更新
	newSc.IsManuallyAdded = true
	meta, err := cluc.classRepo.GetClassMetaData(ctx, stuID, year, semester, oldClassId)
	if err != nil {
		return errcode.ErrClassDelete
	}
	if meta.IsOfficial {
		logh.Error(fmt.Sprintf("class [%v] is official, cannot delete", oldClassId))
		return fmt.Errorf("class [%v] is official, cannot delete", oldClassId)
	}
	if err := cluc.classRepo.UpdateClass(ctx, stuID, year, semester, oldClassId, newClassInfo, newSc); err != nil {
		return err
	}
	return nil
}

func (cluc *ClassUsecase) CheckSCIdsExist(ctx context.Context, stuID, year, semester, classId string) bool {
	return cluc.classRepo.CheckSCIdsExist(ctx, stuID, year, semester, classId)
}

func (cluc *ClassUsecase) GetAllSchoolClassInfosToOtherService(ctx context.Context, year, semester string, cursor time.Time) []*ClassInfoBO {
	return cluc.classRepo.GetAllSchoolClassInfos(ctx, year, semester, cursor)
}

func (cluc *ClassUsecase) GetStuIdsByJxbId(ctx context.Context, jxbId string) ([]string, error) {
	res, err := cluc.jxbRepo.FindStuIdsByJxbId(ctx, jxbId)
	if err != nil || len(res) == 0 {
		return []string{}, errcode.ErrGetStuIdByJxbId
	}

	return res, nil
}

func (cluc *ClassUsecase) addClass(ctx context.Context, stuID string, info *ClassInfoBO) error {
	logh := logger.GetLoggerFromCtx(ctx)
	sc := &StudentCourse{
		StuID:           stuID,
		ClaID:           info.ID,
		Year:            info.Year,
		Semester:        info.Semester,
		IsManuallyAdded: !info.MetaData.IsOfficial, // 手动添加课程
		Note:            info.MetaData.Note,
	}
	// 检查是否添加的课程是否已经存在
	if cluc.classRepo.CheckSCIdsExist(ctx, stuID, info.Year, info.Semester, info.ID) {
		logh.Error(fmt.Sprintf("[%v] already exists", info))
		return errcode.ErrClassIsExist
	}
	// 添加课程
	err := cluc.classRepo.AddClass(ctx, stuID, info.Year, info.Semester, info, sc)
	if err != nil {
		return err
	}
	return nil
}

func (cluc *ClassUsecase) getCourseFromCrawler(ctx context.Context, stuID string, year string, semester string) ([]*ClassInfoBO, []*StudentCourse, int, error) {
	logh := logger.GetLoggerFromCtx(ctx)
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
			cookieSuccess = false // 设置cookie获取状态
			logh.Error(fmt.Sprintf("Error getting cookie(stu_id:%v) from other service: %v", stuID, err))
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

	var stu Student

	sType := tool.ParseStudentType(stuID)
	switch sType {
	case tool.UnderGraduate:
		stu = &Undergraduate{}
	case tool.PostGraduate:
		stu = &GraduateStudent{}
	default:
		return nil, nil, -1, fmt.Errorf("the type of student isn't undergraduate")
	}

	ci, sc, sum, err := func() ([]*ClassInfoBO, []*StudentCourse, int, error) {
		defer func(currentTime time.Time) {
			logh.Info(fmt.Sprintf("craw class [%v,%v,%v] cost %v", stuID, year, semester, time.Since(currentTime)))
		}(time.Now())

		classinfos, scs, sum, err := stu.GetClass(ctx, stuID, year, semester, cookie, cluc.crawler)
		if err != nil {
			logh.Error(fmt.Sprintf("craw classlist(stu_id:%v year:%v semester:%v cookie:%v) failed: %v", stuID, year, semester, cookie, err))
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

func (cluc *ClassUsecase) GetClassNatures(ctx context.Context, stuID string) []string {
	return cluc.classRepo.GetClassNatures(ctx, stuID)
}

func extractJxb(infos []*ClassInfoBO) []string {
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

// 发送重试消息
func (cluc *ClassUsecase) sendRetryMsg(ctx context.Context, stuID, year, semester string) error {
	logh := logger.GetLoggerFromCtx(ctx).WithContext(ctx)

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
		logh.Error(fmt.Sprintf("Error sending retry message: %v", err))
	}
	return err
}

// 处理重试消息
func (cluc *ClassUsecase) handleRetryMsg(ctx context.Context, key, val []byte) {
	// 创建一个新 logger 存入上下文
	logh := logger.GlobalLogger.WithContext(ctx)

	retryInfo := map[string]string{}
	err := json.Unmarshal(val, &retryInfo)
	if err != nil {
		logh.Errorf("Error unmarshalling retry info: %v", string(val))
		return
	}
	stuID, ok := retryInfo["stu_id"]
	if !ok {
		logh.Errorf("Error getting stu_id from retry info: %v", string(val))
		return
	}
	year, ok := retryInfo["year"]
	if !ok {
		logh.Errorf("Error getting year from retry info: %v", string(val))
		return
	}
	semester, ok := retryInfo["semester"]
	if !ok {
		logh.Errorf("Error getting semester from retry info: %v", string(val))
		return
	}

	valLogger := logh.With(
		logger.String("stu_id", stuID),
		logger.String("year", year),
		logger.String("semester", semester),
	)
	ctx = logger.WithLogger(ctx, valLogger)

	localClassInfo, _ := cluc.classRepo.GetClassesFromLocal(ctx, stuID, year, semester)
	_ = cluc.crawClass(ctx, stuID, year, semester, time.Now(), localClassInfo, false)
}

func (cluc *ClassUsecase) UpdateClassNote(ctx context.Context, stuID, year, semester, classID, note string) error {
	logh := logger.GetLoggerFromCtx(ctx)
	err := cluc.classRepo.UpdateClassNote(ctx, stuID, year, semester, classID, note)
	if err != nil {
		logh.Error(fmt.Sprintf("Update note [%v] for class [%v %v %v %v] failed:%v", note, stuID, classID, year, semester, err))
		return err
	}
	return nil
}

func (cluc *ClassUsecase) pullClassListTask(
	ctx context.Context,
	log logger.Logger,
) {
	log.Info("Executing PullClassListTask")

	const (
		pageSize        = 200
		workerCount     = 16
		qps             = 20
		progressLogStep = 200
	)

	// 使用官方令牌桶限流
	limiter := rate.NewLimiter(rate.Limit(qps), qps)

	jobs := make(chan string, 1000)

	var (
		wg        sync.WaitGroup
		processed atomic.Int64
	)

	log.Infof("PullClassListTask started pageSize=%d workerCount=%d qps=%d", pageSize, workerCount, qps)

	year, semester := tool.GetCurrentAcademicYearAndSemesterStr(time.Now())

	// worker pool
	for i := 0; i < workerCount; i++ {
		workerID := i + 1
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			log.Infof("worker-%d started", id)
			defer log.Infof("worker-%d stopped", id)

			for stuID := range jobs {
				if tool.IsGraduated(stuID) {
					// 跳过已经毕业的学生
					log.Infof("worker-%d skipping graduated student %s", id, stuID)
					continue
				}

				// 等待令牌,如果上下文被取消则退出
				if err := limiter.Wait(ctx); err != nil {
					log.Warnf("worker-%d limiter wait canceled: %v", id, err)
					return
				}

				log.Infof("worker-%d processing student %s", id, stuID)

				localClassInfo, _ := cluc.classRepo.GetClassesFromLocal(ctx, stuID, year, semester)
				_ = cluc.crawClass(ctx, stuID, year, semester, time.Now(), localClassInfo, false)

				count := processed.Add(1)
				if count%progressLogStep == 0 {
					log.Infof("processed %d students so far", count)
				}
			}
		}(workerID)
	}

	// 分页生产任务
	lastStuID := ""
	for {
		stuIDs, err := cluc.classRepo.GetStudentIDs(ctx, lastStuID, pageSize)
		if err != nil {
			log.Errorf("get ids failed: %v", err)
			break
		}
		if len(stuIDs) == 0 {
			log.Info("no more student ids to enqueue")
			break
		}

		firstID := stuIDs[0]
		lastPageID := stuIDs[len(stuIDs)-1]
		log.Infof("enqueueing %d student ids range %s-%s", len(stuIDs), firstID, lastPageID)

		for _, id := range stuIDs {
			jobs <- id
		}

		lastStuID = lastPageID
	}

	close(jobs)
	wg.Wait()

	total := processed.Load()
	log.Infof("Finished PullClassListTask processed=%d lastStuID=%s", total, lastStuID)
}

func (cluc *ClassUsecase) cacheClass(
	ctx context.Context,
	log logger.Logger,
) {
	log.Info("Executing PullClassListTask")

	const (
		pageSize        = 200
		workerCount     = 16
		qps             = 200
		progressLogStep = 200
	)

	// 使用官方令牌桶限流
	limiter := rate.NewLimiter(rate.Limit(qps), qps)

	jobs := make(chan string, 1000)

	var (
		wg        sync.WaitGroup
		processed atomic.Int64
	)

	log.Infof("PullClassListTask started pageSize=%d workerCount=%d qps=%d", pageSize, workerCount, qps)

	year, semester := tool.GetCurrentAcademicYearAndSemesterStr(time.Now())

	// worker pool
	for i := 0; i < workerCount; i++ {
		workerID := i + 1
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			log.Infof("worker-%d started", id)
			defer log.Infof("worker-%d stopped", id)

			for stuID := range jobs {
				if tool.IsGraduated(stuID) {
					// 跳过已经毕业的学生
					log.Infof("worker-%d skipping graduated student %s", id, stuID)
					continue
				}

				// 等待令牌,如果上下文被取消则退出
				if err := limiter.Wait(ctx); err != nil {
					log.Warnf("worker-%d limiter wait canceled: %v", id, err)
					return
				}

				log.Infof("worker-%d processing student %s", id, stuID)

				// 缓存课表
				cluc.classRepo.CacheClass(ctx, stuID, year, semester)

				count := processed.Add(1)
				if count%progressLogStep == 0 {
					log.Infof("processed %d students so far", count)
				}
			}
		}(workerID)
	}

	// 分页生产任务
	lastStuID := ""
	for {
		stuIDs, err := cluc.classRepo.GetStudentIDs(ctx, lastStuID, pageSize)
		if err != nil {
			log.Errorf("get ids failed: %v", err)
			break
		}
		if len(stuIDs) == 0 {
			log.Info("no more student ids to enqueue")
			break
		}

		firstID := stuIDs[0]
		lastPageID := stuIDs[len(stuIDs)-1]
		log.Infof("enqueueing %d student ids range %s-%s", len(stuIDs), firstID, lastPageID)

		for _, id := range stuIDs {
			jobs <- id
		}

		lastStuID = lastPageID
	}

	close(jobs)
	wg.Wait()

	total := processed.Load()
	log.Infof("Finished PullClassListTask processed=%d lastStuID=%s", total, lastStuID)
}

// cleanRecycleBinTask 定时清理回收站中 zset 里已经过期的元素
func (cluc *ClassUsecase) cleanRecycleBinTask(
	ctx context.Context,
	log logger.Logger,
) {
	log.Info("Executing CleanRecycleBinTask")

	const (
		pageSize        = 200
		workerCount     = 16
		qps             = 200
		progressLogStep = 200
	)

	// 使用官方令牌桶限流
	limiter := rate.NewLimiter(rate.Limit(qps), qps)

	jobs := make(chan string, 1000)

	var (
		wg        sync.WaitGroup
		processed atomic.Int64
	)

	log.Infof("CleanRecycleBinTask started pageSize=%d workerCount=%d qps=%d", pageSize, workerCount, qps)

	year, semester := tool.GetCurrentAcademicYearAndSemesterStr(time.Now())

	// worker pool
	for i := 0; i < workerCount; i++ {
		workerID := i + 1
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			log.Infof("worker-%d started", id)
			defer log.Infof("worker-%d stopped", id)

			for stuID := range jobs {
				// 等待令牌,如果上下文被取消则退出
				if err := limiter.Wait(ctx); err != nil {
					log.Warnf("worker-%d limiter wait canceled: %v", id, err)
					return
				}

				log.Infof("worker-%d cleaning recycle bin for student %s", id, stuID)

				if err := cluc.recycleBinRepo.CleanExpired(ctx, stuID, year, semester); err != nil {
					log.Warnf("worker-%d clean recycle bin failed for student %s: %v", id, stuID, err)
				}

				count := processed.Add(1)
				if count%progressLogStep == 0 {
					log.Infof("processed %d students so far", count)
				}
			}
		}(workerID)
	}

	// 分页生产任务
	lastStuID := ""
	for {
		stuIDs, err := cluc.classRepo.GetStudentIDs(ctx, lastStuID, pageSize)
		if err != nil {
			log.Errorf("get ids failed: %v", err)
			break
		}
		if len(stuIDs) == 0 {
			log.Info("no more student ids to enqueue")
			break
		}

		firstID := stuIDs[0]
		lastPageID := stuIDs[len(stuIDs)-1]
		log.Infof("enqueueing %d student ids range %s-%s", len(stuIDs), firstID, lastPageID)

		for _, id := range stuIDs {
			jobs <- id
		}

		lastStuID = lastPageID
	}

	close(jobs)
	wg.Wait()

	total := processed.Load()
	log.Infof("Finished CleanRecycleBinTask processed=%d lastStuID=%s", total, lastStuID)
}
