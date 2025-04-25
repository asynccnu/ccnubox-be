package crawler

import (
	"context"
	"errors"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"github.com/go-kratos/kratos/v2/log"
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
	log    *log.Helper
	client *http.Client
}

func NewClassCrawler(logger log.Logger) *Crawler {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,              // 最大空闲连接
			IdleConnTimeout:     90 * time.Second, // 空闲连接超时
			TLSHandshakeTimeout: 10 * time.Second, // TLS握手超时
			DisableKeepAlives:   false,            // 确保不会意外关闭 Keep-Alive
		},
	}
	return &Crawler{
		log:    log.NewHelper(logger),
		client: client,
	}
}

// GetClassInfoForGraduateStudent 获取研究生课程信息
func (c *Crawler) GetClassInfoForGraduateStudent(ctx context.Context, r model.GetClassInfoForGraduateStudentReq) (*model.GetClassInfoForGraduateStudentResp, error) {
	return nil, errors.New("this feature is not yet implemented")
}

// GetClassInfosForUndergraduate  获取本科生课程信息
func (c *Crawler) GetClassInfosForUndergraduate(ctx context.Context, r model.GetClassInfosForUndergraduateReq) (*model.GetClassInfosForUndergraduateResp, error) {
	xnm, xqm := r.Year, r.Semester
	sendReqStart := time.Now()

	formdata := fmt.Sprintf("xnm=%s&xqm=%s&kzlx=ck&xsdm=", xnm, semesterMap[xqm])

	var data = strings.NewReader(formdata)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://xk.ccnu.edu.cn/jwglxt/kbcx/xskbcx_cxXsgrkb.html?gnmkdm=N2151", data)
	if err != nil {
		c.log.Errorf("http.NewRequestWithContext err=%v", err)
		return nil, errcode.ErrCrawler
	}
	req.Header = http.Header{
		"Cookie":       []string{r.Cookie},
		"Content-Type": []string{"application/x-www-form-urlencoded;charset=UTF-8"},
		"User-Agent":   []string{"Mozilla/5.0"}, // 精简UA
	}
	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Errorf("client.Do err=%v", err)
		return nil, errcode.ErrCrawler
	}
	defer resp.Body.Close()

	c.log.Infof("发请求耗时%v", time.Since(sendReqStart))

	// 读取 Body 到字节数组
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Errorf("failed to read response body: %v", err)
		return nil, err
	}
	infos, Scs, err := extractUndergraduateData(bodyBytes, r.StuID, xnm, xqm)
	if err != nil {
		c.log.Errorf("extractUndergraduateData err=%v", err)
		return nil, errcode.ErrCrawler
	}
	return &model.GetClassInfosForUndergraduateResp{
		ClassInfos:     infos,
		StudentCourses: Scs,
	}, nil
}

func extractUndergraduateData(rawJson []byte, stuID, xnm, xqm string) ([]*model.ClassInfo, []*model.StudentCourse, error) {
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
	var infos = make([]*model.ClassInfo, 0, length)
	var Scs = make([]*model.StudentCourse, 0, length)
	for _, kb := range kbList.GetArray() {
		if string(kb.GetStringBytes("sxbj")) != "1" {
			continue
		}
		//课程信息
		var info = &model.ClassInfo{}
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
		Sc := &model.StudentCourse{
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
