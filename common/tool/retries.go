package tool

import (
	"time"
)

func Retry[T any](fn func() (T, error)) (T, error) {
	var (
		result T
		err    error
	)
	for i := 0; i < 3; i++ {
		result, err = fn()
		if err == nil {
			return result, nil
		}

		time.Sleep(time.Duration(i) * time.Second)
	}

	return result, err
}
