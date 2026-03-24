package crawler

type getSeatInfoReq struct {
	BeginMinute int `json:"beginMinute"`
	EndMinute   int `json:"endMinute"`
	MinMinute   int `json:"minMinute"`
}

type getDiscussionInfoReq struct {
	CurrentPage int    `json:"currentPage"`
	PageSize    int    `json:"pageSize"`
	RoomTypeId  string `json:"roomTypeId"`
	SelectDate  string `json:"selectDate"`
	VenueId     string `json:"venueId"`
}
