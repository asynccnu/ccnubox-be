package crawler

import (
	"context"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/classLog"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/pkg/tool"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Crawler2 struct {
	client *http.Client
}

func NewClassCrawler2() *Crawler2 {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,              // 最大空闲连接
			IdleConnTimeout:     90 * time.Second, // 空闲连接超时
			TLSHandshakeTimeout: 10 * time.Second, // TLS握手超时
			DisableKeepAlives:   false,            // 确保不会意外关闭 Keep-Alive
		},
	}
	return &Crawler2{
		client: client,
	}
}

func (c *Crawler2) GetClassInfosForUndergraduate(ctx context.Context, stuID, year, semester, cookie string) ([]*biz.ClassInfo, []*biz.StudentCourse, error) {
	logh := classLog.GetLogHelperFromCtx(ctx)
	url := fmt.Sprintf(
		"https://bkzhjw.ccnu.edu.cn/jsxsd/framework/mainV_index_loadkb.htmlx?zc=&kbjcmsid=16FD8C2BE55E15F9E0630100007FF6B5&xnxq01id=%s&xswk=false",
		c.getys(year, semester))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logh.Errorf("http.NewRequest err=%v", err)
		return nil, nil, err
	}
	req.Header = http.Header{
		"Cookie":       []string{cookie},
		"Content-Type": []string{"application/x-www-form-urlencoded;charset=UTF-8"},
		"User-Agent":   []string{"Mozilla/5.0"}, // 精简UA
	}
	resp, err := c.client.Do(req)
	if err != nil {
		logh.Errorf("client.Do err=%v", err)
		return nil, nil, err
	}
	defer resp.Body.Close()

	// 读取 Body 到字节数组
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logh.Errorf("failed to read response body: %v", err)
		return nil, nil, err
	}

	infos, err := c.extractCourses(ctx, year, semester, string(bodyBytes))
	if err != nil {
		logh.Errorf("failed to extract infos: %v", err)
		return nil, nil, fmt.Errorf("failed to extract infos: %v", err)
	}

	scs := make([]*biz.StudentCourse, 0, len(infos))

	for _, info := range infos {
		scs = append(scs, &biz.StudentCourse{
			StuID:           stuID,
			ClaID:           info.ID,
			Year:            year,
			Semester:        semester,
			IsManuallyAdded: false,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		})
	}
	return infos, scs, nil
}

func (c *Crawler2) GetClassInfoForGraduateStudent(ctx context.Context, stuID, year, semester, cookie string) ([]*biz.ClassInfo, []*biz.StudentCourse, error) {
	return c.GetClassInfosForUndergraduate(ctx, stuID, year, semester, cookie)
}

func (c *Crawler2) getys(year, semester string) string {
	// 将年份字符串转为整数
	y, _ := strconv.Atoi(year)

	// 组合结果
	return fmt.Sprintf("%d-%d-%s", y, y+1, semester)
}

func (c *Crawler2) extractCourses(ctx context.Context, year, semester, html string) ([]*biz.ClassInfo, error) {
	logh := classLog.GetLogHelperFromCtx(ctx)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("NewDocumentFromReader err: %v", err)
	}

	var weekdayMap = map[string]int{
		"星期一": 1,
		"星期二": 2,
		"星期三": 3,
		"星期四": 4,
		"星期五": 5,
		"星期六": 6,
		"星期日": 7,
		"星期天": 7, // 星期日和星期天都对应7
	}

	var classInfos []*biz.ClassInfo

	doc.Find("li.qz-toolitiplists").Each(func(i int, selection *goquery.Selection) {
		var classInfo biz.ClassInfo

		classInfo.Year, classInfo.Semester = year, semester

		classInfo.Classname = selection.Find(".qz-tooltipContent-title").Text()
		classInfo.UpdatedAt = time.Now()
		classInfo.CreatedAt = classInfo.UpdatedAt

		selection.Find(".qz-tooltipContent-detailitem").Each(func(i int, selection *goquery.Selection) {
			str := c.extractAfterColon(selection.Text())
			switch i {
			case 1:
				classInfo.Teacher = str
			case 2:
				classInfo.Where = c.parseClassRoom(str)
			case 3:
				classInfo.WeekDuration = c.parseWeekDuration(ctx, str)
				classInfo.Weeks = c.parseWeeks(classInfo.WeekDuration)
				//重新格式化week
				classInfo.WeekDuration = tool.FormatWeeks(tool.ParseWeeks(classInfo.Weeks))
				classInfo.Day = int64(weekdayMap[c.parseDay(ctx, str)])
			case 4:
				classInfo.ClassWhen, err = c.parseClassWhen(str)
				if err != nil {
					logh.Errorf("parseClassWhen: %v", err)
				}
			case 5:
				classInfo.Credit = c.parseCredit(str)
			}
		})

		classInfo.UpdateID()
		classInfo.UpdateJxbId()

		classInfos = append(classInfos, &classInfo)
	})
	return classInfos, nil
}

// extractAfterColon 提取字符串中冒号后的内容（支持中文冒号和英文冒号）
func (c *Crawler2) extractAfterColon(s string) string {
	// 去除前后空格
	s = strings.TrimSpace(s)

	// 查找中文冒号和英文冒号的位置
	idx := strings.Index(s, "：") // 中文冒号
	if idx == -1 {
		return ""
	}

	// 返回冒号后的内容（去除前后空格）
	return strings.TrimSpace(s[idx+len("："):])
}

func (c *Crawler2) parseWeekDuration(ctx context.Context, s string) string {
	// 方法1：使用字符串操作
	logh := classLog.GetLogHelperFromCtx(ctx)
	start := strings.Index(s, "[")
	end := strings.Index(s, "周]")
	if start == -1 || end == -1 || start >= end {
		logh.Error("parseWeekDuration err")
		return "1-17"
	}
	return s[start+1 : end]
}

func (c *Crawler2) parseWeeks(weekDuration string) int64 {
	sections := strings.Split(weekDuration, ",")

	var weeks int64

	for _, section := range sections {
		nums := c.parseNumber(section)
		if len(nums) == 1 {
			weeks |= 1 << (nums[0] - 1)
		}
		if len(nums) == 2 {
			for i := nums[0]; i <= nums[1]; i++ {
				weeks |= 1 << (i - 1)
			}
		}
	}
	return weeks
}

// 提取字符串的全部数字
func (c *Crawler2) parseNumber(s string) []int64 {
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(s, -1)
	var numbers []int64
	for _, match := range matches {
		num, _ := strconv.Atoi(match)
		numbers = append(numbers, int64(num))
	}
	return numbers
}

func (c *Crawler2) parseDay(ctx context.Context, s string) string {
	logh := classLog.GetLogHelperFromCtx(ctx)
	if idx := strings.Index(s, "]"); idx != -1 && idx+1 < len(s) {
		return s[idx+1:]
	}
	logh.Error("parseDay err")
	return "星期一"
}

func (c *Crawler2) parseClassWhen(s string) (string, error) {
	parts := strings.Split(strings.TrimSuffix(s, "小节"), "~")
	var start, end string
	if len(parts) == 0 {
		return "", errors.New("classWhen is not like 1-2 or 2")
	}
	if len(parts) == 1 {
		start = strings.TrimLeft(parts[0], "0")
		end = start
		return start + "-" + end, nil
	}
	start = strings.TrimLeft(parts[0], "0")
	end = strings.TrimLeft(parts[1], "0")
	return start + "-" + end, nil
}

func (c *Crawler2) parseCredit(s string) float64 {
	// 去除"学分"后缀
	numStr := strings.TrimSuffix(s, "学分")
	credits, _ := strconv.ParseFloat(numStr, 64)
	return credits
}

// 从字符串中提取合法的教室号
func (c *Crawler2) parseClassRoom(s string) string {
	// 正则匹配：楼号只能是 3,7,8,9,10 或 n，后跟 3 位数字
	re := regexp.MustCompile(`((?:3|7|8|9|10|n)\d{3})$`)
	match := re.FindString(s)
	if match == "" {
		return s
	}
	return match
}
