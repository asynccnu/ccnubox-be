package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Graduate struct {
	client *http.Client
}

func NewGraduate(client *http.Client) (*Graduate, error) {
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
	// 请求URL
	targetURL := "https://grd.ccnu.edu.cn/yjsxt/cjcx/cjcx_cxDgXscj.html?doType=query&gnmkdm=N305005"

	// 类型转换
	var XnmStr, XqmStr, showCountStr string

	if xnm != 0 {
		XnmStr = strconv.Itoa(int(xnm))
	}

	switch xqm {
	case 1:
		XqmStr = "3"
	case 2:
		XqmStr = "12"
	case 3:
		XqmStr = "16"
	}

	if showCount >= 300 {
		showCountStr = strconv.Itoa(int(showCount))
	} else {
		showCountStr = strconv.Itoa(300)
	}

	// 构建表单数据
	formData := url.Values{
		"xnm":                    {XnmStr}, // 不填的话默认获取所有
		"xqm":                    {XqmStr}, // 不填的话默认获取所有
		"cjzt":                   {"3"},    // 成绩状态,只获取审核通过的
		"_search":                {"false"},
		"nd":                     {strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)},
		"queryModel.showCount":   {showCountStr}, // 重要查询参数
		"queryModel.currentPage": {"1"},
		"queryModel.sortName":    {""},
		"queryModel.sortOrder":   {"asc"},
		"time":                   {"1"},
	}

	// 将表单数据编码为字节流
	reqBody := bytes.NewBufferString(formData.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0")

	// 发送请求
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析JSON响应
	var response GraduateResp
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	return response.Items, nil
}
