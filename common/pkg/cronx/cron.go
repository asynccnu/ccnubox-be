package cronx

// Package cronx 提供一个具备任务增删查能力的定时任务管理器，
// 统一接入项目自定义的 logger，便于在任务执行过程输出一致的结构化日志。

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

var (
	// ErrTaskExists 表示同名任务已经存在。
	ErrTaskExists = errors.New("cronx: task ID already exists")
	// ErrTaskNotFound 表示任务不存在。
	ErrTaskNotFound = errors.New("cronx: task ID not found")
	// ErrInvalidTaskID 表示任务 ID 为空。
	ErrInvalidTaskID = errors.New("cronx: task ID must not be empty")
	// ErrNilTaskFunc 表示任务函数为空。
	ErrNilTaskFunc = errors.New("cronx: task func must not be nil")
)

// TaskFunc 定义任务函数签名，统一透出上下文与 logger。
type TaskFunc func(ctx context.Context, log logger.Logger)

// TaskMeta 用于描述已注册任务的基本信息。
type TaskMeta struct {
	ID   string
	Spec string
	Next time.Time
	Prev time.Time
}

// Manager 负责任务的注册、删除、列出以及生命周期管理。
type Manager struct {
	cron    *cron.Cron
	logger  logger.Logger
	baseCtx context.Context

	mu    sync.RWMutex
	tasks map[string]taskEntry
}

type taskEntry struct {
	entryID cron.EntryID
	spec    string
}

// Option 用于定制 Manager 行为。
type Option func(*config)

type config struct {
	ctx         context.Context
	cronOptions []cron.Option
}

// WithContext 指定任务执行时使用的基础 context。
func WithContext(ctx context.Context) Option {
	return func(c *config) {
		if ctx != nil {
			c.ctx = ctx
		}
	}
}

// WithCronOptions 允许追加 robfig/cron 的可选项，例如时区、拦截器等。
func WithCronOptions(opts ...cron.Option) Option {
	return func(c *config) {
		c.cronOptions = append(c.cronOptions, opts...)
	}
}

// NewManager 创建任务管理器，并立即启动底层调度器。
func NewManager(log logger.Logger, opts ...Option) *Manager {
	if log == nil {
		panic("cronx: logger must not be nil")
	}

	cfg := config{
		ctx: context.Background(),
		cronOptions: []cron.Option{},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.ctx == nil {
		cfg.ctx = context.Background()
	}

	manager := &Manager{
		cron:    cron.New(cfg.cronOptions...),
		logger:  log.AddCallerSkip(1),
		baseCtx: cfg.ctx,
		tasks:   make(map[string]taskEntry),
	}

	manager.cron.Start()
	manager.logger.Info("cron manager started")

	return manager
}

// Stop 停止调度器，直至所有正在执行的任务完成或 context 超时。
func (m *Manager) Stop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	doneCtx := m.cron.Stop()

	select {
	case <-doneCtx.Done():
		m.logger.Info("cron manager stopped")
		return nil
	case <-ctx.Done():
		m.logger.Warn("cron manager stop aborted", logger.String("reason", ctx.Err().Error()))
		return ctx.Err()
	}
}

// AddTask 注册一个新的定时任务。
func (m *Manager) AddTask(id, spec string, job TaskFunc) error {
	if id == "" {
		return ErrInvalidTaskID
	}

	if job == nil {
		return ErrNilTaskFunc
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[id]; exists {
		return ErrTaskExists
	}

	entryID, err := m.cron.AddFunc(spec, m.wrapJob(id, spec, job))
	if err != nil {
		return err
	}

	m.tasks[id] = taskEntry{
		entryID: entryID,
		spec:    spec,
	}

	m.logger.Info("cron task registered", logger.String("task_id", id), logger.String("spec", spec))

	return nil
}

// RemoveTask 删除已注册的任务，返回是否删除成功。
func (m *Manager) RemoveTask(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.tasks[id]
	if !ok {
		return false
	}

	m.cron.Remove(entry.entryID)
	delete(m.tasks, id)

	m.logger.Info("cron task removed", logger.String("task_id", id))

	return true
}

// ListTasks 列出当前所有任务的调度信息。
func (m *Manager) ListTasks() []TaskMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metas := make([]TaskMeta, 0, len(m.tasks))
	for id, entry := range m.tasks {
		cronEntry := m.cron.Entry(entry.entryID)
		metas = append(metas, TaskMeta{
			ID:   id,
			Spec: entry.spec,
			Next: cronEntry.Next,
			Prev: cronEntry.Prev,
		})
	}

	return metas
}

func (m *Manager) wrapJob(id, spec string, job TaskFunc) func() {
	return func() {
		taskLogger := m.logger.With(
			logger.String("task_id", id),
			logger.String("spec", spec),
		)

		ctx := m.baseCtx
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = logger.WithLogger(ctx, taskLogger)

		start := time.Now()
		taskLogger.Debug("cron task started")

		defer func() {
			duration := time.Since(start)
			if r := recover(); r != nil {
				taskLogger.Error(
					"cron task panic",
					logger.Any("panic", r),
					logger.String("duration", duration.String()),
				)
				return
			}
			taskLogger.Info(
				"cron task finished",
				logger.String("duration", duration.String()),
			)
		}()

		job(ctx, taskLogger)
	}
}
