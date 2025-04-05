package cron

type Cron interface {
	StartCronTask()
}

// autoService服务还需要进行一个对表格的清理,如果学号已经超过毕业时间2年应当被自动清理

func NewCron(
	gradeController *GradeController,
) []Cron {
	return []Cron{gradeController}
}
