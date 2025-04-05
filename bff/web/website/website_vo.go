package website

type SaveWebsiteRequest struct {
	Link        string `json:"link"`
	Name        string `json:"name"`
	Id          int64  `json:"id,omitempty"`
	Image       string `json:"image"`
	Description string `json:"description"`
}

type Website struct {
	Link        string `json:"link"`
	Name        string `json:"name"`
	Id          int64  `json:"id"`
	Image       string `json:"image"`
	Description string `json:"description"`
}

type DelWebsiteRequest struct {
	Id int64 `json:"id"`
}

type GetWebsitesResponse struct {
	Websites []*Website `json:"websites"`
}
