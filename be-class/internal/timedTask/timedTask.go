package timedTask

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-class/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-class/internal/client"
	clog "github.com/asynccnu/ccnubox-be/be-class/internal/log"
	"github.com/asynccnu/ccnubox-be/be-class/internal/pkg/semesterInfo"
	"github.com/asynccnu/ccnubox-be/be-class/internal/pkg/tool"
	"github.com/google/wire"
	"github.com/robfig/cron/v3"
	"strconv"
	"time"
)

var ProviderSet = wire.NewSet(NewTask)

// Task 定义 Task 结构体
type Task struct {
	classServiceUserCase *biz.ClassServiceUserCase
	freeClassroomBiz     *biz.FreeClassroomBiz
	classlistService     *client.ClassListService
	c                    *cron.Cron
}

func NewTask(classServiceUserCase *biz.ClassServiceUserCase, freeClassroomBiz *biz.FreeClassroomBiz, classlistService *client.ClassListService) *Task {
	return &Task{
		classServiceUserCase: classServiceUserCase,
		freeClassroomBiz:     freeClassroomBiz,
		classlistService:     classlistService,
		c:                    cron.New(),
	}
}

// RegisterAddClassInfosToESTask 实现 Task 的 RegisterAddClassInfosToESTask 方法
func (t Task) RegisterAddClassInfosToESTask() {
	ctx := context.Background()
	//程序开始时先执行一次
	go func() {
		xnm, xqm := tool.GetXnmAndXqm(time.Now())
		clog.LogPrinter.Info("开始执行 RegisterAddClassInfosToESTask 任务")
		t.classServiceUserCase.AddClassInfosToES(ctx, xnm, xqm)

		clog.LogPrinter.Info("等待数据刷新")
		//等待数据刷新
		time.Sleep(5 * time.Second)

		clog.LogPrinter.Info("开始执行 SaveFreeClassRoomFromLocal 任务")
		_ = t.freeClassroomBiz.SaveFreeClassRoomFromLocal(ctx, xnm, xqm)
	}()

	// 每天凌晨 3 点执行
	err := t.AddTask("0 3 * * *", func() {
		xnm, xqm := tool.GetXnmAndXqm(time.Now())
		clog.LogPrinter.Info("开始执行 AddClassInfosToES 任务")
		t.classServiceUserCase.AddClassInfosToES(ctx, xnm, xqm)
		clog.LogPrinter.Info("开始执行 SaveFreeClassRoomFromLocal 任务")
		_ = t.freeClassroomBiz.SaveFreeClassRoomFromLocal(ctx, xnm, xqm)
	})
	if err != nil {
		panic(err)
	}
}

// RegisterClearClassInfoTask 清洁任务
func (t Task) RegisterClearClassInfoTask() {
	ctx := context.Background()

	// 每天凌晨5点执行（5字段格式）
	err := t.AddTask("0 5 * * *", func() {
		clog.LogPrinter.Info("开始执行 ClearClassInfo 任务")
		xnm, xqm := tool.GetXnmAndXqm(time.Now())
		t.classServiceUserCase.DeleteSchoolClassInfosFromES(ctx, xnm, xqm)
		_ = t.freeClassroomBiz.ClearClassroomOccupancyFromES(ctx, xnm, xqm)
	})
	if err != nil {
		panic(err)
	}
}

func (t Task) RegisterCrawFreeClassroomTask(stuId string) {
	ctx := context.Background()

	schoolTime, err := t.classlistService.GetSchoolDay(ctx)
	if err != nil {
		clog.LogPrinter.Errorf("get school day failed: %v", err)
		return
	}
	si, err := semesterInfo.GetSemesterInfo(schoolTime)
	if err != nil {
		clog.LogPrinter.Errorf("get semester info failed: %v", err)
		return
	}
	go func() {
		// 程序开始时先执行一次
		t.freeClassroomBiz.LoadOneWeekFreeClassRoom(ctx, stuId, strconv.Itoa(si.Year), strconv.Itoa(si.Semester), si.WeekNumber)
	}()

	// 每周一4点执行
	err = t.AddTask("0 4 * * 1", func() {
		t.freeClassroomBiz.LoadOneWeekFreeClassRoom(ctx, stuId, strconv.Itoa(si.Year), strconv.Itoa(si.Semester), si.WeekNumber)
	})
	if err != nil {
		panic(err)
	}
}

// AddTask 用于添加定时任务
func (t Task) AddTask(spec string, task func()) error {
	_, err := t.c.AddFunc(spec, task)
	if err != nil {
		clog.LogPrinter.Errorf("failed to add  short task")
		return err
	}
	return nil
}

func (t Task) Start() {
	t.c.Start()
}

func (t Task) Stop() {
	ctx := t.c.Stop()
	select {
	case <-ctx.Done():
		clog.LogPrinter.Info("所有定时任务已停止")
	}
}
