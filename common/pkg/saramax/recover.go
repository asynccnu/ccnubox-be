package saramax

import (
	"fmt"
	"runtime/debug"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// Catch 捕获 fn 的 panic，转成带堆栈的错误返回，并打出带 KAFKA_CONSUMER_PANIC 关键字的日志。
// sarama 给每个分区 claim 单独起协程，claim 内的 panic 会直接崩掉整个进程，
// 外层 RunConsumer 和消费协程都恢复不到，因此每个消费入口和业务调用点都必须就地兜底。
// panic 转成错误后走和普通失败一样的路径：不确认位点，重投或（永久失败时）按阈值跳过。
func Catch[T any](l logger.Logger, fn func() (T, error), fields ...logger.Field) (v T, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		stack := debug.Stack()
		if l != nil {
			l.With(fields...).Error(LogKeyConsumerPanic+" 消费逻辑 panic，已转为处理失败，位点不确认",
				logger.Any("panic", r), logger.String("stack", string(stack)))
		}
		err = fmt.Errorf("panic: %v\n%s", r, stack)
	}()
	return fn()
}

// CatchPanic 是 Catch 的无返回值封装，用于只需要错误结果的消费循环。
func CatchPanic(l logger.Logger, fn func() error, fields ...logger.Field) error {
	_, err := Catch(l, func() (struct{}, error) {
		return struct{}{}, fn()
	}, fields...)
	return err
}
