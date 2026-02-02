package crawler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/pkg/tool"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/valyala/fastjson"
)

var (
	// 正则匹配：楼号只能是 3,7,8,9,10 或 n，后跟 3 位数字
	parseClassRoomRegex = regexp.MustCompile(`((?:3|7|8|9|10|n)\d{3})$`)
	parseNumberRegex    = regexp.MustCompile(`\d+`)
)

// Crawler2 爬取的是对于智慧教务的“培养服务/我的课表/学期课表/个人课表信息”那个HTML
type Crawler2 struct {
	pg ProxyGetter

	clientPool sync.Pool
}

func NewClassCrawler2(pg ProxyGetter) *Crawler2 {
	newClient := func() interface{} {
		return &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10, // 既然使用sync.Pool管理对象，这个不宜过大
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
				DisableKeepAlives:   false,
				Proxy: func(req *http.Request) (*url.URL, error) {
					// 从 request 的 context 中获取代理地址
					if p, ok := req.Context().Value("proxy_url").(*url.URL); ok {
						return p, nil
					}
					return nil, nil
				},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		}
	}

	c2 := &Crawler2{
		clientPool: sync.Pool{
			New: newClient,
		},
		pg: pg,
	}

	return c2
}

func (c *Crawler2) GetClassInfosForUndergraduate(ctx context.Context, stuID, year, semester, cookie string) ([]*biz.ClassInfo, []*biz.StudentCourse, int, error) {
	// 获取代理并将其存入请求的上下文
	proxyURL := c.pg.GetProxy(ctx)

	var reqCtx context.Context
	reqCtx = ctx
	if proxyURL != nil {
		reqCtx = context.WithValue(ctx, "proxy_url", proxyURL)
	}

	// 使用连接池获取 HTTP 客户端
	client := c.clientPool.Get().(*http.Client)
	defer c.clientPool.Put(client)

	logh := logger.GetLoggerFromCtx(ctx)
	classURL := fmt.Sprintf(
		"https://bkzhjw.ccnu.edu.cn/jsxsd/framework/mainV_index_loadkb.htmlx?zc=&kbjcmsid=16FD8C2BE55E15F9E0630100007FF6B5&xnxq01id=%s&xswk=false",
		c.getys(year, semester))

	req, err := http.NewRequestWithContext(reqCtx, "GET", classURL, nil)
	if err != nil {
		logh.Errorf("http.NewRequest err=%v", err)
		return nil, nil, -1, err
	}
	req.Header = http.Header{
		"Accept":             []string{"*/*"},
		"Accept-Language":    []string{"zh-CN,zh;q=0.9,en;q=0.8"},
		"Connection":         []string{"keep-alive"},
		"Content-Type":       []string{"application/x-www-form-urlencoded;charset=UTF-8"},
		"Origin":             []string{"https://xk.ccnu.edu.cn"},
		"Referer":            []string{"https://xk.ccnu.edu.cn/jwglxt/kbcx/xskbcx_cxXskbcxIndex.html?gnmkdm=N2151&layout=default"},
		"Sec-Ch-Ua":          []string{`"Chromium";v="142", "Google Chrome";v="142", "Not_A Brand";v="99"`},
		"Sec-Ch-Ua-Mobile":   []string{"?0"},
		"Sec-Ch-Ua-Platform": []string{`"Windows"`},
		"Sec-Fetch-Dest":     []string{"empty"},
		"Sec-Fetch-Mode":     []string{"cors"},
		"Sec-Fetch-Site":     []string{"same-origin"},
		"Cookie":             []string{cookie},
		"User-Agent":         []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"},
		"X-Requested-With":   []string{"XMLHttpRequest"},
	}
	resp, err := client.Do(req)
	if err != nil {
		logh.Errorf("client.Do err=%v", err)
		return nil, nil, -1, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logh.Errorf("read body failed:%v", err)
		return nil, nil, -1, err
	}

	infos, err := c.extractCourses(ctx, year, semester, body)
	if err != nil {
		logh.Errorf("failed to extract infos: %v", err)
		return nil, nil, -1, fmt.Errorf("failed to extract infos: %v", err)
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
	sum := len(infos)

	return infos, scs, sum, nil
}

func (c *Crawler2) GetClassInfoForGraduateStudent(ctx context.Context, stuID, year, semester, cookie string) ([]*biz.ClassInfo, []*biz.StudentCourse, int, error) {
	// 获取代理并将其存入请求的上下文
	proxyURL := c.pg.GetProxy(ctx)

	var reqCtx context.Context
	reqCtx = ctx
	if proxyURL != nil {
		reqCtx = context.WithValue(ctx, "proxy_url", proxyURL)
	}

	// 使用连接池获取 HTTP 客户端
	client := c.clientPool.Get().(*http.Client)
	defer c.clientPool.Put(client)

	logh := logger.GetLoggerFromCtx(ctx)
	xnm, xqm := year, semester

	param := fmt.Sprintf("xnm=%s&xqm=%s", xnm, semesterMap[xqm])
	data := strings.NewReader(param)

	req, err := http.NewRequestWithContext(reqCtx, "POST", "https://grd.ccnu.edu.cn/yjsxt/kbcx/xskbcx_cxXsKb.html?gnmkdm=N2151", data)
	if err != nil {
		logh.Errorf("http.NewRequestWithContext err=%v", err)
		return nil, nil, -1, errcode.ErrCrawler
	}
	req.Header = http.Header{
		"Cookie":       []string{cookie},
		"Content-Type": []string{"application/x-www-form-urlencoded;charset=UTF-8"},
		"User-Agent":   []string{"Mozilla/5.0"}, // 精简UA
	}
	resp, err := client.Do(req)
	if err != nil {
		logh.Errorf("client.Do err=%v", err)
		return nil, nil, -1, errcode.ErrCrawler
	}
	defer resp.Body.Close()

	// 读取 Body 到字节数组
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logh.Errorf("failed to read response body: %v", err)
		return nil, nil, -1, err
	}
	infos, Scs, sum, err := extractGraduateData(bodyBytes, stuID, xnm, xqm)
	if err != nil {
		logh.Errorf("extractUndergraduateData err=%v", err)
		return nil, nil, -1, errcode.ErrCrawler
	}
	return infos, Scs, sum, nil
}

func (c *Crawler2) getys(year, semester string) string {
	// 将年份字符串转为整数
	y, _ := strconv.Atoi(year)

	// 组合结果
	return fmt.Sprintf("%d-%d-%s", y, y+1, semester)
}

func (c *Crawler2) extractCourses(ctx context.Context, year, semester string, html []byte) ([]*biz.ClassInfo, error) {
	logh := logger.GetLoggerFromCtx(ctx)
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("NewDocumentFromReader err: %v", err)
	}

	var classInfos []*biz.ClassInfo

	doc.Find("li.qz-toolitiplists").Each(func(i int, selection *goquery.Selection) {
		var classInfo biz.ClassInfo

		classInfo.Year, classInfo.Semester = year, semester
		classInfo.Classname = selection.Find(".qz-tooltipContent-title").Text()
		classInfo.UpdatedAt = time.Now()
		classInfo.CreatedAt = classInfo.UpdatedAt

		// 获取jxb_id
		selection.Find(`img[src*="ewmck"]`).Each(func(_ int, img *goquery.Selection) {
			src, ok := img.Attr("src")
			if !ok {
				return
			}
			re := regexp.MustCompile(`id=(\d+)`)
			m := re.FindStringSubmatch(src)
			if len(m) == 2 {
				classInfo.JxbId = m[1]
			}
		})

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
				// 重新格式化week
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
	// 使用字符串操作
	logh := logger.GetLoggerFromCtx(ctx)
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
	matches := parseNumberRegex.FindAllString(s, -1)
	var numbers []int64
	for _, match := range matches {
		num, _ := strconv.Atoi(match)
		numbers = append(numbers, int64(num))
	}
	return numbers
}

func (c *Crawler2) parseDay(ctx context.Context, s string) string {
	logh := logger.GetLoggerFromCtx(ctx)
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
	match := parseClassRoomRegex.FindString(s)
	if match == "" {
		return s
	}
	return match
}

func extractGraduateData(rawJson []byte, stuID, xnm, xqm string) ([]*biz.ClassInfo, []*biz.StudentCourse, int, error) {
	var p fastjson.Parser
	v, err := p.ParseBytes(rawJson)
	if err != nil {
		return nil, nil, -1, err
	}
	kbList := v.Get("kbList")
	if kbList == nil || kbList.Type() != fastjson.TypeArray {
		return nil, nil, -1, fmt.Errorf("kbList not found or not an array")
	}
	length := len(kbList.GetArray())

	infos := make([]*biz.ClassInfo, 0, length)
	Scs := make([]*biz.StudentCourse, 0, length)
	sum := v.GetInt("xsxx", "KCMS")

	for _, kb := range kbList.GetArray() {
		// 过滤掉没确定被选上的课程
		if string(kb.GetStringBytes("sxbj")) != "1" {
			continue
		}
		// 课程信息
		info := &biz.ClassInfo{}
		info.Day, _ = strconv.ParseInt(string(kb.GetStringBytes("xqj")), 10, 64) // 星期几
		info.Teacher = string(kb.GetStringBytes("xm"))
		info.Where = string(kb.GetStringBytes("cdmc"))                           // 上课地点
		info.ClassWhen = string(kb.GetStringBytes("jcs"))                        // 上课是第几节
		info.WeekDuration = string(kb.GetStringBytes("zcd"))                     // 上课的周数
		info.Classname = string(kb.GetStringBytes("kcmc"))                       // 课程名称
		info.Credit, _ = strconv.ParseFloat(string(kb.GetStringBytes("xf")), 64) // 学分
		info.Semester = xqm                                                      // 学期
		info.Year = xnm                                                          // 学年
		// 添加周数
		info.Weeks, _ = strconv.ParseInt(string(kb.GetStringBytes("oldzc")), 10, 64)
		info.JxbId = string(kb.GetStringBytes("jxb_id")) // 教学班ID
		info.UpdateID()                                  // 课程ID

		// 为防止其时间过于紧凑
		// 选择在这里直接给时间赋值
		info.CreatedAt, info.UpdatedAt = time.Now(), time.Now()

		//-----------------------------------------------------
		//学生与课程的映射关系
		Sc := &biz.StudentCourse{
			StuID:           stuID,
			ClaID:           info.ID,
			Year:            xnm,
			Semester:        xqm,
			IsManuallyAdded: false,
		}
		infos = append(infos, info) // 添加课程
		Scs = append(Scs, Sc)       // 添加"学生与课程的映射关系"
	}
	return infos, Scs, sum, nil
}
