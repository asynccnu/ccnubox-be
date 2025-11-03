package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/be-class/internal/lock"
	clog "github.com/asynccnu/ccnubox-be/be-class/internal/log"
	"github.com/asynccnu/ccnubox-be/be-class/internal/model"
	"github.com/asynccnu/ccnubox-be/be-class/internal/service"
	"github.com/valyala/fastjson"
)

const (
	FreeClassRoomCacheKeyPrefix = "ccnubox_freeclassroom"
	TimeForCache                = 5 * time.Second //缓存的超时时间
	TimeForNext                 = 5 * time.Second
	TimeForWaitForCookie        = 5 * time.Minute
	Expire                      = 7 * 24 * time.Hour //缓存数据的时长
)

type FreeClassRoomData interface {
	AddClassroomOccupancy(ctx context.Context, year, semester string, cwtPairs ...model.CTWPair) error
	ClearClassroomOccupancy(ctx context.Context, year, semester string) error
	GetAllClassroom(ctx context.Context, wherePrefix string) ([]string, error)
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
	lockBuilder       lock.Builder
	cache             Cache
	httpCli           *http.Client
}

func NewFreeClassroomBiz(classData ClassData, data FreeClassRoomData, cookieCli CookieClient, lockBuilder lock.Builder, cache Cache) *FreeClassroomBiz {
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
		lockBuilder:       lockBuilder,
		cache:             cache,
	}
}

func (f *FreeClassroomBiz) ClearClassroomOccupancyFromES(ctx context.Context, year, semester string) error {
	return f.freeClassRoomData.ClearClassroomOccupancy(ctx, year, semester)
}

// SaveFreeClassRoomFromLocal 保存空教室信息从本地ES
func (f *FreeClassroomBiz) SaveFreeClassRoomFromLocal(ctx context.Context, year, semester string) error {
	const pageSize = 500 // 每批获取500条
	page := 1
	var tasks []string

	defer func() {
		_ = f.cache.Del(ctx, tasks...)
	}()

	for {
		classes, total, err := f.classData.GetBatchClassInfos(ctx, year, semester, page, pageSize)
		if err != nil {
			clog.LogPrinter.Errorf("failed to get batch classlist infos: %v", err)
			return err
		}
		if len(classes) == 0 {
			clog.LogPrinter.Warnf("get class from local es, but the length of res is 0")
			return nil
		}

		// 加锁
		lockKey := fmt.Sprintf("save_free_classroom_%v_%v_%v", year, semester, page)
		locker := f.lockBuilder.Build(lockKey)

		lockErr := locker.Lock()
		if lockErr != nil {
			clog.LogPrinter.Infof("Error don't get lock %v: %v", lockKey, lockErr)
			// 判断是否已经获取完所有数据
			if page*pageSize >= total {
				break
			}
			page++
			continue
		}

		clog.LogPrinter.Infof("Lock %v success", lockKey)

		taskName := "task:" + lockKey
		tasks = append(tasks, taskName)

		status, err := f.cache.Get(ctx, taskName)
		if err == nil && status == Finished {
			clog.LogPrinter.Infof("task %v is finished", taskName)

			// 解锁
			ok, err1 := locker.Unlock()
			if err1 != nil || !ok {
				clog.LogPrinter.Errorf("failed to unlock lock %v: %v", lockKey, err1)
			} else {
				clog.LogPrinter.Infof("unlock %v successfully", lockKey)
			}

			// 判断是否已经获取完所有数据
			if page*pageSize >= total {
				break
			}
			page++
			continue
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
			// 设置task任务状态为failed
			err1 := f.cache.Set(ctx, taskName, Failed, 10*time.Minute)
			if err1 != nil {
				clog.LogPrinter.Errorf("failed to set cache %v: %v", taskName, err1)
			}
			return err
		}

		// 设置task任务状态为finished
		err = f.cache.Set(ctx, taskName, Finished, 10*time.Minute)
		if err != nil {
			clog.LogPrinter.Errorf("failed to set cache %v: %v", taskName, err)
		}

		// 解锁
		ok, err := locker.Unlock()
		if err != nil || !ok {
			clog.LogPrinter.Errorf("failed to unlock lock %v: %v", lockKey, err)
		} else {
			clog.LogPrinter.Infof("unlock %v successfully", lockKey)
		}

		// 判断是否已经获取完所有数据
		if page*pageSize >= total {
			break
		}
		page++
	}
	return nil
}

func (f *FreeClassroomBiz) SaveFreeClassRoomInfo(ctx context.Context, year, semester string, cwtPairs []model.CTWPair) error {
	if len(cwtPairs) == 0 {
		clog.LogPrinter.Warnf("no classroom occupancy data to save")
		return nil
	}

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
	classroomSet, err := f.freeClassRoomData.GetAllClassroom(ctx, wherePrefix)
	if err != nil {
		return nil, err
	}
	//从教务系统中爬取
	freeClassroomMp, err := f.getFreeClassrooms(ctx, year, semester, stuID, week, day, sections, wherePrefix)
	if err == nil {
		//如果爬取成功，则使用爬取的数据
		for _, classroom := range classroomSet {
			classroomStats[classroom] = make([]bool, len(sections))
		}
		var secIdx = make(map[int]int)
		for k, v := range sections {
			secIdx[v] = k
		}
		for sec, freeclassrooms := range freeClassroomMp {
			for _, freeclassroom := range freeclassrooms {
				if stats, ok := classroomStats[freeclassroom]; ok {
					stats[secIdx[sec]] = true
				}
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
func (f *FreeClassroomBiz) getFreeClassrooms(ctx context.Context, year, semester, stuID string, week, day int, sections []int, wherePrefix string) (map[int][]string, error) {
	var freeClassroomMp = make(map[int][]string, len(sections))

	var campus = 1
	if wherePrefix[0] == 'n' {
		campus = 2
	}
	preYear := strings.Split(year, "-")[0]

	// 先从缓存拿数据
	freeClassroomCache, err := f.GetFreeClassRoomFromCache(ctx, preYear, semester, week, campus, day, sections, wherePrefix)
	if err == nil {
		return freeClassroomCache, nil
	}

	// 到这里就是缓存全部没有命中
	// 预定就是先提前缓存，所以一般都是命中缓存
	// 当然如果缓存重启什么的，就会触发后面的流程

	cookie, err := f.cookieCli.GetCookie(ctx, stuID)
	if err != nil {
		return nil, err
	}

	for _, section := range sections {
		classrooms, err := f.sendReqFindFreeClassRoom(campus, preYear, semester, wherePrefix, week, day, section, cookie)
		if err != nil {
			return nil, err
		}

		freeClassroomMp[section] = classrooms
	}

	// 加载查询周所有空教室
	go f.LoadOneWeekFreeClassRoom(context.Background(), stuID, preYear, semester, week)

	return freeClassroomMp, nil
}

func (f *FreeClassroomBiz) sendReqFindFreeClassRoom(campus int, preYear, semester, wherePrefix string, week, day, section int, cookie string) ([]string, error) {
	const pageCnt = 1000

	var mp = map[string]string{
		"1": "3",
		"2": "12",
		"3": "16",
	}

	var data = strings.NewReader(fmt.Sprintf(`fwzt=cx&xqh_id=%d&xnm=%s&xqm=%s&cdlb_id=&cdejlb_id=&qszws=&jszws=&cdmc=%s&lh=&jyfs=0&cdjylx=&sfbhkc=&zcd=%d&xqj=%d&jcd=%d&_search=false&nd=%d&queryModel.showCount=%d&queryModel.currentPage=1&queryModel.sortName=cdbh+&queryModel.sortOrder=asc&time=1`,
		campus, preYear, mp[semester], wherePrefix, 1<<(week-1), day, 1<<(section-1), time.Now().UnixMilli(), pageCnt))
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
	defer resp.Body.Close()

	if err != nil {
		clog.LogPrinter.Warnf("failed to read response body: %v", err)
		return nil, nil
	}

	classrooms, err := extractCdIDsWithFastjson(bodyBytes, wherePrefix)
	if err != nil {
		clog.LogPrinter.Errorf("failed to parse response body: %v", err)
		return nil, nil
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
		// 不可以用cd_id,部分数据(南湖)会是乱码不是教室号，改用cdbh
		cdID := item.GetStringBytes("cdbh")
		if cdID != nil && strings.HasPrefix(string(cdID), prefix) {
			cdIDs = append(cdIDs, string(cdID))
		}
	}
	return cdIDs, nil
}

func (f *FreeClassroomBiz) GetFreeClassRoomFromCache(ctx context.Context, year, semester string, week, campus, day int, section []int, wherePrefix string) (map[int][]string, error) {
	// 筛选数据
	var freeClassroomMp = make(map[int][]string, len(section))

	var cacheMissCnt int

	for _, sec := range section {
		key := fmt.Sprintf("%s:%s:%s:%d:%d:%d:%d", FreeClassRoomCacheKeyPrefix, year, semester, week, campus, day, sec)
		strData, err := f.cache.Get(ctx, key)
		if err != nil {
			// 缓存未命中
			cacheMissCnt++
			clog.LogPrinter.Warnf("free classroom cache miss for key %s: %v", key, err)
			continue
		}

		var freeClassrooms []string
		err = json.Unmarshal([]byte(strData), &freeClassrooms)
		if err != nil {
			// 解析失败
			clog.LogPrinter.Errorf("failed to unmarshal free classroom data from cache for key %s [val=%v]: %v", key, strData, err)
			continue
		}
		for _, freeClassroom := range freeClassrooms {
			// 如果前缀匹配则加入结果
			if strings.HasPrefix(freeClassroom, wherePrefix) {
				freeClassroomMp[sec] = append(freeClassroomMp[sec], freeClassroom)
			}
		}
	}

	// 记录下缓存有多少没有命中
	clog.LogPrinter.Infof("cache miss cnt is %d,total is %d, (year=%v,semester=%v,week=%v,campus=%v,day=%v,sections=%v,wherePrefix=%v)",
		cacheMissCnt, len(section),
		year, semester, week, campus, day, section, wherePrefix)

	// 记录结果
	clog.LogPrinter.Infof("free classroom map(year=%v,semester=%v,week=%v,campus=%v,day=%v,sections=%v,wherePrefix=%v): %v", year, semester, week,
		campus, day, section, wherePrefix, freeClassroomMp)

	// 全部未命中则返回错误
	if len(section) > 0 && cacheMissCnt == len(section) {
		return freeClassroomMp, fmt.Errorf("cache miss cnt is %d, all miss", cacheMissCnt)
	}
	return freeClassroomMp, nil
}

// LoadOneWeekFreeClassRoom 加载缓存当前周所有的空教室
func (f *FreeClassroomBiz) LoadOneWeekFreeClassRoom(ctx context.Context, stuID, year, semester string, week int) {

	// 加分布式锁防止重复执行，两倍执行时长的锁保险
	mu := f.lockBuilder.BuildWithExpire("ccnubox_freeClassroom_lock", 3*time.Hour)
	err := mu.Lock()
	if err != nil {
		return
	}
	defer mu.Unlock()

	var cookie string

	campus, day, section := 2, 7, 12

	for c := 1; c <= campus; c++ {
		for d := 1; d <= day; d++ {
			// 定期换一下cookie防过期
			getCookieCtx, getCookieCancel := context.WithTimeout(ctx, TimeForWaitForCookie)
			cookie2, err := f.cookieCli.GetCookie(getCookieCtx, stuID)
			if err != nil {
				clog.LogPrinter.Errorf("failed to get cookie for stuId=%v : %v", stuID, err)
				// 取消
				getCookieCancel()
				continue
			} else {
				cookie = cookie2
			}

			for s := 1; s <= section; s++ {
				classrooms, err := f.sendReqFindFreeClassRoom(c, year, semester, "", week, d, s, cookie)
				if err != nil {
					clog.LogPrinter.Errorf("failed to send request for free_classroom: %v", err)
					continue
				}

				data, err := json.Marshal(classrooms)
				if err != nil {
					clog.LogPrinter.Errorf("failed to marshal value for free_classroom: %v", err)
					continue
				}
				value := string(data)
				key := fmt.Sprintf("%s:%s:%s:%d:%d:%d:%d", FreeClassRoomCacheKeyPrefix, year, semester, week, c, d, s)

				// 设置缓存
				func() {
					cacheTimeoutCtx, cacheTimeoutCancelFunc := context.WithTimeout(ctx, TimeForCache)
					defer cacheTimeoutCancelFunc()

					// 两倍过期时间，防止缓存未命中
					err = f.cache.Set(cacheTimeoutCtx, key, value, 2*Expire)
					if err != nil {
						clog.LogPrinter.Errorf("set free_class_room cache failed(key:%v,val:%v): %v", key, value, err)
						return
					}
				}()
			}

			getCookieCancel()
			time.Sleep(TimeForNext)
		}
	}
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
