package infoSum

type SaveInfoSumRequest struct {
	Link        string `json:"link" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Id          int64  `json:"id"` //可选,如果新增记录不用填写
	Image       string `json:"image" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type InfoSum struct {
	Link        string `json:"link" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Id          int64  `json:"id" binding:"required"`
	Image       string `json:"image" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type DelInfoSumRequest struct {
	Id int64 `json:"id" binding:"required"`
}

type GetInfoSumsResponse struct {
	InfoSums []*InfoSum `json:"info_sums" binding:"required"`
}
