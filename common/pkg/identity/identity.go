package identity

import (
	"os"
	"strings"
)

func GetIdentity() string {

	// 1. 用户名
	user := os.Getenv("USERNAME")
	if user == "" {
		user = os.Getenv("USER")
	}
	if user == "" {
		user = "unknown"
	}

	// 2. 主机名
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}

	return sanitize(user + "@" + host)
}

// 防止非法字符
func sanitize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}
