package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-elecprice/domain"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var (
	client       = http.DefaultClient // 复用
	clientBackup = http.DefaultClient
)

// 通用 HTTP 请求函数
func sendRequest(ctx context.Context, url string, backup bool) (string, error) {
	var (
		resp *http.Response
		err  error
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://jnb.ccnu.edu.cn/MobileWebPayStandard_Vue/")
	req.Header.Set("Host", "jnb.ccnu.edu.cn")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0")

	if backup {
		resp, err = clientBackup.Do(req)
	} else {
		resp, err = client.Do(req)
	}
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

// merge 111空调(id1), 111照明(id2) -> 111[id1,id2], 222(id3) -> 222[id3] //map[id]name
func mergeRoomIds(m map[string]string) domain.RoomInfoList {
	resp := domain.RoomInfoList{}
	res := make([]domain.RoomInfo, 0, len(m))
	mp := make(map[string]domain.RoomInfo, len(m))
	for k, v := range m {
		// 获取房间名称和电费分布
		name := trimWord(v, KeyWordTypeLight, KeyWordTypeAC)
		if val, exists := mp[name]; exists {
			if hasWord(v, KeyWordTypeLight) {
				val.Light = k
			} else if hasWord(v, KeyWordTypeAC) {
				val.AC = k
			} else {
				val.Union = k
			}
			mp[name] = val
			continue
		}

		ri := domain.RoomInfo{
			RoomName: name,
		}
		if hasWord(v, KeyWordTypeLight) {
			ri.Light = k
		} else if hasWord(v, KeyWordTypeAC) {
			ri.AC = k
		} else {
			ri.Union = k
		}
		mp[name] = ri
	}

	for _, v := range mp {
		res = append(res, v)
	}

	resp.Rooms = res
	return resp
}

func hasWord(name string, kw ...KeyWordType) bool {
	for _, v := range kw {
		if strings.Contains(name, string(v)) {
			return true
		}
	}
	return false
}

func trimWord(name string, kw ...KeyWordType) string {
	for _, v := range kw {
		name = strings.ReplaceAll(name, string(v), "")
	}
	return name
}

func filter(m map[string]string) map[string]string {
	res := make(map[string]string, len(m))
	for k, v := range m {
		if isBlackListed(v) || isEmpty(v) || isEqual(v) {
			continue
		}
		v = formatRoomInfo(v)
		res[k] = v
	}
	return res
}

// formatRoomInfo 格式化房间信息
func formatRoomInfo(name string) string {
	return trimSuffixAndPrefix(replaceAlias(removeExcessiveWord(name)))
}

// removeExcessiveWord 去除中间的多余词汇
func removeExcessiveWord(name string) string {
	for _, v := range RemoveItems {
		name = strings.ReplaceAll(name, v, "")
	}
	return name
}

// trim 去除前后缀
func trimSuffixAndPrefix(name string) string {
	for _, item := range TrimPrefixItems {
		name = strings.TrimPrefix(name, item)
	}
	for _, item := range TrimSuffixItems {
		name = strings.TrimSuffix(name, item)
	}
	return name
}

// replaceAlias 替换别名, 尽可能统一名称
func replaceAlias(name string) string {
	for k, v := range ReplaceItems {
		name = strings.ReplaceAll(name, k, v)
	}
	return name
}

// isEqual 这里面是一些意义不明的房间
func isEqual(name string) bool {
	_, ok := EqualFold[name]
	return ok
}

// isEmpty 排除 xxx空, 但保留 xxx空调
func isEmpty(name string) bool {
	return strings.Contains(name, "空") && !strings.Contains(name, "空调")
}

func isBlackListed(name string) bool {
	for _, b := range BlackList {
		if strings.Contains(name, b) {
			return true
		}
	}
	return false
}

// handleDirtyArch 处理一下学校拉的屎, 楼层显示不对, 宿舍楼栋不匹配
func handleDirtyArch(ctx context.Context, res *domain.ResultArchitectureInfo, name string) {
	switch name {
	case YuanBaoShan:
		removeDong23(res)
	case EastRegion:
		addDong23(ctx, res)
	case SouthEast:
		adjustFloor(res)
	}
}

func removeDong23(res *domain.ResultArchitectureInfo) {
	i := 0
	list := res.ArchitectureInfoList.ArchitectureInfo
	for _, arch := range list {
		if !strings.Contains(arch.ArchitectureName, Dong23) {
			list[i] = arch
			i++
		}
	}
	res.ArchitectureInfoList.ArchitectureInfo = list[:i]
}

func addDong23(ctx context.Context, res *domain.ResultArchitectureInfo) {
	body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getArchitectureInfo?Area_ID=%s", ConstantMap[YuanBaoShan]), false)
	if err != nil {
		return
	}
	var result domain.ResultArchitectureInfo

	if err = xml.Unmarshal([]byte(body), &result); err != nil {
		return
	}

	if d23 := extractDong23(&result.ArchitectureInfoList); d23 != nil {
		res.ArchitectureInfoList.ArchitectureInfo = append(res.ArchitectureInfoList.ArchitectureInfo, *d23)
	}
}

func extractDong23(list *domain.ArchitectureInfoList) *domain.Architecture {
	for i := range list.ArchitectureInfo {
		arch := &list.ArchitectureInfo[i]
		if strings.Contains(arch.ArchitectureName, Dong23) {
			return arch
		}
	}
	return nil
}

func adjustFloor(res *domain.ResultArchitectureInfo) {
	list := res.ArchitectureInfoList.ArchitectureInfo
	for i := range list {
		if strings.Contains(list[i].ArchitectureName, Dong18) {
			num, err := strconv.Atoi(list[i].ArchitectureStorys)
			if err != nil {
				return
			}
			list[i].ArchitectureStorys = strconv.Itoa(num + 1)
		}
	}
}
