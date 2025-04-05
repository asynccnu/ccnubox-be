package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/classLog"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/errcode"
	model2 "github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/model"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/pkg/tool"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Notice: 爬虫相关
var mp = map[string]string{
	"1": "3",
	"2": "12",
	"3": "16",
}

type Crawler struct {
	log classLog.Clogger
}

func NewClassCrawler(logger classLog.Clogger) *Crawler {
	return &Crawler{
		log: logger,
	}
}

// GetClassInfoForGraduateStudent 获取研究生课程信息
func (c *Crawler) GetClassInfoForGraduateStudent(ctx context.Context, r model2.GetClassInfoForGraduateStudentReq) (*model2.GetClassInfoForGraduateStudentResp, error) {
	var (
		reply CrawReply2
		Xnm   = r.Year
		Xqm   = r.Semester
		yn    = tool.CheckSY(Xqm, Xnm)
	)

	if !yn {
		return nil, errcode.ErrParam
	}
	client := &http.Client{}
	tmp1 := GetXNM(Xnm)
	tmp2 := GetXQM(Xqm)
	param := fmt.Sprintf("xnm=%s&xqm=%s", tmp1, tmp2)
	var data = strings.NewReader(param)
	req, err := http.NewRequest("POST", "https://grd.ccnu.edu.cn/yjsxt/kbcx/xskbcx_cxXsKb.html?gnmkdm=N2151", data)
	if err != nil {
		c.log.Errorw(classLog.Msg, "func:http.NewRequest err",
			classLog.Param, fmt.Sprintf("%v,%v,%v", "POST", "https://grd.ccnu.edu.cn/yjsxt/kbcx/xskbcx_cxXsKb.html?gnmkdm=N2151", data),
			classLog.Reason, err)
		return nil, errcode.ErrCrawler
	}
	req.Header.Set("Cookie", r.Cookie)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("Origin", "https://grd.ccnu.edu.cn")
	req.Header.Set("Referer", "https://grd.ccnu.edu.cn/yjsxt/kbcx/xskbcx_cxXskbcxIndex.html?gnmkdm=N2151&layout=default")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 Edg/127.0.0.0")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("sec-ch-ua", `"Not)A;Brand";v="99", "Microsoft Edge";v="127", "Chromium";v="127"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	resp, err := client.Do(req)
	if err != nil {
		c.log.Errorw(classLog.Msg, "http request send err",
			classLog.Reason, err)
		return nil, errcode.ErrCrawler
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&reply)
	if err != nil {
		c.log.Errorw(classLog.Msg, "json decode err",
			classLog.Reason, err)
		return nil, errcode.ErrCrawler
	}
	infos, Scs, err := ToClassInfo2(reply, Xnm, Xqm)
	if err != nil {
		c.log.Errorw(classLog.Msg, "func:ToClassInfo2",
			classLog.Param, fmt.Sprintf("%v,%v,%v", reply, Xnm, Xqm),
			classLog.Reason, err)
		return nil, errcode.ErrCrawler
	}
	return &model2.GetClassInfoForGraduateStudentResp{
		ClassInfos:     infos,
		StudentCourses: Scs,
	}, nil
}

// GetClassInfosForUndergraduate  获取本科生课程信息
func (c *Crawler) GetClassInfosForUndergraduate(ctx context.Context, r model2.GetClassInfosForUndergraduateReq) (*model2.GetClassInfosForUndergraduateResp, error) {
	var (
		reply  CrawReply1
		Xnm    = r.Year
		Xqm    = r.Semester
		yn     = tool.CheckSY(Xqm, Xnm)
		client = &http.Client{}
	)

	if !yn {
		return nil, errcode.ErrParam
	}
	tmp1 := GetXNM(Xnm)
	tmp2 := GetXQM(Xqm)
	formdata := fmt.Sprintf("xnm=%s&xqm=%s&kzlx=ck&xsdm=", tmp1, tmp2)
	var data = strings.NewReader(formdata)
	req, err := http.NewRequest("POST", "https://xk.ccnu.edu.cn/jwglxt/kbcx/xskbcx_cxXsgrkb.html?gnmkdm=N2151", data)
	if err != nil {
		c.log.Errorw(classLog.Msg, "func:http.NewRequest err",
			classLog.Param, fmt.Sprintf("%v,%v,%v", "POST", "https://xk.ccnu.edu.cn/jwglxt/kbcx/xskbcx_cxXsgrkb.html?gnmkdm=N2151", data),
			classLog.Reason, err)
		return nil, errcode.ErrCrawler
	}
	req.Header.Set("Cookie", r.Cookie) //设置cookie
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("Origin", "https://xk.ccnu.edu.cn")
	req.Header.Set("Referer", "https://xk.ccnu.edu.cn/jwglxt/kbcx/xskbcx_cxXskbcxIndex.html?gnmkdm=N2151&layout=default")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 Edg/123.0.0.0")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("sec-ch-ua", `"Microsoft Edge";v="123", "Not:A-Brand";v="8", "Chromium";v="123"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	resp, err := client.Do(req)
	if err != nil {
		c.log.Errorw(classLog.Msg, "http request send err",
			classLog.Reason, err)
		return nil, errcode.ErrCrawler
	}
	defer resp.Body.Close()
	//fmt.Println(resp)
	err = json.NewDecoder(resp.Body).Decode(&reply)
	if err != nil {
		c.log.Errorw(classLog.Msg, "json decode err",
			classLog.Reason, err)
		return nil, errcode.ErrCrawler
	}
	infos, Scs, err := ToClassInfo1(reply, Xnm, Xqm)
	if err != nil {
		c.log.Errorw(classLog.Msg, "func:ToClassInfo2",
			classLog.Param, fmt.Sprintf("%v,%v,%v", reply, Xnm, Xqm),
			classLog.Reason, err)
		return nil, errcode.ErrCrawler
	}
	return &model2.GetClassInfosForUndergraduateResp{
		ClassInfos:     infos,
		StudentCourses: Scs,
	}, nil
}

// ToClassInfo1 处理本科生
func ToClassInfo1(reply CrawReply1, xnm, xqm string) ([]*model2.ClassInfo, []*model2.StudentCourse, error) {
	var infos = make([]*model2.ClassInfo, 0)
	var Scs = make([]*model2.StudentCourse, 0)
	for _, v := range reply.KbList {
		//这个字段为1代表这个课程是已经选上的
		//所以防止把没选上的课程给包含进来
		if v.Sxbj != "1" {
			continue
		}

		//课程信息
		var info = &model2.ClassInfo{}
		//var Sc = &biz.StudentCourse{}
		//info.ClassId = v.Kch //课程编号
		//info.StuID = reply.Xsxx.Xh                    //学号
		info.Day, _ = strconv.ParseInt(v.Xqj, 10, 64) //星期几
		info.Teacher = v.Xm                           //教师姓名
		info.Where = v.Cdmc                           //上课地点
		info.ClassWhen = v.Jcs                        //上课是第几节
		info.WeekDuration = v.Zcd                     //上课的周数
		info.Classname = v.Kcmc                       //课程名称
		info.Credit, _ = strconv.ParseFloat(v.Xf, 64) //学分
		info.Semester = xqm                           //学期
		info.Year = xnm                               //学年
		//添加周数
		info.Weeks, _ = strconv.ParseInt(v.Oldzc, 10, 64)
		info.JxbId = v.JxbID //教学班ID
		info.UpdateID()      //课程ID

		//为防止其时间过于紧凑
		//选择在这里直接给时间赋值
		info.CreatedAt, info.UpdatedAt = time.Now(), time.Now()

		//-----------------------------------------------------
		//学生与课程的映射关系
		Sc := &model2.StudentCourse{
			StuID:           reply.Xsxx.Xh,
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

// ToClassInfo2 处理研究生
func ToClassInfo2(reply CrawReply2, xnm, xqm string) ([]*model2.ClassInfo, []*model2.StudentCourse, error) {
	var infos = make([]*model2.ClassInfo, 0)
	var Scs = make([]*model2.StudentCourse, 0)
	for _, v := range reply.KbList {
		//课程信息
		var info = &model2.ClassInfo{}
		//var Sc = &biz.StudentCourse{}
		//info.ClassId = v.Kch //课程编号
		//info.StuID = reply.Xsxx.Xh                    //学号
		info.Day, _ = strconv.ParseInt(v.Xqj, 10, 64) //星期几
		info.Teacher = v.Xm                           //教师姓名
		info.Where = v.Cdmc                           //上课地点
		info.ClassWhen = v.Jcs                        //上课是第几节
		info.WeekDuration = v.Zcd                     //上课的周数
		info.Classname = v.Kcmc                       //课程名称
		info.Credit, _ = strconv.ParseFloat(v.Xf, 64) //学分
		info.Semester = xqm                           //学期
		info.Year = xnm                               //学年
		//添加周数
		info.Weeks, _ = strconv.ParseInt(v.Oldzc, 10, 64)
		info.UpdateID() //课程ID

		//为防止其时间过于紧凑
		//选择在这里直接给时间赋值
		info.CreatedAt, info.UpdatedAt = time.Now(), time.Now()

		//-----------------------------------------------------
		//学生与课程的映射关系
		Sc := &model2.StudentCourse{
			StuID:           reply.Xsxx.Xh,
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
func GetXNM(s string) string {
	// // 定义正则表达式模式
	// re := regexp.MustCompile(`^(\d{4})-\d{4}$`)

	// // 查找字符串中与正则表达式模式匹配的部分
	// matches := re.FindStringSubmatch(s)

	// // 检查是否匹配成功
	// if len(matches) > 1 {
	// 	return matches[1] // 第一个捕获组是我们需要的部分
	// }
	return s
}
func GetXQM(s string) string {
	return mp[s]
}
