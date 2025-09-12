package crawler

import (
	"context"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/classLog"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/valyala/fastjson"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Notice: 爬虫相关
var semesterMap = map[string]string{
	"1": "3",
	"2": "12",
	"3": "16",
}

type Crawler struct {
	client *http.Client
}

func NewClassCrawler() *Crawler {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,              // 最大空闲连接
			IdleConnTimeout:     90 * time.Second, // 空闲连接超时
			TLSHandshakeTimeout: 10 * time.Second, // TLS握手超时
			DisableKeepAlives:   false,            // 确保不会意外关闭 Keep-Alive
		},
	}
	return &Crawler{
		client: client,
	}
}

// GetClassInfoForGraduateStudent 获取研究生课程信息
func (c *Crawler) GetClassInfoForGraduateStudent(ctx context.Context, stuID, year, semester, cookie string) ([]*biz.ClassInfo, []*biz.StudentCourse, error) {
	logh := classLog.GetLogHelperFromCtx(ctx)
	xnm, xqm := year, semester

	param := fmt.Sprintf("xnm=%s&xqm=%s", xnm, semesterMap[xqm])
	var data = strings.NewReader(param)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://grd.ccnu.edu.cn/yjsxt/kbcx/xskbcx_cxXsKb.html?gnmkdm=N2151", data)
	if err != nil {
		logh.Errorf("http.NewRequestWithContext err=%v", err)
		return nil, nil, errcode.ErrCrawler
	}
	req.Header = http.Header{
		"Cookie":       []string{cookie},
		"Content-Type": []string{"application/x-www-form-urlencoded;charset=UTF-8"},
		"User-Agent":   []string{"Mozilla/5.0"}, // 精简UA
	}
	resp, err := c.client.Do(req)
	if err != nil {
		logh.Errorf("client.Do err=%v", err)
		return nil, nil, errcode.ErrCrawler
	}
	defer resp.Body.Close()

	// 读取 Body 到字节数组
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logh.Errorf("failed to read response body: %v", err)
		return nil, nil, err
	}
	infos, Scs, err := extractUndergraduateData(bodyBytes, stuID, xnm, xqm)
	if err != nil {
		logh.Errorf("extractUndergraduateData err=%v", err)
		return nil, nil, errcode.ErrCrawler
	}
	return infos, Scs, nil
}

// GetClassInfosForUndergraduate  获取本科生课程信息
func (c *Crawler) GetClassInfosForUndergraduate(ctx context.Context, stuID, year, semester, cookie string) ([]*biz.ClassInfo, []*biz.StudentCourse, error) {
	logh := classLog.GetLogHelperFromCtx(ctx)
	xnm, xqm := year, semester

	formdata := fmt.Sprintf("xnm=%s&xqm=%s&kzlx=ck&xsdm=", xnm, semesterMap[xqm])

	var data = strings.NewReader(formdata)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://xk.ccnu.edu.cn/jwglxt/kbcx/xskbcx_cxXsgrkb.html?gnmkdm=N2151", data)
	if err != nil {
		logh.Errorf("http.NewRequestWithContext err=%v", err)
		return nil, nil, errcode.ErrCrawler
	}
	req.Header = http.Header{
		"Cookie":       []string{cookie},
		"Content-Type": []string{"application/x-www-form-urlencoded;charset=UTF-8"},
		"User-Agent":   []string{"Mozilla/5.0"}, // 精简UA
	}
	resp, err := c.client.Do(req)
	if err != nil {
		logh.Errorf("client.Do err=%v", err)
		return nil, nil, errcode.ErrCrawler
	}
	defer resp.Body.Close()

	// 读取 Body 到字节数组
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logh.Errorf("failed to read response body: %v", err)
		return nil, nil, err
	}
	infos, Scs, err := extractUndergraduateData(bodyBytes, stuID, xnm, xqm)
	if err != nil {
		logh.Errorf("extractUndergraduateData err=%v", err)
		return nil, nil, errcode.ErrCrawler
	}
	return infos, Scs, nil
}

func extractUndergraduateData(rawJson []byte, stuID, xnm, xqm string) ([]*biz.ClassInfo, []*biz.StudentCourse, error) {
	var p fastjson.Parser
	v, err := p.ParseBytes(rawJson)
	if err != nil {
		return nil, nil, err
	}
	kbList := v.Get("kbList")
	if kbList == nil || kbList.Type() != fastjson.TypeArray {
		return nil, nil, fmt.Errorf("kbList not found or not an array")
	}
	length := len(kbList.GetArray())

	infos := make([]*biz.ClassInfo, 0, length)
	Scs := make([]*biz.StudentCourse, 0, length)

	for _, kb := range kbList.GetArray() {
		// 过滤掉没确定被选上的课程
		if string(kb.GetStringBytes("sxbj")) != "1" {
			continue
		}
		//课程信息
		var info = &biz.ClassInfo{}
		info.Day, _ = strconv.ParseInt(string(kb.GetStringBytes("xqj")), 10, 64) //星期几
		info.Teacher = string(kb.GetStringBytes("xm"))
		info.Where = string(kb.GetStringBytes("cdmc"))                           //上课地点
		info.ClassWhen = string(kb.GetStringBytes("jcs"))                        //上课是第几节
		info.WeekDuration = string(kb.GetStringBytes("zcd"))                     //上课的周数
		info.Classname = string(kb.GetStringBytes("kcmc"))                       //课程名称
		info.Credit, _ = strconv.ParseFloat(string(kb.GetStringBytes("xf")), 64) //学分
		info.Semester = xqm                                                      //学期
		info.Year = xnm                                                          //学年
		//添加周数
		info.Weeks, _ = strconv.ParseInt(string(kb.GetStringBytes("oldzc")), 10, 64)
		info.JxbId = string(kb.GetStringBytes("jxb_id")) //教学班ID
		info.UpdateID()                                  //课程ID

		//为防止其时间过于紧凑
		//选择在这里直接给时间赋值
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
		infos = append(infos, info) //添加课程
		Scs = append(Scs, Sc)       //添加"学生与课程的映射关系"
	}
	return infos, Scs, nil
}
