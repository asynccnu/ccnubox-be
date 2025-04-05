package banner

type SaveBannerRequest struct {
	PictureLink string `json:"picture_link"`
	Id          int64  `json:"id,omitempty"`
	WebLink     string `json:"web_link"`
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
	Id int64 `json:"id"`
}
