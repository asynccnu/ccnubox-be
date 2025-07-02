package library

type GetSeatRequest struct {
	RoomID string `json:"room_id"`
	StuID  string `json:"stu_id"`
}

type GetSeatResponse struct {
	Seats []Seat `json:"seats"`
}

type Seat struct {
	Name      string     `json:"name"`
	DevID     string     `json:"dev_id"`
	KindName  string     `json:"kind_name"`
	TimeSlots []TimeSlot `json:"ts"`
}

type TimeSlot struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Owner string `json:"owner"`
}
