package website

type SaveWebsiteRequest struct {
	Link        string `json:"link" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Id          int64  `json:"id"` //可选,如果新增记录不用填写
	Image       string `json:"image" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type Website struct {
	Link        string `json:"link" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Id          int64  `json:"id" binding:"required"`
	Image       string `json:"image" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type DelWebsiteRequest struct {
	Id int64 `json:"id" binding:"required"`
}

type GetWebsitesResponse struct {
	Websites []*Website `json:"websites" binding:"required"`
}
