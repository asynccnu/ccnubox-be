package timedTask

import (
	"github.com/robfig/cron/v3"
	"testing"
	"time"
)

func TestStartTask(t *testing.T) {
	taskManager := &Task{c: cron.New()}
	err := taskManager.startTask("* * * * *", func() {
		t.Log("task is executed")
	})
	if err != nil {
		t.Error(err)
	}
	time.Sleep(5 * time.Minute)
}
