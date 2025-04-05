package static

type GetStaticByNameReq struct {
	StaticName string `form:"static_name"`
}

type StaticVo struct {
	Name    string            `json:"name"`
	Content string            `json:"content"`
	Labels  map[string]string `json:"labels"`
}

type SaveStaticReq struct {
	Name    string            `json:"name"`
	Content string            `json:"content"`
	Labels  map[string]string `json:"labels"`
}

type SaveStaticByFileReq struct {
	Name   string            `form:"name"`
	Labels map[string]string `form:"labels"`
}

type GetStaticByLabelsReq struct {
	Labels map[string]string `form:"labels"`
}

type GetStaticByLabelsResp struct {
	Statics []StaticVo `form:"statics"`
}
