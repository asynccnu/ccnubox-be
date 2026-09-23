package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/biz/model"
	classTool "github.com/asynccnu/ccnubox-be/be-classlist/pkg/tool"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	refreshRetryConsumerGroup  = "be-classlist-refresh-retry-worker"
	refreshRetryMessageVersion = 1
	maxRefreshRetryAttempts    = 3
	defaultRefreshJobTimeout   = 15 * time.Second
	retryPublishTimeout        = 5 * time.Second
	refreshStatusUpdateTimeout = 2 * time.Second
)

type refreshRetryMessage struct {
	Version     int    `json:"version"`
	StuID       string `json:"stu_id"`
	Year        string `json:"year"`
	Semester    string `json:"semester"`
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"max_attempts"`
}

// retryHandoffError 表示刷新失败后，下一条延迟重试消息未能发布。
// 只有该错误需要保留当前 Kafka 消息并让 consumer group 重新消费。
type retryHandoffError struct {
	cause error
}

func (e *retryHandoffError) Error() string {
	return fmt.Sprintf("refresh retry handoff failed: %v", e.cause)
}

func (e *retryHandoffError) Unwrap() error {
	return e.cause
}

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
		if latestLog.IsReady() && (localErr == nil || errors.Is(localErr, biz.ErrClassNotFound)) {
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
		return true, errorx.Errorf("usecase.class.hasScheduleConflict: classID=%s: %w", info.ID, biz.ErrClassAlreadyExists)
	}

	// 拉取本地课表检查是否有冲突
	classes, err := cluc.classRepo.GetClassesFromLocal(ctx, stuID, info.Year, info.Semester)
	if err != nil {
		if errors.Is(err, biz.ErrClassNotFound) {
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
		return false, errorx.Errorf("usecase.class.hasScheduleConflictWithClassesExcept: classWhen=%s: %w",
			info.ClassWhen, biz.ErrInvalidParam)
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

// 合并同一课表的并发刷新。调用者只控制自己的等待；共享任务不会被首个调用者取消。
func classRefreshSingleflightKey(stuID, year, semester string, attempt, maxAttempts int) string {
	// attempt 会影响失败后的重试调度，不能让不同重试阶段共享同一个闭包。
	// 否则重试消息可能合入 attempt=0 的普通刷新并把计数重置为 1。
	return fmt.Sprintf("craw:%s:%s:%s:attempt:%d:max:%d", stuID, year, semester, attempt, maxAttempts)
}

func (cluc *ClassUsecase) doCrawlWithSingleflight(ctx context.Context, stuID, year, semester string, jobTimeout time.Duration) ([]*model.ClassInfoBO, error) {
	return cluc.doCrawlAttemptWithSingleflight(ctx, stuID, year, semester, 0, maxRefreshRetryAttempts, jobTimeout)
}

func (cluc *ClassUsecase) doCrawlAttemptWithSingleflight(ctx context.Context, stuID, year, semester string, attempt, maxAttempts int, jobTimeout time.Duration) ([]*model.ClassInfoBO, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if jobTimeout <= 0 {
		jobTimeout = defaultRefreshJobTimeout
	}
	key := classRefreshSingleflightKey(stuID, year, semester, attempt, maxAttempts)
	resultCh := cluc.sfGroup.DoChan(key, func() (interface{}, error) {
		jobCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), jobTimeout)
		defer cancel()
		res, err := cluc.crawlOnce(jobCtx, stuID, year, semester)
		if err != nil {
			if handoffErr := cluc.coordinateRefreshRetry(jobCtx, stuID, year, semester, attempt, maxAttempts, err); handoffErr != nil {
				return nil, errors.Join(err, &retryHandoffError{cause: handoffErr})
			}
			return nil, err
		}
		if res == nil {
			return nil, fmt.Errorf("%w: crawler returned nil result", biz.ErrRefreshInvariant)
		}
		return res, nil
	})

	var result singleflight.Result
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result = <-resultCh:
	}

	if result.Err != nil {
		return nil, result.Err
	}

	res, ok := result.Val.([]*model.ClassInfoBO)
	if !ok {
		return nil, fmt.Errorf("%w: crawler returned unexpected result type %T", biz.ErrRefreshInvariant, result.Val)
	}
	return res, nil
}

func (cluc *ClassUsecase) updateRefreshLogStatus(ctx context.Context, logID uint64, status string) error {
	statusCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), refreshStatusUpdateTimeout)
	defer cancel()
	if err := cluc.refreshLogRepo.UpdateRefreshLogStatus(statusCtx, logID, status); err != nil {
		cluc.log.WithContext(statusCtx).Error("update class refresh log status failed",
			logger.Int64("log_id", int64(logID)),
			logger.String("status", status),
			logger.Error(err),
		)
		return err
	}
	return nil
}

// crawlOnce 只执行一次刷新；重试策略由调用方统一协调。
func (cluc *ClassUsecase) crawlOnce(ctx context.Context, stuID, year, semester string) ([]*model.ClassInfoBO, error) {
	local, _, err := cluc.loadLocal(ctx, stuID, year, semester)
	if err != nil && !errors.Is(err, biz.ErrClassNotFound) {
		cluc.log.WithContext(ctx).Warnf("load local metadata before refresh failed: %+v", err)
	}
	return cluc.crawMergedClass(ctx, stuID, year, semester, time.Now(), local, true)
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
		return nil, fmt.Errorf("%w: insert refresh log: %w", biz.ErrRefreshPersistence, err)
	}

	// 执行爬虫
	crawClassInfos, crawScs, _, err := cluc.getCourseFromCrawler(ctx, stuID, year, semester)
	if err != nil {
		if statusErr := cluc.updateRefreshLogStatus(ctx, logID, model.Failed); statusErr != nil {
			return nil, errors.Join(err, fmt.Errorf("%w: mark refresh failed: %w", biz.ErrRefreshPersistence, statusErr))
		}
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
		refreshErr := fmt.Errorf("%w: save class snapshot: %w", biz.ErrRefreshPersistence, err)
		if statusErr := cluc.updateRefreshLogStatus(ctx, logID, model.Failed); statusErr != nil {
			return nil, errors.Join(refreshErr, fmt.Errorf("%w: mark refresh failed: %w", biz.ErrRefreshPersistence, statusErr))
		}
		return nil, refreshErr
	}

	if !mergeAdd {
		if err := cluc.updateRefreshLogStatus(ctx, logID, model.Ready); err != nil {
			return nil, fmt.Errorf("%w: mark refresh ready: %w", biz.ErrRefreshPersistence, err)
		}
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
			refreshErr := fmt.Errorf("%w: delete conflicting added classes: %w", biz.ErrRefreshPersistence, err)
			if statusErr := cluc.updateRefreshLogStatus(ctx, logID, model.Failed); statusErr != nil {
				return nil, errors.Join(refreshErr, fmt.Errorf("%w: mark refresh failed: %w", biz.ErrRefreshPersistence, statusErr))
			}
			return nil, refreshErr
		}
	}

	if err := cluc.updateRefreshLogStatus(ctx, logID, model.Ready); err != nil {
		return nil, fmt.Errorf("%w: mark refresh ready: %w", biz.ErrRefreshPersistence, err)
	}
	_ = cluc.jxbRepo.SaveJxb(ctx, stuID, jxbIDs)

	crawClassInfos = append(crawClassInfos, addedInfos...)
	return crawClassInfos, nil
}

func (cluc *ClassUsecase) getCourseFromCrawler(ctx context.Context, stuID string, year string, semester string) ([]*model.ClassInfoBO, []*model.StudentCourseBO, int, error) {
	logh := cluc.log.WithContext(ctx)
	crawSuccess := true
	defer func(currentTime time.Time) {
		logh.Debug(fmt.Sprintf("getCourseFromCrawler(year:%v semester:%v success:%v) took %v", year, semester, crawSuccess, time.Since(currentTime)))
	}(time.Now())

	cookie, err := func() (string, error) {
		cookieSuccess := true
		defer func(currentTime time.Time) {
			logh.Debug(fmt.Sprintf("Get cookie (success:%v) from other service,cost %v", cookieSuccess, time.Since(currentTime)))
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
		return nil, nil, -1, fmt.Errorf("%w: empty cookie for stu_id=%s", biz.ErrCrawlerAuthentication, stuID)
	}

	var stu biz.Student

	sType := tool.ParseStudentType(stuID)
	switch sType {
	case tool.UnderGraduate:
		stu = &biz.Undergraduate{}
	case tool.PostGraduate:
		stu = &biz.GraduateStudent{}
	default:
		crawSuccess = false
		return nil, nil, -1, fmt.Errorf("%w: stu_id=%s", biz.ErrUnsupportedStudentType, stuID)
	}

	ci, sc, sum, err := func() ([]*model.ClassInfoBO, []*model.StudentCourseBO, int, error) {
		defer func(currentTime time.Time) {
			logh.Debug(fmt.Sprintf("craw class [year:%v semester:%v] cost %v", year, semester, time.Since(currentTime)))
		}(time.Now())

		classinfos, scs, sum, err := stu.GetClass(ctx, stuID, year, semester, cookie, cluc.crawler)
		if err != nil {
			logh.Errorf("craw classlist stu_id=%s year=%s semester=%s failed: %+v", stuID, year, semester, err)
			return nil, nil, -1, err
		}
		if len(classinfos) == 0 && len(scs) == 0 {
			return make([]*model.ClassInfoBO, 0), make([]*model.StudentCourseBO, 0), sum, nil
		}
		if len(classinfos) == 0 || len(scs) == 0 {
			return nil, nil, -1, fmt.Errorf("%w: inconsistent class and student-course results", biz.ErrCrawlerProtocol)
		}
		return classinfos, scs, sum, nil
	}()
	if err != nil {
		crawSuccess = false
		return nil, nil, -1, err
	}
	return ci, sc, sum, nil
}

// 发送一次有界重试消息。attempt 表示即将执行的异步重试序号，从 1 开始。
func (cluc *ClassUsecase) sendRetryMsg(ctx context.Context, stuID, year, semester string, attempt, maxAttempts int) error {
	logh := cluc.log.WithContext(ctx)
	if cluc.delayQue == nil {
		return errors.New("refresh retry queue is unavailable")
	}

	retryInfo := refreshRetryMessage{
		Version:     refreshRetryMessageVersion,
		StuID:       stuID,
		Year:        year,
		Semester:    semester,
		Attempt:     attempt,
		MaxAttempts: maxAttempts,
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

func (cluc *ClassUsecase) coordinateRefreshRetry(ctx context.Context, stuID, year, semester string, attempt, maxAttempts int, cause error) error {
	logh := cluc.log.WithContext(ctx)
	if !shouldRetryClassRefresh(cause) {
		logh.Warn("class refresh failure is not retryable",
			logger.String("stu_id", stuID),
			logger.String("year", year),
			logger.String("semester", semester),
			logger.Int("attempt", attempt),
			logger.Error(cause),
		)
		return nil
	}
	if attempt >= maxAttempts {
		// 超过最大重试，打日志
		logh.Error("class refresh retries exhausted",
			logger.String("stu_id", stuID),
			logger.String("year", year),
			logger.String("semester", semester),
			logger.Int("attempt", attempt),
			logger.Int("max_attempts", maxAttempts),
			logger.Error(cause),
		)
		return nil
	}

	nextAttempt := attempt + 1
	// 刷新失败经常意味着 jobCtx 已到 deadline。发布下一条消息必须脱离该取消信号
	publishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), retryPublishTimeout)
	defer cancel()
	if err := cluc.sendRetryMsg(publishCtx, stuID, year, semester, nextAttempt, maxAttempts); err != nil {
		logh.Error("schedule class refresh retry failed",
			logger.String("stu_id", stuID),
			logger.String("year", year),
			logger.String("semester", semester),
			logger.Int("attempt", nextAttempt),
			logger.Int("max_attempts", maxAttempts),
			logger.Error(err),
		)
		return fmt.Errorf("publish refresh retry attempt %d: %w", nextAttempt, err)
	}
	return nil
}

// 错误白名单，确定是否需要重试
func shouldRetryClassRefresh(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}

	if errors.Is(err, biz.ErrCrawlerProtocol) ||
		errors.Is(err, biz.ErrCrawlerEmptyResult) ||
		errors.Is(err, biz.ErrCookieUnavailable) ||
		errors.Is(err, biz.ErrUnsupportedStudentType) ||
		errors.Is(err, biz.ErrRefreshInvariant) ||
		errors.Is(err, biz.ErrInvalidParam) {
		return false
	}

	if errors.Is(err, biz.ErrRefreshPersistence) {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			switch mysqlErr.Number {
			case 1205, 1213, 2006, 2013:
				return true
			}
		}
	}

	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, biz.ErrCrawlerAuthentication) ||
		errors.Is(err, biz.ErrCrawlerTemporary) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) {
		return true
	}

	switch status.Code(err) {
	case codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted, codes.Unavailable:
		return true
	case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.PermissionDenied, codes.Unauthenticated:
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func (cluc *ClassUsecase) startRetryConsumer() {
	if cluc.delayQue == nil {
		return
	}

	go func() {
		if err := cluc.delayQue.Consume(refreshRetryConsumerGroup, cluc.handleRetryMessage); err != nil {
			cluc.log.Errorf("delayQue.Consume retry msg failed: %+v", err)
		}
	}()
}

func decodeRefreshRetryMessage(value []byte) (refreshRetryMessage, error) {
	var retryInfo refreshRetryMessage
	if err := json.Unmarshal(value, &retryInfo); err != nil {
		return refreshRetryMessage{}, fmt.Errorf("unmarshal refresh retry message: %w", err)
	}
	if retryInfo.StuID == "" || retryInfo.Year == "" || retryInfo.Semester == "" {
		return refreshRetryMessage{}, errors.New("invalid refresh retry message: missing required field")
	}

	// 兼容旧消息
	switch retryInfo.Version {
	case 0:
		if retryInfo.Attempt != 0 || retryInfo.MaxAttempts != 0 {
			return refreshRetryMessage{}, fmt.Errorf(
				"invalid legacy refresh retry message: attempt=%d max_attempts=%d",
				retryInfo.Attempt,
				retryInfo.MaxAttempts,
			)
		}
		retryInfo.Version = refreshRetryMessageVersion
		retryInfo.Attempt = 1
		retryInfo.MaxAttempts = maxRefreshRetryAttempts
	case refreshRetryMessageVersion:
		if retryInfo.MaxAttempts <= 0 || retryInfo.MaxAttempts > maxRefreshRetryAttempts {
			retryInfo.MaxAttempts = maxRefreshRetryAttempts
		}
	default:
		return refreshRetryMessage{}, fmt.Errorf("unsupported refresh retry message version: %d", retryInfo.Version)
	}

	if retryInfo.Attempt < 1 || retryInfo.Attempt > retryInfo.MaxAttempts {
		return refreshRetryMessage{}, fmt.Errorf(
			"invalid refresh retry attempt: attempt=%d max_attempts=%d",
			retryInfo.Attempt,
			retryInfo.MaxAttempts,
		)
	}
	return retryInfo, nil
}

func (cluc *ClassUsecase) handleRetryMessage(ctx context.Context, _ []byte, value []byte) (bool, error) {
	logh := cluc.log.WithContext(ctx)

	retryInfo, err := decodeRefreshRetryMessage(value)
	if err != nil {
		logh.Errorf("invalid refresh retry msg: err=%+v", err)
		// 消息体非法属于永久失败：消费器按阈值决定保留位点（等人工处理）还是跳过。
		return false, saramax.Permanent(err)
	}

	logh.Infof("consume refresh retry msg stu_id=%s year=%s semester=%s attempt=%d max_attempts=%d", retryInfo.StuID, retryInfo.Year, retryInfo.Semester, retryInfo.Attempt, retryInfo.MaxAttempts)
	retryJobTimeout := defaultRefreshJobTimeout
	if cluc.conf != nil && cluc.conf.ClassListConf != nil {
		configuredTimeout := time.Duration(cluc.conf.ClassListConf.WaitCrawTime) * time.Millisecond
		retryJobTimeout = max(retryJobTimeout, configuredTimeout)
	}
	_, err = cluc.doCrawlAttemptWithSingleflight(ctx, retryInfo.StuID, retryInfo.Year, retryInfo.Semester, retryInfo.Attempt, retryInfo.MaxAttempts, retryJobTimeout)
	if err != nil {
		logh.Errorf("handle refresh retry msg failed stu_id=%s year=%s semester=%s attempt=%d max_attempts=%d: %+v", retryInfo.StuID, retryInfo.Year, retryInfo.Semester, retryInfo.Attempt, retryInfo.MaxAttempts, err)
		var handoffErr *retryHandoffError
		if errors.As(err, &handoffErr) {
			return false, err
		}
		// 下一条重试已发布，或当前错误已判定为终态，可以确认当前消息。
		return true, err
	}
	logh.Infof("handle refresh retry msg succeeded stu_id=%s year=%s semester=%s attempt=%d", retryInfo.StuID, retryInfo.Year, retryInfo.Semester, retryInfo.Attempt)
	return true, nil
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
