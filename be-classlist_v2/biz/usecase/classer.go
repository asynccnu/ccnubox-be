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

	classRepo      biz.ClassRepo
	refreshLogRepo biz.RefreshLogRepo

	sfGroup singleflight.Group
}

func (cluc *ClassUsecase) GetClasses(ctx context.Context, stuID, year, semester string, refresh bool) ([]*model.ClassInfoBO, *time.Time, error) {
	// 当前时间
	currentTime := time.Now()

	logh := logger.GetLoggerFromCtx(ctx)
	noExpireCtx := logger.WithLogger(context.Background(), logh)

	// 等待爬虫的时间
	waitCrawTime := time.Duration(cluc.conf.ClassListConf.WaitCrawTime) * time.Millisecond
	// 刷新间隔,当前时间距离上次刷新时间超过该值时,需要重新刷新
	refreshInterval := time.Duration(cluc.conf.ClassListConf.RefreshInterval) * time.Millisecond

	// 1. 本地查询阶段
	lastRefreshTime := cluc.refreshLogRepo.GetLastRefreshTime(ctx, stuID, year, semester, model.Ready, currentTime) // 获取上次刷新成功的时间
	localClassInfo, err := cluc.classRepo.GetClassesFromLocal(ctx, stuID, year, semester)                           // 获取本地课表

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
		if refreshLog.UpdatedAt.After(currentTime.Add(-refreshInterval)) {
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
		resChan := make(chan []*model.ClassInfoBO, 1)
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
		if res, ok := v.([]*model.ClassInfoBO); ok {
			return res, &currentTime, nil
		}
	}

	// 如果爬取失败或超时，降级返回本地旧数据
	return localClassInfo, lastRefreshTime, nil
}

// 爬取课表并保存
func (cluc *ClassUsecase) crawClass(ctx context.Context, stuID, year, semester string, logTime time.Time, localClassInfo []*model.ClassInfoBO, mergeAdd bool) []*model.ClassInfoBO {
	logh := logger.From(ctx)

	metaMap := make(map[string]model.ClassMetaDataBO, len(localClassInfo))
	// 构建本地课程 ID -> MetaData 的映射，避免 O(n^2) 比较
	for _, lc := range localClassInfo {
		metaMap[lc.ID] = lc.MetaData
	}

	logID, err := cluc.refreshLogRepo.InsertRefreshLog(ctx, stuID, year, semester, model.Pending, logTime)
	if err != nil {
		logh.Errorf("failed to insert refresh log,param(%v,%v,%v)", stuID, year, semester)
		return nil
	}

	crawClassInfos, crawScs, _, err := cluc.getCourseFromCrawler(ctx, stuID, year, semester)
	if err != nil {
		_ = cluc.refreshLogRepo.UpdateRefreshLogStatus(ctx, logID, model.Failed)
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
