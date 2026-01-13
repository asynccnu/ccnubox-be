package conf

import (
	"fmt"
	"testing"
)

func TestInitBootstrap(t *testing.T) {
	bc := InitBootstrap()
	if bc == nil {
		t.Fatal("初始化失败")
	}
	fmt.Printf("Bootstrap: %+v\n", bc)
}
