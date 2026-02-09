package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
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
	Items []GraduatePoints `json:"items"`
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
		"queryModel.currentPage": {"1"},
		"queryModel.sortName":    {""},
		"queryModel.sortOrder":   {"asc"},
		"time":                   {"1"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, errorx.Errorf("crawler: create graduate request failed, err: %w", err)
	}

	// 注入关键 Header
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, errorx.Errorf("crawler: do graduate request failed, url: %s, err: %w", targetURL, err)
	}
	defer resp.Body.Close()

	// 增加 HTTP 状态校验：非 200 可能意味着被防火墙拦截或 Session 过期
	if resp.StatusCode != http.StatusOK {
		return nil, errorx.Errorf("crawler: graduate system status error, code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorx.Errorf("crawler: read graduate response failed, err: %w", err)
	}

	var response GraduateResp
	if err := json.Unmarshal(body, &response); err != nil {
		// 当教务系统返回 HTML（如登录页）而非 JSON 时，捕获前 100 字符用于排查
		bodySample := string(body)
		if len(bodySample) > 100 {
			bodySample = bodySample[:100]
		}
		return nil, errorx.Errorf("crawler: unmarshal graduate json failed, body: %s, err: %w", bodySample, err)
	}

	return response.Items, nil
}
