package crawler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	GET_GRADE_URL = "https://bkzhjw.ccnu.edu.cn/jsxsd/kscj/cjcx_list"
	DETAIL_GRADE  = "https://bkzhjw.ccnu.edu.cn/jsxsd/kscj/pscj_list.do"
	Login_URL     = "https://account.ccnu.edu.cn/cas/login"
)

var (
	COOKIE_TIMEOUT = errors.New("cookie过期")
)

// 存放本科生院相关的爬虫
type UnderGrad struct {
	client *http.Client
}

func NewUnderGrad(client *http.Client) (*UnderGrad, error) {
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
	XNXQID   string  `json:"xnxqid"` // 学年学期ID
	KCH      string  `json:"kch"`    // 课程号
	KCMC     string  `json:"kc_mc"`  // 课程名称
	KSDW     string  `json:"ksdw"`   // 开课单位
	XQMC     string  `json:"xqmc"`   // 学期名称
	XF       float32 `json:"xf"`     // 学分
	ZXS      float32 `json:"zxs"`    // 总学时
	KSFS     string  `json:"ksfs"`   // 考试方式
	KCSX     string  `json:"kcsx"`   // 课程属性
	XQStr    string  `json:"xqstr"`
	ZCJ      float32 `json:"zcj"`    // 最终成绩
	ZCJStr   string  `json:"zcjstr"` // 最终成绩字符串
	KZ       int     `json:"kz"`
	KCXZMC   string  `json:"kcxzmc"` // 课程性质
	XS0101ID string  `json:"xs0101id"`
	JX0404ID string  `json:"jx0404id"`
	KSXZ     string  `json:"ksxz"` // 考试性质
	RowNum   int     `json:"rownum_"`
}

// 1.前置请求,从html中提取相关参数,xnm,xqm建议默认都填写为0,0表示获取所有
func (c *UnderGrad) GetGrade(ctx context.Context, xnm, xqm int64, showCount int) ([]Grade, error) {
	var kksj string
	// 如下格式: 2029-2030-1
	if xnm != 0 && xqm != 0 {
		kksj = fmt.Sprintf("%d-%d-%d", xnm, xnm+1, xqm)
	}

	// 构造请求 URL
	reqURL := fmt.Sprintf(
		"%s?pageNum=1&pageSize=%d&kksj=%s&kcxz=&kcsx=&kcmc=&xsfs=all&sfxsbcxq=1",
		GET_GRADE_URL, showCount, kksj,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var gradeResp GradeResponse
	if err := json.Unmarshal(body, &gradeResp); err != nil {
		if strings.Contains(string(body), Login_URL) {
			return nil, COOKIE_TIMEOUT
		}
		return nil, fmt.Errorf("解析成绩数据失败: %w\n原始响应: %s", err, string(body))
	}

	return gradeResp.Data, nil
}

func (c *UnderGrad) GetDetail(ctx context.Context, xs0101id string, jx0404id string, cj0708id string, zcj float32) (Score, error) {
	// 构造动态参数
	reqURL := fmt.Sprintf(
		"%s?xs0101id=%s&jx0404id=%s&cj0708id=%s&zcj=%0.1f",
		DETAIL_GRADE, xs0101id, jx0404id, cj0708id, zcj,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return Score{}, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return Score{}, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Score{}, fmt.Errorf("读取响应失败: %w", err)
	}
	res := string(body)
	score, err := ParseScoreFromHTML(res)
	if err != nil {
		if strings.Contains(string(body), Login_URL) {
			return score, COOKIE_TIMEOUT
		}
		return Score{}, err
	}
	return score, nil
}

type Score struct {
	Cjxm1   float32 `json:"cjxm1"`   // 期末
	Zcj     string  `json:"zcj"`     // 总成绩
	Cjxm3   float32 `json:"cjxm3"`   // 平时
	Cjxm3bl string  `json:"cjxm3bl"` // 平时比重
	Cjxm1bl string  `json:"cjxm1bl"` // 期末比重
}

func ParseScoreFromHTML(html string) (Score, error) {
	// 使用 goquery 解析 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return Score{}, fmt.Errorf("解析 HTML 失败: %v", err)
	}

	var jsonStr string
	found := false

	// 遍历所有 script 标签
	doc.Find("script").EachWithBreak(func(i int, s *goquery.Selection) bool {
		scriptText := s.Text()

		// 正则查找 let arr = [{...}]
		re := regexp.MustCompile(`let\s+arr\s*=\s*(\[\{.*?\}\]);`)
		match := re.FindStringSubmatch(scriptText)
		if len(match) >= 2 {
			jsonStr = match[1]
			found = true
			return false // 停止遍历
		}
		return true
	})

	if !found {
		return Score{}, fmt.Errorf("未找到成绩数据")
	}

	// 反序列化 JSON
	var scores []Score
	err = json.Unmarshal([]byte(jsonStr), &scores)
	if err != nil {

		return Score{}, fmt.Errorf("成绩 JSON 解析失败: %v", err)
	}

	return scores[0], nil
}
