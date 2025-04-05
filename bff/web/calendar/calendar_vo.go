package calendar

type SaveCalendarRequest struct {
	Link string `json:"link"`
	Year int64  `json:"year"`
}

type GetCalendarRequest struct {
	Year int64 `form:"year"`
}

type GetCalendarResponse struct {
	Link string `json:"link"`
	Year int64  `json:"year"`
}

type DelCalendarRequest struct {
	Year int64 `json:"year"`
}
