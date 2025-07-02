package biz

// Seat, TimeSlot

type Seat struct {
	Name     string
	DevID    string
	KindName string
	Ts       []*TimeSlot
}

type TimeSlot struct {
	Start string
	End   string
	Owner string
}
