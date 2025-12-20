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
		return nil, err
	}
	return data, nil
}

func Send(cookie, ksxq, jsxq string) (*domain.GetRankByTermResp, error) {
	formData := url.Values{}
	formData.Set("ksxq", ksxq) //开始
	formData.Set("jsxq", jsxq) //结束
	formData.Set("_search", "false")
	formData.Set("nd", generateTimestamp())
	formData.Set("queryModel.showCount", "1000")
	formData.Set("queryModel.currentPage", "1")
	formData.Set("queryModel.sortName", "")
	formData.Set("queryModel.sortOrder", "asc")
	formData.Set("time", "0")

	req, err := http.NewRequest("POST", addr, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}

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
		return nil, err
	}
	defer resp.Body.Close()

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	body, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, err
	}

	var r Response
	err = json.Unmarshal(body, &r)
	if err != nil {
		return nil, err
	}

	var score, rank string
	if len(r.Items) >= 2 {
		score, rank = GetRankAndScore(r.Items[0].Tiptitle)
	}
	include := GetSubject(r.Items)

	return &domain.GetRankByTermResp{
		Rank:    rank,
		Score:   score,
		Include: include,
	}, nil
}

// 提取排名和学分
func GetRankAndScore(text string) (string, string) {
	pattern := `<span class='red'>(\d+\.?\d*)</?span>`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(text, -1)

	return matches[0][1], matches[1][1]
}

// 提取统计排名包含的科目
func GetSubject(data []Item) []string {
	var include []string

	for _, v := range data {
		include = append(include, v.Kcmc)
	}

	return include
}
