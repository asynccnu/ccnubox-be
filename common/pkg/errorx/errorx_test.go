package errorx

import (
	"fmt"
	"testing"

	"github.com/go-kratos/kratos/v2/errors"
)

func TestError(t *testing.T) {
	err := New("test")
	err1 := fmt.Errorf("测试: %w", err)
	if !errors.Is(err1, err) {
		t.Log("不相等")
	}

	err2 := Errorf("测试: %w", err1)
	if !errors.Is(err2, err) {
		t.Log("不相等")
	}

	err3 := Errorf("再测试一下: %w 测试两下: %d", err2, 1)
	fmt.Println()
	fmt.Println(err2.Error())
	fmt.Println()
	fmt.Printf("%+v\n", err3)

	// fmt.Sprintf("%+v", err3)
}
