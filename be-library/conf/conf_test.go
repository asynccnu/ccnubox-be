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

func TestInvitationConfigurationGate(t *testing.T) {
	defaults := (*ServerConf)(nil).Reminder()
	if defaults.NotificationTypes.TeamInvitation || defaults.TeamInvitation.ContractVerified || defaults.TeamInvitation.MaxRecipients != 20 || defaults.TeamInvitation.MaxDeliveryAge != 15*time.Minute {
		t.Fatalf("defaults=%+v", defaults)
	}
	if err := defaults.ValidateInvitation(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*LibraryReminderConf){
		func(c *LibraryReminderConf) { c.NotificationTypes.TeamInvitation = true },
		func(c *LibraryReminderConf) { c.Enabled = true; c.NotificationTypes.TeamInvitation = true },
		func(c *LibraryReminderConf) { c.TeamInvitation.MaxRecipients = -1 },
		func(c *LibraryReminderConf) { c.TeamInvitation.MaxRecipients = 21 },
		func(c *LibraryReminderConf) { c.TeamInvitation.MaxDeliveryAge = time.Hour },
		func(c *LibraryReminderConf) { c.TeamInvitation.RequestTimeout = time.Minute },
		func(c *LibraryReminderConf) {
			c.TeamInvitation.ContractVerified = true
			c.TeamInvitation.KnownMemberStatuses = []int{1, 7}
			c.TeamInvitation.PendingMemberStatuses = []int{1}
		},
	} {
		c := (*ServerConf)(nil).Reminder()
		mutate(&c)
		if err := c.ValidateInvitation(); err == nil {
			t.Fatalf("accepted=%+v", c)
		}
	}
	c := (*ServerConf)(nil).Reminder()
	c.Enabled = true
	c.NotificationTypes.TeamInvitation = true
	c.TeamInvitation.ContractVerified = true
	c.TeamInvitation.KnownMemberStatuses = []int{1, 7, 8}
	c.TeamInvitation.PendingMemberStatuses = []int{7}
	if err := c.ValidateInvitation(); err != nil {
		t.Fatal(err)
	}
}
