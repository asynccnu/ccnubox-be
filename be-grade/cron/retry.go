package cron

import (
	"time"

	"github.com/avast/retry-go/v4"
)

const (
	WaitForNext = 30 //重试等待30s
	MaxTries    = 3  //重试次数
)

func Retry[T any](fn func() (T, error)) (T, error) {
	var data T
	err := retry.Do(
		func() error {
			var err error
			data, err = fn()
			return err
		},
		retry.Attempts(MaxTries),
		retry.DelayType(retry.BackOffDelay),
		retry.Delay(WaitForNext*time.Second),
	)

	if err != nil {
		return data, err
	}

	return data, nil
}
