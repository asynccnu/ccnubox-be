package conf

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestInitInfraConfig(t *testing.T) {
	requireNacosIntegration(t)
	infra := InitInfraConfig()
	if infra == nil {
		t.Fatal("Failed to init infraConfig")
	}

	fmt.Printf("InitInfraConfig: %+v\n", infra)
}

func TestInitTransConfig(t *testing.T) {
	requireNacosIntegration(t)
	trans := InitServerConf()
	if trans == nil {
		t.Fatal("Failed to init transConfig")
	}

	fmt.Printf("InitServerConf: %+v\n", trans)
}

func TestNotificationConfigRemainsOptional(t *testing.T) {
	for _, tc := range []struct {
		name    string
		yaml    string
		present bool
		enabled bool
	}{
		{"legacy", "gradeConf:\n  high: 15\nconsumeConf:\n  consumeNum: 100\n", false, false},
		{"poll only", "notificationConf:\n  pollSeconds: 10\n", false, false},
		{"disabled", "notificationConf:\n  snapshotVerified: false\n", true, false},
		{"enabled", "notificationConf:\n  snapshotVerified: true\n", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("yaml")
			if err := v.ReadConfig(strings.NewReader(tc.yaml)); err != nil {
				t.Fatal(err)
			}
			var cfg ServerConf
			if err := v.Unmarshal(&cfg); err != nil {
				t.Fatal(err)
			}
			var value *bool
			if cfg.NotificationConf != nil {
				value = cfg.NotificationConf.SnapshotVerified
			}
			if (value != nil) != tc.present || (value != nil && *value != tc.enabled) {
				t.Fatalf("value=%v want present=%v enabled=%v", value, tc.present, tc.enabled)
			}
		})
	}
}

func requireNacosIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_NACOS_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_NACOS_INTEGRATION_TESTS=1 to run Nacos integration tests")
	}
}
