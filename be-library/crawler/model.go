package crawler

import "time"

type Response struct {
	Message string `json:"message"`
}

type Seat struct {
	ID        string
	Label     string
	Name      string
	Status    string
	AfterFree bool
	FreeList  []*FreeTime
}

type FreeTime struct {
	Start string
	End   string
}

type Record struct {
	ID        string
	RoomID    string
	RoomName  string
	BuildName string
	FloorName string
	SeatID    string
	SeatLabel string
	MakeBegin time.Time
	MakeEnd   time.Time
	MakeDate  time.Time
	Status    string
	Message   string
}

type Discussion struct {
	RoomID      string
	Name        string
	VenueID     string
	RoomType    string
	Address     string
	DisableList []*DisableTime
}

type DisableTime struct {
	Start string
	End   string
}
