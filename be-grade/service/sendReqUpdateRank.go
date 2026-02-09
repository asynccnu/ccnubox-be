package service

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

const (
	// 查询学分排名的url
	addr = "https://xk.ccnu.edu.cn/jwglxt/cjtjfx/cjxftj_cxXscjxftjIndex.html?doType=query&gnmkdm=N309021"
)

type Response struct {
	CurrentPage   int      `json:"currentPage"`
	CurrentResult int      `json:"currentResult"`
	EntityOrField bool     `json:"entityOrField"`
	Items         []Item   `json:"items"`
	Limit         int      `json:"limit"`
	Offset        int      `json:"offset"`
	PageNo        int      `json:"pageNo"`
	PageSize      int      `json:"pageSize"`
	ShowCount     int      `json:"showCount"`
	SortName      string   `json:"sortName"`
	SortOrder     string   `json:"sortOrder"`
	Sorts         []string `json:"sorts"`
	TotalCount    int      `json:"totalCount"`
	TotalPage     int      `json:"totalPage"`
	TotalResult   int      `json:"totalResult"`
}

type Item struct {
	Kch         string  `json:"kch"`
	Cjxzm       string  `json:"cjxzm"`
	Kcxzmc      string  `json:"kcxzmc"`
	Tiptitle    string  `json:"tiptitle"`
	Cj          string  `json:"cj"`
	Jd          float64 `json:"jd"`
	Kcmc        string  `json:"kcmc"`
	RowID       int     `json:"row_id"`
	TotalResult int     `json:"totalresult"`
	Xf          string  `json:"xf"`
}

func generateTimestamp() string {
	return strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
}

func SendReqUpdateRank(cookie, xmnBegin, xmnEnd string) (*domain.GetRankByTermResp, error) {
	data, err := Send(cookie, xmnBegin, xmnEnd)
	if err != nil {
		return nil, errorx.Errorf("crawler: update rank failed, xmnBegin: %s, xmnEnd: %s, err: %w", xmnBegin, xmnEnd, err)
	}
	return data, nil
}

func Send(cookie, ksxq, jsxq string) (*domain.GetRankByTermResp, error) {
	formData := url.Values{}
	formData.Set("ksxq", ksxq) // 开始学期
	formData.Set("jsxq", jsxq) // 结束学期
	formData.Set("_search", "false")
	formData.Set("nd", generateTimestamp())
	formData.Set("queryModel.showCount", "1000")
	formData.Set("queryModel.currentPage", "1")
	formData.Set("queryModel.sortName", "")
	formData.Set("queryModel.sortOrder", "asc")
	formData.Set("time", "0")

	req, err := http.NewRequest("POST", addr, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, errorx.Errorf("crawler: create http request failed, err: %w", err)
	}

	// 模拟浏览器 Header
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Length", strconv.Itoa(len(formData.Encode())))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Host", "xk.ccnu.edu.cn")
	req.Header.Set("Origin", "https://xk.ccnu.edu.cn")
	req.Header.Set("Referer", "https://xk.ccnu.edu.cn/jwglxt/cjtjfx/cjxftj_cxXscjxftjIndex.html?gnmkdm=N309021&layout=default")
	req.Header.Set("Sec-Ch-Ua", `"Microsoft Edge";v="141", "Not?A_Brand";v="8", "Chromium";v="141"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		return nil, errorx.Errorf("crawler: http post failed, err: %w", err)
	}
	defer resp.Body.Close()

	// 增加状态码校验
	if resp.StatusCode != http.StatusOK {
		return nil, errorx.Errorf("crawler: school system status exception, code: %d", resp.StatusCode)
	}

	// 动态处理 Gzip 解压
	var reader io.ReadCloser
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, errorx.Errorf("crawler: init gzip reader failed, err: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	} else {
		reader = resp.Body
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, errorx.Errorf("crawler: read response body failed, err: %w", err)
	}

	var r Response
	err = json.Unmarshal(body, &r)
	if err != nil {
		// 解析失败时，在错误中附带 body 片段以便排查是否为 HTML 错误页
		bodyPreview := string(body)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200]
		}
		return nil, errorx.Errorf("crawler: unmarshal json failed, body: %s, err: %w", bodyPreview, err)
	}

	var score, rank string
	if len(r.Items) > 0 {
		var getErr error
		score, rank, getErr = GetRankAndScore(r.Items[0].Tiptitle)
		if getErr != nil {
			return nil, errorx.Errorf("crawler: parse rank info failed, tiptitle: %s, err: %w", r.Items[0].Tiptitle, getErr)
		}
	} else {
		// 处理教务系统返回空 Item 的情况（可能是该时间段无成绩）
		return nil, errorx.Errorf("crawler: items in response is empty")
	}

	include := GetSubject(r.Items)

	return &domain.GetRankByTermResp{
		Rank:    rank,
		Score:   score,
		Include: include,
	}, nil
}

// GetRankAndScore 提取排名和学分，增加了安全性校验
func GetRankAndScore(text string) (string, string, error) {
	pattern := `<span class='red'>(\d+\.?\d*)</?span>`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(text, -1)

	// 教务系统的 Tiptitle 通常包含两个红色高亮：学分绩和排名
	if len(matches) < 2 {
		return "", "", errorx.Errorf("regex match count insufficient, text: %s", text)
	}

	// matches[0][1] 为学分绩, matches[1][1] 为排名
	return matches[0][1], matches[1][1], nil
}

// GetSubject 提取统计排名包含的科目
func GetSubject(data []Item) []string {
	var include []string
	for _, v := range data {
		include = append(include, v.Kcmc)
	}
	return include
}
