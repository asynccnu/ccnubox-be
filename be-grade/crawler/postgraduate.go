package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/httpx"
)

type Graduate struct {
	client *http.Client
}

func NewGraduate(client *http.Client) (*Graduate, error) {
	if client == nil {
		return nil, errorx.Errorf("crawler: http client is nil")
	}
	return &Graduate{
		client: client,
	}, nil
}

type GraduateResp struct {
	Items      []GraduatePoints `json:"items"`
	TotalCount *int64           `json:"totalCount"`
}

type GraduatePoints struct {
	Xh     string `json:"xh"`     // 学号
	JxbID  string `json:"jxb_id"` // 教学班ID
	Kclbmc string `json:"kclbmc"` // 课程类别
	Kcxzmc string `json:"kcxzmc"` // 课程性质(必修)
	Kcbj   string `json:"kcbj"`   // 课程标记(主修)
	Xnm    string `json:"xnm"`    // 学年
	Xqm    string `json:"xqm"`    // 学期代号
	Kcmc   string `json:"kcmc"`   // 课程名称
	Xf     string `json:"xf"`     // 学分
	Jd     string `json:"jd"`     // 绩点
	Cj     string `json:"cj"`     // 成绩
}

func (g *Graduate) GetGraduateGrades(ctx context.Context, cookie string, xnm, xqm, showCount int64) ([]GraduatePoints, error) {
	return collectGradePages(ctx, func(page int) ([]GraduatePoints, *int64, error) {
		response, err := g.getGraduatePage(ctx, cookie, xnm, xqm, showCount, page)
		return response.Items, response.TotalCount, err
	}, func(p GraduatePoints) string { return p.JxbID })
}

func (g *Graduate) getGraduatePage(ctx context.Context, cookie string, xnm, xqm, showCount int64, page int) (GraduateResp, error) {
	// 研究生系统成绩查询 URL
	targetURL := "https://grd.ccnu.edu.cn/yjsxt/cjcx/cjcx_cxDgXscj.html?doType=query&gnmkdm=N305005"

	// 参数规格化
	var xnmStr, xqmStr, showCountStr string

	if xnm != 0 {
		xnmStr = strconv.FormatInt(xnm, 10)
	}

	// 映射教务系统的学期代号：1->3(秋), 2->12(春), 3->16(暑)
	switch xqm {
	case 1:
		xqmStr = "3"
	case 2:
		xqmStr = "12"
	case 3:
		xqmStr = "16"
	}

	// 保证单次查询覆盖量
	if showCount < 300 {
		showCount = 300
	}
	showCountStr = strconv.FormatInt(showCount, 10)

	// 构建表单
	formData := url.Values{
		"xnm":                    {xnmStr},
		"xqm":                    {xqmStr},
		"cjzt":                   {"3"}, // 成绩状态：已审核
		"_search":                {"false"},
		"nd":                     {strconv.FormatInt(time.Now().UnixMilli(), 10)},
		"queryModel.showCount":   {showCountStr},
		"queryModel.currentPage": {strconv.Itoa(page)},
		"queryModel.sortName":    {""},
		"queryModel.sortOrder":   {"asc"},
		"time":                   {"1"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return GraduateResp{}, errorx.Errorf("crawler: create graduate request failed, err: %w", err)
	}

	// 注入关键 Header
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := g.client.Do(req)
	if err != nil {
		return GraduateResp{}, errorx.Errorf("crawler: do graduate request failed, err: %w", err)
	}
	defer resp.Body.Close()

	// 增加 HTTP 状态校验：非 200 可能意味着被防火墙拦截或 Session 过期
	if resp.StatusCode != http.StatusOK {
		return GraduateResp{}, errorx.Errorf("crawler: graduate system status error, code: %d", resp.StatusCode)
	}

	body, err := httpx.ReadResponse(resp)
	if err != nil {
		return GraduateResp{}, errorx.Errorf("crawler: read graduate response failed, err: %w", err)
	}

	return parseGraduatePage(body)
}

func parseGraduateResponse(body []byte) ([]GraduatePoints, error) {
	response, err := parseGraduatePage(body)
	return response.Items, err
}

func parseGraduatePage(body []byte) (GraduateResp, error) {
	var response GraduateResp
	if err := json.Unmarshal(body, &response); err != nil {
		return GraduateResp{}, errorx.New("crawler: unmarshal graduate json failed")
	}
	if response.Items == nil {
		return GraduateResp{}, errorx.New("crawler: graduate system returned null grade data")
	}
	for _, grade := range response.Items {
		if _, err := ParseTotalScore(grade.Cj); err != nil {
			return GraduateResp{}, err
		}
	}
	return response, nil
}
