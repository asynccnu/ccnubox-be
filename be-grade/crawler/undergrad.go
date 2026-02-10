package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

const (
	GET_GRADE_URL = "https://bkzhjw.ccnu.edu.cn/jsxsd/kscj/cjcx_list"
	DETAIL_GRADE  = "https://bkzhjw.ccnu.edu.cn/jsxsd/kscj/pscj_list.do"
	Login_URL     = "https://account.ccnu.edu.cn/cas/login"
)

var (
	// ErrCookieTimeout 定义为 errorx 类型，方便上层做类型断言或错误码识别
	ErrCookieTimeout = errorx.New("crawler: cookie expired or session invalid")
)

// UnderGrad 存放本科生院相关的爬虫
type UnderGrad struct {
	client *http.Client
}

func NewUnderGrad(client *http.Client) (*UnderGrad, error) {
	if client == nil {
		return nil, errorx.New("crawler: http client is nil")
	}
	return &UnderGrad{
		client: client,
	}, nil
}

type GradeResponse struct {
	Code int     `json:"code"`
	Msg  string  `json:"msg"`
	Data []Grade `json:"data"`
}

type Grade struct {
	CJ0708ID string  `json:"cj0708id"`
	XNXQID   string  `json:"xnxqid"`
	KCH      string  `json:"kch"`
	KCMC     string  `json:"kc_mc"`
	KSDW     string  `json:"ksdw"`
	XQMC     string  `json:"xqmc"`
	XF       float32 `json:"xf"`
	ZXS      float32 `json:"zxs"`
	KSFS     string  `json:"ksfs"`
	KCSX     string  `json:"kcsx"`
	XQStr    string  `json:"xqstr"`
	ZCJ      float32 `json:"zcj"`
	ZCJStr   string  `json:"zcjstr"`
	KZ       int     `json:"kz"`
	KCXZMC   string  `json:"kcxzmc"`
	XS0101ID string  `json:"xs0101id"`
	JX0404ID string  `json:"jx0404id"`
	KSXZ     string  `json:"ksxz"`
	RowNum   int     `json:"rownum_"`
}

// GetGrade 获取本科生成绩列表
func (c *UnderGrad) GetGrade(ctx context.Context, xnm, xqm int64, showCount int) ([]Grade, error) {
	var kksj string
	// 格式转换: 2024-2025-1
	if xnm != 0 && xqm != 0 {
		kksj = fmt.Sprintf("%d-%d-%d", xnm, xnm+1, xqm)
	}

	reqURL := fmt.Sprintf(
		"%s?pageNum=1&pageSize=%d&kksj=%s&kcxz=&kcsx=&kcmc=&xsfs=all&sfxsbcxq=1",
		GET_GRADE_URL, showCount, kksj,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, errorx.Errorf("crawler: create undergrad request failed, err: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errorx.Errorf("crawler: do undergrad request failed, err: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorx.Errorf("crawler: read undergrad body failed, err: %w", err)
	}

	// 优先检测是否重定向到了登录页
	if strings.Contains(string(body), Login_URL) {
		return nil, ErrCookieTimeout
	}

	var gradeResp GradeResponse
	if err := json.Unmarshal(body, &gradeResp); err != nil {
		return nil, errorx.Errorf("crawler: unmarshal undergrad grade failed, body_sample: %.100s, err: %w", string(body), err)
	}

	return gradeResp.Data, nil
}

// GetDetail 获取成绩详情（平时分、期末分比例等）
func (c *UnderGrad) GetDetail(ctx context.Context, xs0101id string, jx0404id string, cj0708id string, zcj float32) (Score, error) {
	reqURL := fmt.Sprintf(
		"%s?xs0101id=%s&jx0404id=%s&cj0708id=%s&zcj=%0.1f",
		DETAIL_GRADE, xs0101id, jx0404id, cj0708id, zcj,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return Score{}, errorx.Errorf("crawler: create detail request failed, err: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36")

	resp, err := c.client.Do(req)
	if err != nil {
		return Score{}, errorx.Errorf("crawler: do detail request failed, err: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Score{}, errorx.Errorf("crawler: read detail body failed, err: %w", err)
	}

	if strings.Contains(string(body), Login_URL) {
		return Score{}, ErrCookieTimeout
	}

	score, err := ParseScoreFromHTML(string(body))
	if err != nil {
		return Score{}, errorx.Errorf("crawler: parse detail html failed, sid_info: %s, err: %w", cj0708id, err)
	}
	return score, nil
}

type Score struct {
	Cjxm1   float32 `json:"cjxm1"`   // 期末成绩
	Zcj     string  `json:"zcj"`     // 总成绩
	Cjxm3   float32 `json:"cjxm3"`   // 平时成绩
	Cjxm3bl string  `json:"cjxm3bl"` // 平时比重
	Cjxm1bl string  `json:"cjxm1bl"` // 期末比重
}

// ParseScoreFromHTML 从 HTML 的脚本中解析成绩 JSON
func ParseScoreFromHTML(htmlContent string) (Score, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return Score{}, errorx.Errorf("html: goquery load failed, err: %w", err)
	}

	var jsonStr string
	found := false

	// 本科生院详情页通常将数据存在 script 标签的 let arr = [...] 变量中
	doc.Find("script").EachWithBreak(func(i int, s *goquery.Selection) bool {
		scriptText := s.Text()
		re := regexp.MustCompile(`let\s+arr\s*=\s*(\[\{.*?\}\]);`)
		match := re.FindStringSubmatch(scriptText)
		if len(match) >= 2 {
			jsonStr = match[1]
			found = true
			return false
		}
		return true
	})

	if !found {
		return Score{}, errorx.New("html: grade array 'let arr' not found in script tags")
	}

	var scores []Score
	if err := json.Unmarshal([]byte(jsonStr), &scores); err != nil {
		return Score{}, errorx.Errorf("html: unmarshal inner json failed, raw: %.100s, err: %w", jsonStr, err)
	}

	if len(scores) == 0 {
		return Score{}, errorx.New("html: score list is empty")
	}

	return scores[0], nil
}
