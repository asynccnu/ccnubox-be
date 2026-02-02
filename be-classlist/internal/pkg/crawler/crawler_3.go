package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/pkg/tool"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/valyala/fastjson"
)

var (
	parseTimeDescRegexp = regexp.MustCompile(
		`^(周[一二三四五六日天])第([\d、\-]+)节\{第([^}]+)周}`,
	)
)

// Crawler3 爬取的是对于智慧教务的“培养服务/我的课表/学期课表/有课表课程”那个列表
// 注意：Crawler3开始才有课程性质这个字段
type Crawler3 struct {
	pg ProxyGetter

	clientPool sync.Pool
}

func NewClassCrawler3(pg ProxyGetter) *Crawler3 {
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

	c2 := &Crawler3{
		clientPool: sync.Pool{
			New: newClient,
		},
		pg: pg,
	}

	return c2
}

func (c *Crawler3) GetClassInfosForUndergraduate(ctx context.Context, stuID, year, semester, cookie string) ([]*biz.ClassInfo, []*biz.StudentCourse, int, error) {
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

	// 构造请求URL
	base, _ := url.Parse("https://bkzhjw.ccnu.edu.cn/jsxsd/xskb/xskb_list.do")
	q := base.Query()
	q.Set("viweType", "1")
	q.Set("needData", "1")
	q.Set("pageNum", "1")
	q.Set("pageSize", "84")
	q.Set("demoStr", "")
	q.Set("baseUrl", "/jsxsd")
	q.Set("sfykb", "2")
	q.Set("xsflMapListJsonStr", "授课,实验(实践),课外,实习,研讨,")
	q.Set("xnxq01id", c.getys(year, semester))
	q.Set("zc", "")
	q.Set("kbjcmsid", "16FD8C2BE55E15F9E0630100007FF6B5")
	base.RawQuery = q.Encode()
	classURL := base.String()

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

func (c *Crawler3) getys(year, semester string) string {
	// 将年份字符串转为整数
	y, _ := strconv.Atoi(year)

	// 组合结果
	return fmt.Sprintf("%d-%d-%s", y, y+1, semester)
}

func (c *Crawler3) extractCourses(ctx context.Context, year, semester string, res []byte) ([]*biz.ClassInfo, error) {
	var p fastjson.Parser
	v, err := p.ParseBytes(res)
	if err != nil {
		return nil, err
	}

	// 检查code
	code := v.GetInt("code")
	if code != 0 {
		return nil, fmt.Errorf("code not 0: %d", code)
	}

	// 获取data
	data := v.Get("data")
	if data == nil || data.Type() != fastjson.TypeArray {
		return nil, fmt.Errorf("data not found or not an array")
	}

	list := data.GetArray()
	infos := make([]*biz.ClassInfo, 0, len(list))

	for _, item := range list {
		where := string(item.GetStringBytes("skddmc"))                // 上课地点
		classTimeDescription := string(item.GetStringBytes("sktime")) // 上课时间描述

		basics, err := c.parseWhereAndClassTimeDescription(ctx, where, classTimeDescription)
		if err != nil {
			return nil, fmt.Errorf("failed to parse where and class time description: %w", err)
		}
		for _, basic := range basics {
			info := &biz.ClassInfo{}

			info.Classname = string(item.GetStringBytes("kc_mc"))  // 课程名
			info.Teacher = string(item.GetStringBytes("jg0101mc")) // 教师
			info.Nature = string(item.GetStringBytes("kcxz"))      // 课程性质
			info.JxbId = string(item.GetStringBytes("jx0404id"))   // 教学班ID
			info.Credit = item.GetFloat64("xf")                    // 学分
			info.Semester = semester
			info.Year = year

			info.Where = basic.where
			info.WeekDuration = basic.weekDuration
			info.ClassWhen = basic.classWhen
			info.Day = int64(basic.day)
			info.Weeks = basic.weeks

			// 生成课程ID
			info.UpdateID()

			info.CreatedAt = time.Now()
			info.UpdatedAt = time.Now()

			infos = append(infos, info)
		}

	}

	return infos, nil
}

type basicInfo struct {
	where        string
	weekDuration string
	weeks        int64
	classWhen    string
	day          int
}

func (c *Crawler3) parseWhereAndClassTimeDescription(ctx context.Context, where, classTimeDescription string) ([]basicInfo, error) {
	var res []basicInfo

	wheres := strings.Split(where, ";")
	classTimes := strings.Split(classTimeDescription, ";")

	if len(wheres) != len(classTimes) {
		return nil, errors.New("mismatched where and class time description lengths")
	}

	for i := 0; i < len(wheres); i++ {
		day, sections, weeks, err := c.parseTimeDesc(classTimes[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse time description: %w", err)
		}

		var week int64
		for _, w := range weeks {
			week |= 1 << (w - 1)
		}
		res = append(res, basicInfo{
			where:        wheres[i],
			weekDuration: tool.FormatWeeks(weeks),
			weeks:        week,
			classWhen:    fmt.Sprintf("%d-%d", sections[0], sections[len(sections)-1]),
			day:          weekdayMap[day],
		})
	}
	return res, nil
}

func (c *Crawler3) parseTimeDesc(s string) (day string, sections []int, weeks []int, err error) {
	m := parseTimeDescRegexp.FindStringSubmatch(s)
	if len(m) != 4 {
		return "", nil, nil, fmt.Errorf("invalid time format: %s", s)
	}

	day = m[1]
	if _, ok := weekdayMap[day]; !ok {
		return "", nil, nil, fmt.Errorf("invalid day format about parse day: %s", s)
	}

	// 解析节次（支持 3、4 / 9-11 / 3、5-7 混合）
	secStr := m[2]
	secParts := strings.Split(secStr, "、")
	for _, part := range secParts {
		if strings.Contains(part, "-") {
			r := strings.Split(part, "-")
			if len(r) != 2 {
				return "", nil, nil, fmt.Errorf("invalid section range: %s", s)
			}
			start, err1 := strconv.Atoi(r[0])
			end, err2 := strconv.Atoi(r[1])
			if err1 != nil || err2 != nil || start <= 0 || end < start {
				return "", nil, nil, fmt.Errorf("invalid section range: %s", s)
			}
			for i := start; i <= end; i++ {
				sections = append(sections, i)
			}
		} else {
			v, err := strconv.Atoi(part)
			if err != nil || v <= 0 {
				return "", nil, nil, fmt.Errorf("invalid section value: %s", s)
			}
			sections = append(sections, v)
		}
	}

	if len(sections) == 0 {
		return "", nil, nil, fmt.Errorf("invalid time format about parse sections: %s", s)
	}
	sort.Ints(sections)

	// 解析周次（支持 1-17 / 2,4,6 / 1,3-11,13-17 混合）
	weekStr := m[3]
	parts := strings.Split(weekStr, ",")
	for _, part := range parts {
		if strings.Contains(part, "-") {
			r := strings.Split(part, "-")
			if len(r) != 2 {
				return "", nil, nil, fmt.Errorf("invalid week range: %s", s)
			}
			start, err1 := strconv.Atoi(r[0])
			end, err2 := strconv.Atoi(r[1])
			if err1 != nil || err2 != nil || start <= 0 || end < start {
				return "", nil, nil, fmt.Errorf("invalid week range: %s", s)
			}
			for i := start; i <= end; i++ {
				weeks = append(weeks, i)
			}
		} else {
			v, err := strconv.Atoi(part)
			if err != nil || v <= 0 {
				return "", nil, nil, fmt.Errorf("invalid week value: %s", s)
			}
			weeks = append(weeks, v)
		}
	}

	if len(weeks) == 0 {
		return "", nil, nil, fmt.Errorf("invalid time format about parse weeks: %s", s)
	}

	// 去重 + 排序，防止 1,1,2-3 这种
	sort.Ints(weeks)
	uniq := weeks[:0]
	for i, w := range weeks {
		if i == 0 || w != weeks[i-1] {
			uniq = append(uniq, w)
		}
	}
	weeks = uniq

	return
}

func (c *Crawler3) GetClassInfoForGraduateStudent(ctx context.Context, stuID, year, semester, cookie string) ([]*biz.ClassInfo, []*biz.StudentCourse, int, error) {
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
	var data = strings.NewReader(param)

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
