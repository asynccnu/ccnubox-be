package content

// banner
type SaveBannerRequest struct {
	PictureLink string `json:"picture_link" binding:"required"`
	Id          int64  `json:"id,omitempty"` //可选,如果新增记录不用填写
	WebLink     string `json:"web_link" binding:"required"`
}

type Banner struct {
	WebLink     string `json:"web_link"`
	Id          int64  `json:"id"`
	PictureLink string `json:"picture_link"`
}

type GetBannersResponse struct {
	Banners []Banner `json:"banners"`
}

type DelBannerRequest struct {
	Id int64 `form:"id" json:"id" binding:"required"`
}

// calendar
type SaveCalendarRequest struct {
	Link string `json:"link" binding:"required"`
	Year int64  `json:"year"  binding:"required"`
}

type GetCalendarsResponse struct {
	Calendars []Calendar `json:"calendars"`
}

type DelCalendarRequest struct {
	Year int64 `json:"year"  binding:"required"`
}

type Calendar struct {
	Link string `json:"link"`
	Year int64  `json:"year"`
}

// department
type SaveDepartmentRequest struct {
	Id    int64  `json:"id"` //可选,如果新增记录不用填写
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Place string `json:"place"`
	Time  string `json:"time"`
}

type Department struct {
	Id    int64  `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Place string `json:"place"`
	Time  string `json:"time"`
}

type DelDepartmentRequest struct {
	Id int64 `json:"id" binding:"required"`
}

type GetDepartmentsResponse struct {
	Departments []*Department `json:"departments"`
}

type SaveInfoSumRequest struct {
	Link        string `json:"link" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Id          int64  `json:"id"` //可选,如果新增记录不用填写
	Image       string `json:"image" binding:"required"`
	Description string `json:"description" binding:"required"`
}

// infosum
type InfoSum struct {
	Link        string `json:"link"`
	Name        string `json:"name"`
	Id          int64  `json:"id"`
	Image       string `json:"image"`
	Description string `json:"description"`
}

type DelInfoSumRequest struct {
	Id int64 `json:"id" binding:"required"`
}

type GetInfoSumsResponse struct {
	InfoSums []*InfoSum `json:"info_sums"`
}

// website
type SaveWebsiteRequest struct {
	Link        string `json:"link" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Id          int64  `json:"id"` //可选,如果新增记录不用填写
	Image       string `json:"image" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type Website struct {
	Link        string `json:"link"`
	Name        string `json:"name"`
	Id          int64  `json:"id"`
	Image       string `json:"image"`
	Description string `json:"description"`
}

type DelWebsiteRequest struct {
	Id int64 `json:"id" binding:"required"`
}

type GetWebsitesResponse struct {
	Websites []*Website `json:"websites"`
}

// version
type GetUpdateVersionResponse struct {
	Version string `json:"version"`
}

type SaveVersionRequest struct {
	Version string `json:"version" binding:"required"`
}
