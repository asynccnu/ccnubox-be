package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/valyala/fastjson"
)

// Notice: 爬虫相关
var semesterMap = map[string]string{
	"1": "3",
	"2": "12",
	"3": "16",
}

type Crawler struct {
	pg ProxyGetter

	clientPool sync.Pool
}

func NewClassCrawler(pg ProxyGetter) *Crawler {
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

	c := &Crawler{
		clientPool: sync.Pool{
			New: newClient,
		},
		pg: pg,
	}

	return c
}

// GetClassInfoForGraduateStudent 获取研究生课程信息
func (c *Crawler) GetClassInfoForGraduateStudent(ctx context.Context, stuID, year, semester, cookie string) ([]*biz.ClassInfo, []*biz.StudentCourse, int, error) {
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
		return nil, nil, -1, errcode.ErrCrawler
	}
	defer resp.Body.Close()

	// 读取 Body 到字节数组
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logh.Errorf("failed to read response body: %v", err)
		return nil, nil, -1, err
	}
	infos, Scs, sum, err := extractUndergraduateData(bodyBytes, stuID, xnm, xqm)
	if err != nil {
		logh.Errorf("extractUndergraduateData err=%v", err)
		return nil, nil, -1, errcode.ErrCrawler
	}
	return infos, Scs, sum, nil
}

// GetClassInfosForUndergraduate  获取本科生课程信息
func (c *Crawler) GetClassInfosForUndergraduate(ctx context.Context, stuID, year, semester, cookie string) ([]*biz.ClassInfo, []*biz.StudentCourse, int, error) {
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

	formdata := fmt.Sprintf("xnm=%s&xqm=%s&kzlx=ck&xsdm=", xnm, semesterMap[xqm])

	data := strings.NewReader(formdata)

	req, err := http.NewRequestWithContext(reqCtx, "POST", "https://xk.ccnu.edu.cn/jwglxt/kbcx/xskbcx_cxXsgrkb.html?gnmkdm=N2151", data)
	if err != nil {
		logh.Errorf("http.NewRequestWithContext err=%v", err)
		return nil, nil, -1, errcode.ErrCrawler
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
		return nil, nil, -1, errcode.ErrCrawler
	}
	defer resp.Body.Close()

	// 读取 Body 到字节数组
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logh.Errorf("failed to read response body: %v", err)
		return nil, nil, -1, err
	}
	infos, Scs, sum, err := extractUndergraduateData(bodyBytes, stuID, xnm, xqm)
	if err != nil {
		logh.Errorf("extractUndergraduateData err=%v", err)
		return nil, nil, -1, errcode.ErrCrawler
	}

	return infos, Scs, sum, nil
}

func extractUndergraduateData(rawJson []byte, stuID, xnm, xqm string) ([]*biz.ClassInfo, []*biz.StudentCourse, int, error) {
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
