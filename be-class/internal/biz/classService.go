package biz

import (
	"context"
	"fmt"
	v1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classlist/v1"
	"github.com/asynccnu/ccnubox-be/be-class/internal/lock"
	clog "github.com/asynccnu/ccnubox-be/be-class/internal/log"
	"github.com/asynccnu/ccnubox-be/be-class/internal/model"
	"time"
)

type EsProxy interface {
	AddClassInfo(ctx context.Context, classInfo ...model.ClassInfo) error
	ClearClassInfo(ctx context.Context, xnm, xqm string)
	SearchClassInfo(ctx context.Context, keyWords string, xnm, xqm string, page, pageSize int) ([]model.ClassInfo, error)
}

type ClassListService interface {
	GetAllSchoolClassInfos(ctx context.Context, xnm, xqm, cursor string) ([]model.ClassInfo, string, error)
	AddClassInfoToClassListService(ctx context.Context, req *v1.AddClassRequest) (*v1.AddClassResponse, error)
}

type ClassSerivceUserCase struct {
	es          EsProxy
	cs          ClassListService
	lockBuilder lock.Builder
	cache       Cache
}

func NewClassSerivceUserCase(es EsProxy, cs ClassListService, lockBuilder lock.Builder, cache Cache) *ClassSerivceUserCase {
	return &ClassSerivceUserCase{
		es:          es,
		cs:          cs,
		lockBuilder: lockBuilder,
		cache:       cache,
	}
}

func (c *ClassSerivceUserCase) AddClassInfoToClassListService(ctx context.Context, request *v1.AddClassRequest) (*v1.AddClassResponse, error) {
	return c.cs.AddClassInfoToClassListService(ctx, request)
}

func (c *ClassSerivceUserCase) SearchClassInfo(ctx context.Context, keyWords string, xnm, xqm string, page, pageSize int) ([]model.ClassInfo, error) {
	return c.es.SearchClassInfo(ctx, keyWords, xnm, xqm, page, pageSize)
}

func (c *ClassSerivceUserCase) AddClassInfosToES(ctx context.Context, xnm, xqm string) {
	//xnm, xqm := tool.GetXnmAndXqm()
	reqTime := "1949-10-01T00:00:00.000000"
	var tasks []string

	defer func() {
		_ = c.cache.Del(ctx, tasks...)
	}()

	for {
		classInfos, lastTime, err := c.cs.GetAllSchoolClassInfos(ctx, xnm, xqm, reqTime)
		if len(classInfos) == 0 {
			clog.LogPrinter.Warnf("request other service but get 0 classes")
			return
		}
		if err != nil {
			clog.LogPrinter.Errorf("failed to get all classlist")
			return
		}

		// 使用分布式锁来确保只有一个实例在执行
		lockKey := fmt.Sprintf("add_classlist_to_es_%v_%v_%v", xnm, xqm, reqTime)
		locker := c.lockBuilder.Build(lockKey)

		err = locker.Lock()

		if err != nil {
			clog.LogPrinter.Infof("the lock is not get, maybe other instance is doing this job")
			reqTime = lastTime
			continue
		}

		// 成功获取到锁
		clog.LogPrinter.Infof("get the lock: %v", lockKey)

		// 应该标识下任务是否完成
		// 如果任务已经完成了,应该接着看下一个
		taskName := "task:" + lockKey
		tasks = append(tasks, taskName)

		status, err := c.cache.Get(ctx, taskName)
		if err == nil && status == Finished {
			// 解锁
			ok, err1 := locker.Unlock()
			if !ok || err1 != nil {
				clog.LogPrinter.Errorf("unlock %v failed: %v", lockKey, err1)
			} else {
				clog.LogPrinter.Infof("unlock %v successfully", lockKey)
			}

			reqTime = lastTime
			continue
		}

		err = c.es.AddClassInfo(ctx, classInfos...)
		if err != nil {
			err1 := c.cache.Set(ctx, taskName, Failed, 10*time.Minute)
			if err1 != nil {
				clog.LogPrinter.Errorf("failed to set %v %v", taskName, err1)
			}
			clog.LogPrinter.Errorf("add classlist[%v] failed: %v", classInfos, err)
		}

		clog.LogPrinter.Infof("es has save %d classes", len(classInfos))

		err = c.cache.Set(ctx, taskName, Finished, 10*time.Minute)
		if err != nil {
			clog.LogPrinter.Errorf("failed to set %v %v", taskName, err)
		}

		// 解锁
		ok, err := locker.Unlock()
		if !ok || err != nil {
			clog.LogPrinter.Errorf("unlock %v failed: %v", lockKey, err)
		} else {
			clog.LogPrinter.Infof("unlock %v successfully", lockKey)
		}

		reqTime = lastTime
	}

}
func (c *ClassSerivceUserCase) DeleteSchoolClassInfosFromES(ctx context.Context, xnm, xqm string) {
	//xnm, xqm := tool.GetXnmAndXqm()
	c.es.ClearClassInfo(ctx, xnm, xqm)
}
