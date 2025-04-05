package infoSum

type SaveInfoSumRequest struct {
	Link        string `json:"link"`
	Name        string `json:"name"`
	Id          int64  `json:"id,omitempty"`
	Image       string `json:"image"`
	Description string `json:"description"`
}

type InfoSum struct {
	Link        string `json:"link"`
	Name        string `json:"name"`
	Id          int64  `json:"id"`
	Image       string `json:"image"`
	Description string `json:"description"`
}

type DelInfoSumRequest struct {
	Id int64 `json:"id"`
}

type GetInfoSumsResponse struct {
	InfoSums []*InfoSum `json:"info_sums"`
}
