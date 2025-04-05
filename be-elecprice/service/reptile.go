package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

// 通用 HTTP 请求函数
func sendRequest(ctx context.Context, url string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应体失败: %w", err)
	}

	return string(body), nil
}

// 匹配正则工具
func matchRegex(input, pattern string) (map[string]string, error) {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(input, -1)
	if matches == nil {
		return nil, errors.New("未匹配到结果")
	}
	res := make(map[string]string)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		//"xx":"123"
		res[match[1]] = match[2]
	}
	return res, nil
}

func matchRegexpOneEle(input, pattern string) (string, error) {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if matches == nil {
		return "", errors.New("未匹配到结果")
	}
	if len(matches) < 2 {
		return "", errors.New("未匹配到结果")
	}
	return matches[1], nil
}
