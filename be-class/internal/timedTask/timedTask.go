package timedTask

import (
	"context"
	clog "github.com/asynccnu/ccnubox-be/be-class/internal/log"
	"github.com/asynccnu/ccnubox-be/be-class/internal/pkg/tool"
	"github.com/google/wire"
	"github.com/robfig/cron/v3"
	"time"
)

var ProviderSet = wire.NewSet(NewTask)

// OptClassInfoToEs 定义接口
type OptClassInfoToEs interface {
	AddClassInfosToES(ctx context.Context, xnm, xqm string)
	DeleteSchoolClassInfosFromES(ctx context.Context, xnm, xqm string)
}

type ClassroomTask interface {
	ClearClassroomOccupancyFromES(ctx context.Context, year, semester string) error
	SaveFreeClassRoomFromLocal(ctx context.Context, year, semester string) error
}

// Task 定义 Task 结构体
type Task struct {
	a  OptClassInfoToEs
	cc ClassroomTask
	c  *cron.Cron
}

func NewTask(a OptClassInfoToEs, cc ClassroomTask) *Task {
	return &Task{
		a:  a,
		cc: cc,
		c:  cron.New(),
	}
}

// AddClassInfosToES 实现 Task 的 AddClassInfosToES 方法
func (t Task) AddClassInfosToES() {
	ctx := context.Background()
	//程序开始时先执行一次
	go func() {
		xnm, xqm := tool.GetXnmAndXqm(time.Now())
		clog.LogPrinter.Info("开始执行 AddClassInfosToES 任务")
		t.a.AddClassInfosToES(ctx, xnm, xqm)

		clog.LogPrinter.Info("等待数据刷新")
		//等待数据刷新
		time.Sleep(5 * time.Second)

		clog.LogPrinter.Info("开始执行 SaveFreeClassRoomFromLocal 任务")
		_ = t.cc.SaveFreeClassRoomFromLocal(ctx, xnm, xqm)
	}()

	// 每天凌晨 3 点执行
	err := t.startTask("0 3 * * *", func() {
		xnm, xqm := tool.GetXnmAndXqm(time.Now())
		clog.LogPrinter.Info("开始执行 AddClassInfosToES 任务")
		t.a.AddClassInfosToES(ctx, xnm, xqm)
		clog.LogPrinter.Info("开始执行 SaveFreeClassRoomFromLocal 任务")
		_ = t.cc.SaveFreeClassRoomFromLocal(ctx, xnm, xqm)
	})
	if err != nil {
		panic(err)
	}
}

// Clear 清洁任务
func (t Task) Clear() {
	ctx := context.Background()

	// 每天凌晨5点执行（5字段格式）
	err := t.startTask("0 5 * * *", func() {
		clog.LogPrinter.Info("开始执行 Clear 任务")
		xnm, xqm := tool.GetXnmAndXqm(time.Now())
		t.a.DeleteSchoolClassInfosFromES(ctx, xnm, xqm)
		_ = t.cc.ClearClassroomOccupancyFromES(ctx, xnm, xqm)
	})
	if err != nil {
		panic(err)
	}
}

// startTask 用于启动定时任务
func (t Task) startTask(spec string, task func()) error {
	_, err := t.c.AddFunc(spec, task)

	if err != nil {
		clog.LogPrinter.Errorf("failed to add  short task")
		return err
	}
	//task()
	// 启动定时任务调度器
	t.c.Start()
	return nil
}
