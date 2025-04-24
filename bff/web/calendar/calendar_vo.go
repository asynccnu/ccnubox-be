package calendar

type SaveCalendarRequest struct {
	Link string `json:"link" binding:"required"`
	Year int64  `json:"year"  binding:"required"`
}

type GetCalendarRequest struct {
	Year int64 `form:"year" binding:"required"`
}

type GetCalendarResponse struct {
	Link string `json:"link"  binding:"required"`
	Year int64  `json:"year"  binding:"required"`
}

type DelCalendarRequest struct {
	Year int64 `json:"year"  binding:"required"`
}
