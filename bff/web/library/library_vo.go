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
	ID        string     `json:"id"`
	Label     string     `json:"label"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	AfterFree bool       `json:"afterFree"`
	FreeList  []FreeTime `json:"ft"`
}

type FreeTime struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type ReserveSeatRequest struct {
	DevID string `json:"dev_id"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type ReserveSeatResponse struct {
	Message string `json:"message"`
}

type GetSeatRecordResponse struct {
	Records []Record `json:"records"`
}

type Record struct {
	ID        string `json:"id"`
	StuID     string `json:"stu_id"`
	SeatID    string `json:"seat_id"`
	RoomID    string `json:"room_id"`
	RoomName  string `json:"room_name"`
	BuildName string `json:"build_name"`
	FloorName string `json:"floor_name"`
	SeatLabel string `json:"seat_label"`
	MakeBegin string `json:"make_begin"`
	MakeEnd   string `json:"make_end"`
	MakeDate  string `json:"make_date"`
	Message   string `json:"message"`
	Status    string `json:"status"`
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
	RoomTypeID string `json:"room_type_id"`
	VenueID    string `json:"venue_id"`
	Date       string `json:"date"`
}

type GetDiscussionResponse struct {
	Discussions []Discussion
}

type Discussion struct {
	RoomID      string `json:"room_id"`
	Name        string `json:"name"`
	VenueID     string `json:"venue_id"`
	RoomType    string `json:"room_type"`
	Address     string `json:"address"`
	DisableList []DisableTime
}

type DisableTime struct {
	Start string `json:"start"`
	End   string `json:"end"`
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
	RoomID []string `json:"room_ids"`
	Start  string   `json:"start"`
	End    string   `json:"end"`
}

type ReserveSeatRandomlyResponse struct {
	Message string `json:"message"`
}

type GetSeatRecordRequest struct {
	Date []string `json:"date" binding:"required"` // YYYY-M-D 或 YYYY-MM-DD
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
