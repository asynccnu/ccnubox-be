package conf

import (
	"fmt"
	"testing"
)

func TestInitInfraConfig(t *testing.T) {
	infra := InitInfraConfig()
	if infra == nil {
		t.Fatal("Failed to init infraConfig")
	}

	fmt.Printf("InitInfraConfig: %+v\n", infra)
}

func TestInitTransConfig(t *testing.T) {
	trans := InitTransConfig()
	if trans == nil {
		t.Fatal("Failed to init transConfig")
	}

	fmt.Printf("InitTransConfig: %+v\n", trans)
}
