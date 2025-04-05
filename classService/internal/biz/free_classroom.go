package biz

import (
	"context"
	"fmt"
	clog "github.com/asynccnu/ccnubox-be/classService/internal/log"
	"github.com/asynccnu/ccnubox-be/classService/internal/model"
	"github.com/asynccnu/ccnubox-be/classService/internal/service"
	"github.com/valyala/fastjson"
	"io"
	"net/http"
	"strings"
	"time"
)

type FreeClassRoomData interface {
	AddClassroomOccupancy(ctx context.Context, year, semester string, cwtPairs ...model.CTWPair) error
	ClearClassroomOccupancy(ctx context.Context, year, semester string) error
	GetAllClassroom(ctx context.Context, year, semester, wherePrefix string) ([]string, error)
	QueryAvailableClassrooms(ctx context.Context, year, semester string, week, day, section int, wherePrefix string) (map[string]bool, error)
}

type ClassData interface {
	GetBatchClassInfos(ctx context.Context, year, semester string, page, pageSize int) ([]model.ClassInfo, int, error)
}

type CookieClient interface {
	GetCookie(ctx context.Context, stuID string) (string, error)
}

type FreeClassroomBiz struct {
	classData         ClassData
	freeClassRoomData FreeClassRoomData
	cookieCli         CookieClient
	httpCli           *http.Client
}

func NewFreeClassroomBiz(classData ClassData, data FreeClassRoomData, cookieCli CookieClient) *FreeClassroomBiz {
	httpCli := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,              // 最大空闲连接
			IdleConnTimeout:     90 * time.Second, // 空闲连接超时
			TLSHandshakeTimeout: 10 * time.Second, // TLS握手超时
		},
		Timeout: 30 * time.Second, // 总请求超时
	}
	httpCli.Transport = &http.Transport{
		MaxIdleConnsPerHost: 20, // 每个主机最大空闲连接
	}

	return &FreeClassroomBiz{
		classData:         classData,
		freeClassRoomData: data,
		cookieCli:         cookieCli,
		httpCli:           httpCli,
	}
}

func (f *FreeClassroomBiz) ClearClassroomOccupancyFromES(ctx context.Context, year, semester string) error {
	return f.freeClassRoomData.ClearClassroomOccupancy(ctx, year, semester)
}

func (f *FreeClassroomBiz) SaveFreeClassRoomFromLocal(ctx context.Context, year, semester string) error {
	pageSize := 500 // 每批获取500条
	page := 1
	for {
		classes, total, err := f.classData.GetBatchClassInfos(ctx, year, semester, page, pageSize)
		if err != nil {
			clog.LogPrinter.Errorf("failed to get batch class infos: %v", err)
			return err
		}
		var cwtPairs []model.CTWPair
		for _, class := range classes {
			var (
				sections []int
				weeks    []int
			)
			var secStart, secEnd int
			_, err = fmt.Sscanf(class.ClassWhen, "%d-%d", &secStart, &secEnd)
			if err != nil {
				continue
			}

			for i := secStart; i <= secEnd; i++ {
				sections = append(sections, i)
			}
			for i := 1; i <= 30; i++ {
				if class.Weeks&(1<<(i-1)) != 0 {
					weeks = append(weeks, i)
				}
			}
			cwtPairs = append(cwtPairs, model.CTWPair{
				CT: model.CTime{
					Day:      int(class.Day),
					Sections: sections,
					Weeks:    weeks,
				},
				Where: class.Where,
			})
		}
		err = f.SaveFreeClassRoomInfo(ctx, year, semester, cwtPairs)
		if err != nil {
			return err
		}

		// 判断是否已经获取完所有数据
		if page*pageSize >= total || len(classes) == 0 {
			break
		}
		page++
	}
	return nil
}

func (f *FreeClassroomBiz) SaveFreeClassRoomInfo(ctx context.Context, year, semester string, cwtPairs []model.CTWPair) error {
	//添加新数据
	err := f.freeClassRoomData.AddClassroomOccupancy(ctx, year, semester, cwtPairs...)
	if err != nil {
		clog.LogPrinter.Errorf("failed to add classroom occupancy data to es: %v", err)
		return err
	}
	clog.LogPrinter.Infof("add %d classroom occupancy data to es", len(cwtPairs))
	return nil
}

func (f *FreeClassroomBiz) SearchAvailableClassroom(ctx context.Context, year, semester, stuID string, week, day int, sections []int, wherePrefix string) ([]service.AvailableClassroomStat, error) {
	var (
		classroomStats = make(map[string][]bool)
		err            error
	)

	//先获取全部的教室
	classroomSet, err := f.freeClassRoomData.GetAllClassroom(ctx, year, semester, wherePrefix)
	if err != nil {
		return nil, err
	}
	//从教务系统中爬取
	freeClassroomMp, err := f.crawFreeClassroom(ctx, year, semester, stuID, week, day, sections, wherePrefix)
	if err == nil { //如果爬取成功，则使用爬取的数据
		for _, classroom := range classroomSet {
			classroomStats[classroom] = make([]bool, len(sections))
		}
		var secIdx = make(map[int]int)
		for k, v := range sections {
			secIdx[v] = k
		}
		for sec, freeclassrooms := range freeClassroomMp {
			for _, freeclassroom := range freeclassrooms {
				classroomStats[freeclassroom][secIdx[sec]] = true
			}
		}
		return toSerializableClassroomStats(classroomStats), nil
	}
	//爬取失败就使用本地数据
	classroomStats, err = f.queryAvailableClassroomFromLocal(ctx, year, semester, week, day, sections, wherePrefix)
	if err != nil {
		return nil, err
	}
	return toSerializableClassroomStats(classroomStats), nil
}

func toSerializableClassroomStats(classroomStats map[string][]bool) []service.AvailableClassroomStat {
	var res = make([]service.AvailableClassroomStat, 0, len(classroomStats))
	for classroom, stats := range classroomStats {
		res = append(res, service.AvailableClassroomStat{
			Classroom:     classroom,
			AvailableStat: stats,
		})
	}
	return res
}

func (f *FreeClassroomBiz) queryAvailableClassroomFromLocal(ctx context.Context, year, semester string, week, day int, sections []int, wherePrefix string) (map[string][]bool, error) {
	var classroomStats = make(map[string][]bool)
	for i, section := range sections {
		availableClassrooms, err := f.freeClassRoomData.QueryAvailableClassrooms(ctx, year, semester, week, day, section, wherePrefix)
		if i == 0 {
			if err != nil {
				clog.LogPrinter.Errorf("failed to query available classrooms at the first section: %v", err)
				return nil, err
			}
			for classroom, stat := range availableClassrooms {
				classroomStats[classroom] = make([]bool, len(sections))
				classroomStats[classroom][i] = stat
			}
			continue
		}
		if err != nil {
			clog.LogPrinter.Warnf("failed to query available classrooms: %v", err)
		}
		if err == nil {
			for classroom := range classroomStats {
				classroomStats[classroom][i] = availableClassrooms[classroom]
			}
		}
	}
	return classroomStats, nil
}

// 返回每一节课的空闲教室
func (f *FreeClassroomBiz) crawFreeClassroom(ctx context.Context, year, semester, stuID string, week, day int, sections []int, wherePrefix string) (map[int][]string, error) {
	cookie, err := f.cookieCli.GetCookie(ctx, stuID)
	if err != nil {
		return nil, err
	}

	var freeClassroomMp = make(map[int][]string, len(sections))

	var mp = map[string]string{
		"1": "3",
		"2": "12",
		"3": "16",
	}

	var campus = 1
	if wherePrefix[0] == 'n' {
		campus = 2
	}

	for _, section := range sections {
		var data = strings.NewReader(fmt.Sprintf(`fwzt=cx&xqh_id=%d&xnm=%s&xqm=%s&cdlb_id=&cdejlb_id=&qszws=&jszws=&cdmc=%s&lh=&jyfs=0&cdjylx=&sfbhkc=&zcd=%d&xqj=%d&jcd=%d&_search=false&nd=%d&queryModel.showCount=1000&queryModel.currentPage=1&queryModel.sortName=cdbh+&queryModel.sortOrder=asc&time=1`,
			campus, year, mp[semester], wherePrefix, 1<<(week-1), day, 1<<(section-1), time.Now().UnixMilli()))
		req, err := http.NewRequest("POST", "https://xk.ccnu.edu.cn/jwglxt/cdjy/cdjy_cxKxcdlb.html?doType=query&gnmkdm=N2155", data)
		if err != nil {
			clog.LogPrinter.Errorf("failed to create request: %v", err)
			return nil, err
		}
		req.Header = http.Header{
			"Cookie":       []string{cookie},
			"Content-Type": []string{"application/x-www-form-urlencoded;charset=UTF-8"},
			"User-Agent":   []string{"Mozilla/5.0"}, // 精简UA
		}
		resp, err := f.httpCli.Do(req)
		if err != nil {
			clog.LogPrinter.Errorf("failed to send request: %v", err)
			return nil, err
		}
		// 读取 Body 到字节数组
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			clog.LogPrinter.Warnf("failed to read response body: %v", err)
			//记得关闭body
			resp.Body.Close()
			continue
		}
		classrooms, err := extractCdIDsWithFastjson(bodyBytes, wherePrefix)
		if err != nil {
			clog.LogPrinter.Errorf("failed to parse response body: %v", err)
			continue
		}

		freeClassroomMp[section] = classrooms

		//关闭body
		resp.Body.Close()
	}
	return freeClassroomMp, nil
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
		if cdID != nil && strings.HasPrefix(string(cdID), prefix) {
			cdIDs = append(cdIDs, string(cdID))
		}
	}
	return cdIDs, nil
}

//type JSONData struct {
//	CurrentPage   int           `json:"currentPage"`
//	CurrentResult int           `json:"currentResult"`
//	EntityOrField bool          `json:"entityOrField"`
//	Items         []Items       `json:"items"`
//	Limit         int           `json:"limit"`
//	Offset        int           `json:"offset"`
//	PageNo        int           `json:"pageNo"`
//	PageSize      int           `json:"pageSize"`
//	ShowCount     int           `json:"showCount"`
//	SortName      string        `json:"sortName"`
//	SortOrder     string        `json:"sortOrder"`
//	Sorts         []interface{} `json:"sorts"`
//	TotalCount    int           `json:"totalCount"`
//	TotalPage     int           `json:"totalPage"`
//	TotalResult   int           `json:"totalResult"`
//}
//type QueryModel struct {
//	CurrentPage   int           `json:"currentPage"`
//	CurrentResult int           `json:"currentResult"`
//	EntityOrField bool          `json:"entityOrField"`
//	Limit         int           `json:"limit"`
//	Offset        int           `json:"offset"`
//	PageNo        int           `json:"pageNo"`
//	PageSize      int           `json:"pageSize"`
//	ShowCount     int           `json:"showCount"`
//	Sorts         []interface{} `json:"sorts"`
//	TotalCount    int           `json:"totalCount"`
//	TotalPage     int           `json:"totalPage"`
//	TotalResult   int           `json:"totalResult"`
//}
//type UserModel struct {
//	Monitor    bool   `json:"monitor"`
//	RoleCount  int    `json:"roleCount"`
//	RoleKeys   string `json:"roleKeys"`
//	RoleValues string `json:"roleValues"`
//	Status     int    `json:"status"`
//	Usable     bool   `json:"usable"`
//}
//type Items struct {
//	CdID               string     `json:"cd_id"`
//	Cdbh               string     `json:"cdbh"`
//	Cdjc               string     `json:"cdjc"`
//	CdlbID             string     `json:"cdlb_id"`
//	Cdlbmc             string     `json:"cdlbmc"`
//	Cdmc               string     `json:"cdmc"`
//	CdxqxxID           string     `json:"cdxqxx_id"`
//	Date               string     `json:"date"`
//	DateDigit          string     `json:"dateDigit"`
//	DateDigitSeparator string     `json:"dateDigitSeparator"`
//	Day                string     `json:"day"`
//	Jgpxzd             string     `json:"jgpxzd"`
//	Jxlmc              string     `json:"jxlmc"`
//	Kszws1             string     `json:"kszws1"`
//	Lh                 string     `json:"lh"`
//	Listnav            string     `json:"listnav"`
//	LocaleKey          string     `json:"localeKey"`
//	Month              string     `json:"month"`
//	PageTotal          int        `json:"pageTotal"`
//	Pageable           bool       `json:"pageable"`
//	QueryModel         QueryModel `json:"queryModel"`
//	Rangeable          bool       `json:"rangeable"`
//	RowID              string     `json:"row_id"`
//	TotalResult        string     `json:"totalResult"`
//	UserModel          UserModel  `json:"userModel"`
//	XqhID              string     `json:"xqh_id"`
//	Xqmc               string     `json:"xqmc"`
//	Year               string     `json:"year"`
//	Zws                string     `json:"zws"`
//}
