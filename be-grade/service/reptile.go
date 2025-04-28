package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// 定义响应结构体
type GetDetailResp struct {
	Items []GetDetailItem `json:"items"`
}

var (
	COOKIE_TIMEOUT = errors.New("cookie过期")
)

// 创建一个全局client
var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse // 禁止自动跳转，返回原始响应
	},
	Transport: &http.Transport{
		MaxIdleConns:        100, // 最大空闲连接数
		MaxIdleConnsPerHost: 10,  // 每个主机最大空闲连接数
		MaxConnsPerHost:     100, // 每个主机最大连接数
	},
}

type GetDetailItem struct {
	JxbID  string `json:"jxb_id"` //教学班id
	Xmblmc string `json:"xmblmc"` //分数的描述:平时(70%)
	Xmcj   string `json:"xmcj"`   //分数: 88
}

type GetKcxzResp struct {
	Items []GetKcxzItem `json:"items"`
}

type GetKcxzItem struct {
	Xh     string `json:"xh"`
	JxbID  string `json:"jxb_id"`
	Kclbmc string `json:"kclbmc"`
	Kcxzmc string `json:"kcxzmc"` //课程性质名称
	Kcbj   string `json:"kcbj"`   //课程标记
	Xnm    string `json:"xnm"`
	Xqm    string `json:"xqm"`
	Kcmc   string `json:"kcmc"` //课程名称
	Xf     string `json:"xf"`   //学分
	Jd     string `json:"jd"`
	Cj     string `json:"cj"`
}

// getDetail 根据学期获取所有成绩,使用的是本科生院成绩详细信息的接口
func getDetail(ctx context.Context, cookie string, xnm int64, xqm int64, showCount int64) ([]GetDetailItem, error) {

	// 请求URL
	targetUrl := "https://xk.ccnu.edu.cn/jwglxt/cjcx/cjcx_cxXsKccjList.html?gnmkdm=N305007"

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
		"xnm":                    {XnmStr},
		"xqm":                    {XqmStr}, // 不填的话默认获取所有
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
	req, err := http.NewRequestWithContext(ctx, "POST", targetUrl, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析JSON响应
	var response GetDetailResp
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	// 返回解析后的成绩列表
	return response.Items, nil
}

// 获取课程性质
func getKcxz(ctx context.Context, cookie string, xnm int64, xqm int64, showCount int64) ([]GetKcxzItem, error) {

	// 请求URL
	targetUrl := "https://xk.ccnu.edu.cn/jwglxt/cjcx/cjcx_cxXsgrcj.html?doType=query&gnmkdm=N305005"

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
		"xnm":                    {XnmStr},
		"xqm":                    {XqmStr}, // 不填的话默认获取所有
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
	req, err := http.NewRequestWithContext(ctx, "POST", targetUrl, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	//如果被重定向的话要做处理
	if 400 <= resp.StatusCode && resp.StatusCode < 500 {
		return nil, COOKIE_TIMEOUT
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析JSON响应
	var response GetKcxzResp
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	// 返回解析后的成绩列表
	return response.Items, nil
}

func GetGrade(ctx context.Context, cookie string, xnm int64, xqm int64, showCount int64) ([]model.Grade, error) {
	var errChan = make(chan error, 2) // 使用带缓冲的通道
	var detailChan = make(chan []GetDetailItem, 1)
	var kcxzChan = make(chan []GetKcxzItem, 1)

	go func() {
		detail, err := getDetail(ctx, cookie, xnm, xqm, showCount)

		if err != nil {
			errChan <- err
			return
		}
		detailChan <- detail
	}()

	go func() {
		kcxz, err := getKcxz(ctx, cookie, xnm, xqm, showCount)
		if err != nil {
			errChan <- err
			return
		}
		kcxzChan <- kcxz

	}()

	select {
	case err := <-errChan:
		return nil, err

	case detail := <-detailChan:
		select {
		case kcxz := <-kcxzChan:
			return aggregateGrades(detail, kcxz), nil
		case err := <-errChan:
			return nil, err
		}

	case kcxz := <-kcxzChan:
		select {
		case detail := <-detailChan:
			return aggregateGrades(detail, kcxz), nil
		case err := <-errChan:
			return nil, err
		}

	}

}
