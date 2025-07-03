package banner

type SaveBannerRequest struct {
	PictureLink string `json:"picture_link" binding:"required"`
	Id          int64  `json:"id,omitempty"` //可选,如果新增记录不用填写
	WebLink     string `json:"web_link" binding:"required"`
}

type Banner struct {
	WebLink     string `json:"web_link" binding:"required"`
	Id          int64  `json:"id" binding:"required"`
	PictureLink string `json:"picture_link" binding:"required"`
}

type GetBannersResponse struct {
	Banners []Banner `json:"banners" binding:"required"`
}

type DelBannerRequest struct {
	Id int64 `form:"id" json:"id" binding:"required"`
}
