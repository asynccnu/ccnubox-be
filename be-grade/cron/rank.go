package cron

import (
	"context"
	"sort"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"github.com/go-redsync/redsync/v4"
	cronv3 "github.com/robfig/cron/v3"
)

const (
	Limit         = 10 //默认每次更新十条
	WaitForFinish = 90 //等待每一组完成一分半
	Save          = 2  //保留毕业学生数据多少年
	LessUse       = 6  //6个月没被查询的数据会清除
)

func (c *GradeController) StartRankCronTask() {
	// 更新和删除任务不能一起执行，不然可能删了又创建
	cron := cronv3.New()

	// 2-3月和8-9月执行一次
	spec1 := "0 0 1 2,3,8,9 *"
	_, err := cron.AddFunc(spec1, func() {
		lock := c.muRedis.NewMutex("AutoUpdateRank", redsync.WithTries(1))

		err := lock.Lock()
		if err != nil {
			// 防止不是竞争锁失败，而是别的问题导致的出错
			c.l.Warn("获取分布式锁失败", logger.Error(err))
			return
		}
		defer lock.Unlock()

		c.AutoUpdateRank()
	})

	// 每年10月1号执行一次
	spec2 := "0 0 1 10 *"
	_, err = cron.AddFunc(spec2, func() {
		lock := c.muRedis.NewMutex("CleanGraduateStudentRank", redsync.WithTries(1))

		err := lock.Lock()
		if err != nil {
			// 防止不是竞争锁失败，而是别的问题导致的出错
			c.l.Warn("获取分布式锁失败", logger.Error(err))
			return
		}
		defer lock.Unlock()

		c.CleanGraduateStudentRank()
	})

	// 每年6，12月执行一次
	spec3 := "0 0 1 6,12 *"
	_, err = cron.AddFunc(spec3, func() {
		lock := c.muRedis.NewMutex("CleanLessUseRank", redsync.WithTries(1))

		err := lock.Lock()
		if err != nil {
			// 防止不是竞争锁失败，而是别的问题导致的出错
			c.l.Warn("获取分布式锁失败", logger.Error(err))
			return
		}
		defer lock.Unlock()

		c.CleanLessUseRank()
	})

	if err != nil {
		c.l.Error("获取学分绩排名定时更新操作启动失败", logger.Error(err))
	}

	cron.Start()
}

// 自动更新学分绩排名
func (c *GradeController) AutoUpdateRank() {
	lastId := int64(0)

	for {
		data, err := Retry(func() ([]model.Rank, error) {
			return c.rankService.GetRankWhichShouldUpdate(context.Background(), Limit, lastId)
		})

		if err != nil {
			c.l.Error("多次重试自动更新学分绩排名失败", logger.Error(err))
			break
		}

		if len(data) == 0 {
			break
		}

		// 因为新的cookie生成会使同一学号旧的cookie失效，如果返回的同一组中有相同的学号则不能并发
		// 同一个cookie并发更新同一个学生的多种组合的排名教务系统给出的结果是不对的，感觉是教务系统没考虑过同一cookie同时发多请求的问题
		// 故策略为先排序，如果这一个与下一个学号不同则并发，相同则阻塞执行，每执行完一批等待充足时间后再搜索下一批防止数据出错
		sort.Slice(data, func(i, j int) bool {
			return data[i].StudentId < data[j].StudentId
		})

		for i := 0; i < len(data); i++ {
			t := &dao.Period{XnmBegin: data[i].XnmBegin, XqmBegin: data[i].XqmBegin, XqmEnd: data[i].XqmEnd, XnmEnd: data[i].XnmEnd}
			if i == len(data)-1 {
				go c.rankService.UpdateRank(context.Background(), data[i].StudentId, t)
				break
			}
			if data[i].StudentId == data[i+1].StudentId {
				c.rankService.UpdateRank(context.Background(), data[i].StudentId, t)
			} else {
				go c.rankService.UpdateRank(context.Background(), data[i].StudentId, t)
			}
		}

		lastId = data[len(data)-1].Id
		time.Sleep(WaitForFinish * time.Second)
	}
}

// 定期清除毕业学生的排名数据
func (c *GradeController) CleanGraduateStudentRank() {
	_, err := Retry(func() (struct{}, error) {
		err := c.rankService.DeleteGraduateStudentRank(context.Background(), Save)
		return struct{}{}, err
	})

	if err != nil {
		c.l.Error("多次重试清除毕业学生的排名数据失败", logger.Error(err))
	}
}

// 定期清除距离上次查询已经很久的数据
func (c *GradeController) CleanLessUseRank() {
	_, err := Retry(func() (struct{}, error) {
		err := c.rankService.DeleteLessUseRank(context.Background(), LessUse)
		return struct{}{}, err
	})

	if err != nil {
		c.l.Error("多次重试清除距离上次查询已经很久的学分绩排名数据失败", logger.Error(err))
	}
}
