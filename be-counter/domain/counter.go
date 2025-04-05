package domain

type ChangeCounterLevels struct {
	StudentIds []string
	IsReduce   bool
	Steps      int64
}
