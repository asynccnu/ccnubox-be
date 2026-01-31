package cron

type Cron interface {
	StartCronTask()
}

func NewCron(
	gradeController *CalendarController,
	versionController *UpdateVersionController,
) []Cron {
	return []Cron{gradeController, versionController}
}
