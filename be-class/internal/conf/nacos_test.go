package conf

import (
	"fmt"
	"testing"
)

func TestInitBootstrapFromNacos(t *testing.T) {
	bc := InitBootstrapFromNacos()
	if bc == nil {
		t.Fatal("初始化失败")
	}
	fmt.Printf("Bootstrap: %+v\n", bc)
}
