package library

type GetSeatRequest struct {
	RoomIDs []string `json:"room_ids"`
}

type GetSeatResponse struct {
	Rooms []Room `json:"rooms"`
}

type Room struct {
	RoomID string `json:"room_id"`
	Seats  []Seat `json:"seats"`
}

type Seat struct {
	LabName   string     `json:"labName"`
	KindName  string     `json:"kindName"`
	DevID     string     `json:"devId"`
	DevName   string     `json:"devName"`
	TimeSlots []TimeSlot `json:"ts"`
}

type TimeSlot struct {
	Start  string `json:"start"`
	End    string `json:"end"`
	State  string `json:"state"`
	Owner  string `json:"owner"`
	Occupy bool   `json:"occupy"`
}

type ReserveSeatRequest struct {
	DevID string `json:"dev_id"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type GetSeatRecordResponse struct {
	Records []Record `json:"records"`
}

type Record struct {
	ID       string `json:"id"`
	Owner    string `json:"owner"`
	Start    string `json:"start"`
	End      string `json:"end"`
	TimeDesc string `json:"timeDesc"`
	States   string `json:"states"`
	DevName  string `json:"devName"`
	RoomID   string `json:"roomId"`
	RoomName string `json:"roomName"`
	LabName  string `json:"labName"`
}

type GetHistoryResponse struct {
	Histories []History `json:"history"`
}

type History struct {
	Place      string `json:"place"`
	Floor      string `json:"floor"`
	Status     string `json:"status"`
	Date       string `json:"date"`
	SubmitTime string `json:"submitTime"`
}

type GetCreditPointResponse struct {
	CreditPoints CreditPoints
}

type CreditPoints struct {
	Summary CreditSummary  `json:"summary"`
	Records []CreditRecord `json:"records"`
}

type CreditSummary struct {
	System string `json:"system"` // 个人预约制度
	Remain string `json:"remain"`
	Total  string `json:"total"`
}

type CreditRecord struct {
	Title    string `json:"title"`    // 原因标题
	Subtitle string `json:"subtitle"` // 扣分及时间
	Location string `json:"location"` // 地点及备注
}

type GetDiscussionRequest struct {
	ClassID string `json:"class_id"`
	Date    string `json:"date"`
}

type GetDiscussionResponse struct {
	Discussions []Discussion
}

type Discussion struct {
	LabID    string `json:"labId"`
	LabName  string `json:"labName"`
	KindID   string `json:"kindId"`
	KindName string `json:"kindName"`
	DevID    string `json:"devId"`
	DevName  string `json:"devName"`
	TS       []DiscussionTS
}

type DiscussionTS struct {
	Start  string `json:"start"`
	End    string `json:"end"`
	State  string `json:"state"`
	Title  string `json:"title"`
	Owner  string `json:"owner"`
	Occupy bool   `json:"occupy"`
}

type SearchUserRequest struct {
	StudentID string `form:"student_id" binding:"required"`
}

type SearchUserResponse struct {
	Search Search
}

type Search struct {
	ID    string `json:"id"`
	Pid   string `json:"Pid"`
	Name  string `json:"name"`
	Label string `json:"label"`
}

type ReserveDiscussionRequest struct {
	DevID  string   `json:"dev_id"`
	LabID  string   `json:"lab_id"`
	KindID string   `json:"kind_id"`
	Title  string   `json:"title"`
	Start  string   `json:"start"`
	End    string   `json:"end"`
	List   []string `json:"list"`
}

type CancelReserveRequest struct {
	ID string `form:"id" binding:"required"`
}

type ReserveSeatRandomlyRequest struct {
	RoomID string `json:"room_ids"`
	DevID   string   `json:"dev_id"`
	Start   string   `json:"start"`
	End     string   `json:"end"`
}

type ReserveSeatRandomlyResponse struct {
	Message string `json:"message"`
}

type Comment struct {
	ID        int    `json:"id"`         // 评论ID
	SeatID    string `json:"seat_id"`    // 关联座位
	Username  string `json:"user_id"`    // 发表评论的用户
	Content   string `json:"content"`    // 评论内容
	Rating    int    `json:"rating"`     // 评分（1-5）
	CreatedAt string `json:"created_at"` // 创建时间
}

type CreateCommentReq struct {
	SeatID   string `json:"seat_id"`
	Content  string `json:"content"`
	Rating   int    `json:"rating"`
	Username string `json:"username"`
}

type IDreq struct {
	ID int `json:"id" form:"id"`
}
