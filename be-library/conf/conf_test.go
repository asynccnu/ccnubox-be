package conf

import (
	"fmt"
	"os"
	"testing"
	"time"
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

func requireNacosIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_NACOS_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_NACOS_INTEGRATION_TESTS=1 to run Nacos integration tests")
	}
}

func TestTeamReminderDefaults(t *testing.T) {
	old := (&ServerConf{LibraryReminder: &LibraryReminderConf{Enabled: true}}).Reminder()
	if old.NotificationTypes.TeamSuccess || old.TeamScanCron != "*/5 * * * *" || old.TeamScanMinInterval != 4*time.Minute {
		t.Fatalf("旧配置不应自动开启组队扫描: %+v", old)
	}
	configured := (&ServerConf{LibraryReminder: &LibraryReminderConf{Enabled: true, NotificationTypes: &NotificationTypesConf{TeamSuccess: true}}}).Reminder()
	if !configured.NotificationTypes.TeamSuccess {
		t.Fatal("显式配置未开启组队提醒")
	}
}
