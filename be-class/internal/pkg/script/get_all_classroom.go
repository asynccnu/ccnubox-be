package script

import (
	"encoding/json"
	"fmt"
	"github.com/valyala/fastjson"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Data struct {
	ClassRooms []string `json:"class_rooms"`
}

// 3,7,8,9,10,n 只爬这几个楼
func GetAllClassRooms(year, semester, cookie string) error {
	cli := &http.Client{}
	var wherePrefixs = []string{"3", "7", "8", "9", "10", "n"}

	var res []string
	for _, wherePrefix := range wherePrefixs {
		classrooms, err := getAllClassRooms(cli, year, semester, wherePrefix, cookie)
		if err != nil {
			return err
		}
		res = append(res, classrooms...)
	}

	f, err := os.Create("internal/data/classrooms.json")
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(&Data{ClassRooms: res}); err != nil {
		return err
	}
	return nil
}

func getAllClassRooms(cli *http.Client, year, semester, wherePrefix, cookie string) ([]string, error) {
	var mp = map[string]string{
		"1": "3",
		"2": "12",
		"3": "16",
	}
	var campus = 1
	if wherePrefix[0] == 'n' {
		campus = 2
	}
	var data = strings.NewReader(fmt.Sprintf(`fwzt=cx&xqh_id=%d&xnm=%s&xqm=%s&cdlb_id=&cdejlb_id=&qszws=&jszws=&cdmc=%s&lh=&jyfs=0&cdjylx=&sfbhkc=&zcd=%d&xqj=%d&jcd=%d&_search=false&nd=%d&queryModel.showCount=1000&queryModel.currentPage=1&queryModel.sortName=cdbh+&queryModel.sortOrder=asc&time=1`,
		campus, year, mp[semester], wherePrefix, 1<<(1-1), 6, 1<<(12-1), time.Now().UnixMilli()))
	req, err := http.NewRequest("POST", "https://xk.ccnu.edu.cn/jwglxt/cdjy/cdjy_cxKxcdlb.html?doType=query&gnmkdm=N2155", data)
	if err != nil {
		return nil, err
	}
	req.Header = http.Header{
		"Cookie":       []string{cookie},
		"Content-Type": []string{"application/x-www-form-urlencoded;charset=UTF-8"},
		"User-Agent":   []string{"Mozilla/5.0"}, // 精简UA
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 读取 Body 到字节数组
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	classrooms, err := extractCdIDsWithFastjson(bodyBytes, wherePrefix)
	if err != nil {
		return nil, err
	}
	return classrooms, nil
}
func extractCdIDsWithFastjson(rawJSON []byte, prefix string) ([]string, error) {
	var p fastjson.Parser
	v, err := p.ParseBytes(rawJSON)
	if err != nil {
		return nil, err
	}

	items := v.Get("items")
	if items == nil || items.Type() != fastjson.TypeArray {
		return nil, fmt.Errorf("items not found or not an array")
	}
	var cdIDs []string
	for _, item := range items.GetArray() {
		cdID := item.GetStringBytes("cd_id")
		if cdID != nil && strings.HasPrefix(string(cdID), prefix) && !containsSpecialChars(string(cdID)) && (len(string(cdID))-len(prefix)) == 3 {
			cdIDs = append(cdIDs, string(cdID))
		}
	}
	return cdIDs, nil
}
func containsSpecialChars(s string) bool {
	for _, r := range s {
		if !(r >= 'a' && r <= 'z') &&
			!(r >= 'A' && r <= 'Z') &&
			!(r >= '0' && r <= '9') {
			return true
		}
	}
	return false
}
