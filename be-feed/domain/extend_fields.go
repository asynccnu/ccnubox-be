package domain

import (
	"errors"
	"fmt"
	"github.com/ecodeclub/ekit"
)

type ExtendFields map[string]string

var errKeyNotFound = errors.New("没有找到对应的 key")

func (f ExtendFields) Get(key string) ekit.AnyValue {
	val, ok := f[key]
	if !ok {
		return ekit.AnyValue{
			Err: fmt.Errorf("%w, key %s", errKeyNotFound),
		}
	}
	return ekit.AnyValue{Val: val}
}
