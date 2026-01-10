package cron

type Cron interface {
	StartCronTask()
}

func NewCron(
	gradeController *CalendarController,
) []Cron {
	return []Cron{gradeController}
}
