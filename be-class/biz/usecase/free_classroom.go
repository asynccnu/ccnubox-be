package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-class/biz/model"
	"github.com/asynccnu/ccnubox-be/be-class/repository/lock"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
	"github.com/asynccnu/ccnubox-be/common/pkg/httpx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

const (
	FreeClassRoomCacheKeyPrefix = "ccnubox_freeclassroom:v2"
	freeClassroomReadyKeyPrefix = "ccnubox_freeclassroom_ready:v1"
	freeClassroomQueryURL       = "https://bkzhjw.ccnu.edu.cn/jsxsd/kbxx/jsjy_query2"
	freeClassroomReferer        = "https://bkzhjw.ccnu.edu.cn/jsxsd/kbxx/jsjy_query"
	freeClassroomTimeModelID    = "16FD8C2BE55E15F9E0630100007FF6B5"
	TimeForCache                = 5 * time.Second
	TimeForCacheRead            = 800 * time.Millisecond
	TimeForLocalQuery           = 2 * time.Second
	TimeForCrawlFallback        = 6 * time.Second
	Expire                      = 7 * 24 * time.Hour
)

type FreeClassRoomData interface {
	AddClassroomOccupancy(ctx context.Context, year, semester string, cwtPairs ...model.CTWPair) error
	ReplaceCrawledClassroomOccupancy(ctx context.Context, year, semester string, week int, cwtPairs ...model.CTWPair) error
	HasCrawledClassroomOccupancy(ctx context.Context, year, semester string, week int) (bool, error)
	ClearClassroomOccupancy(ctx context.Context, year, semester string) error
	GetAllClassroom(ctx context.Context, wherePrefix string) ([]string, error)
	RefreshClassroomOccupancy(ctx context.Context) error
	QueryAvailableClassrooms(ctx context.Context, year, semester string, week, day, section int, wherePrefix string, allWheres []string) (map[string]bool, error)
	QueryAvailableClassroomsFromCrawler(ctx context.Context, year, semester string, week, day, section int, wherePrefix string, allWheres []string) (map[string]bool, error)
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
	p                 proxy.Client
	logger            logger.Logger
	crawlerRepairs    sync.Map
}

func NewFreeClassroomBiz(classData ClassData, data FreeClassRoomData, cookieCli CookieClient, lockBuilder lock.Builder, cache Cache, p proxy.Client, l logger.Logger) *FreeClassroomBiz {
	fcb := &FreeClassroomBiz{
		classData:         classData,
		freeClassRoomData: data,
		cookieCli:         cookieCli,
		lockBuilder:       lockBuilder,
		cache:             cache,
		p:                 p,
		logger:            l,
	}
	return fcb
}

func (f *FreeClassroomBiz) ClearClassroomOccupancyFromES(ctx context.Context, year, semester string) error {
	return f.freeClassRoomData.ClearClassroomOccupancy(ctx, year, semester)
}

// SaveFreeClassRoomFromLocal 保存空教室信息从本地ES
func (f *FreeClassroomBiz) SaveFreeClassRoomFromLocal(ctx context.Context, year, semester string) error {
	const pageSize = 500 // 每批获取500条
	page := 1
	var tasks []string
	var savedAny bool
	readyKey := classroomOccupancyReadyKey(year, semester)

	if err := f.cache.Del(ctx, readyKey); err != nil {
		return fmt.Errorf("failed to clear classroom occupancy readiness for year=%s semester=%s: %w", year, semester, err)
	}

	defer func() {
		_ = f.cache.Del(ctx, tasks...)
	}()

	for {
		classes, total, err := f.classData.GetBatchClassInfos(ctx, year, semester, page, pageSize)
		if err != nil {
			return fmt.Errorf("failed to get batch classlist infos (year=%s semester=%s page=%d): %w", year, semester, page, err)
		}
		if len(classes) == 0 {
			if !savedAny {
				return fmt.Errorf("no class data available in local es for year=%s semester=%s", year, semester)
			}
			break
		}

		// 加锁
		lockKey := fmt.Sprintf("save_free_classroom_%v_%v_%v", year, semester, page)
		locker := f.lockBuilder.Build(lockKey)

		lockErr := locker.Lock()
		if lockErr != nil {
			return fmt.Errorf("failed to acquire classroom occupancy sync lock %s: %w", lockKey, lockErr)
		}

		f.logger.Infof("Lock %v success", lockKey)

		taskName := "task:" + lockKey
		tasks = append(tasks, taskName)

		status, err := f.cache.Get(ctx, taskName)
		if err == nil && status == Finished {
			savedAny = true
			f.logger.Infof("task %v is finished", taskName)

			if err := unlockFreeClassroomSync(locker, lockKey); err != nil {
				return err
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
			if tmp := strings.TrimSpace(class.Where); len(tmp) == 0 {
				continue
			}

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
				f.logger.Errorf("failed to set cache %v: %v", taskName, err1)
			}
			syncErr := fmt.Errorf("save classroom occupancy page %d: %w", page, err)
			if unlockErr := unlockFreeClassroomSync(locker, lockKey); unlockErr != nil {
				return errors.Join(syncErr, unlockErr)
			}
			return syncErr
		}
		if len(cwtPairs) > 0 {
			savedAny = true
		}

		// 设置task任务状态为finished
		err = f.cache.Set(ctx, taskName, Finished, 10*time.Minute)
		if err != nil {
			f.logger.Errorf("failed to set cache %v: %v", taskName, err)
		}

		// 解锁
		if err := unlockFreeClassroomSync(locker, lockKey); err != nil {
			return err
		}

		// 判断是否已经获取完所有数据
		if page*pageSize >= total {
			break
		}
		page++
	}
	if !savedAny {
		return fmt.Errorf("no valid classroom occupancy generated for year=%s semester=%s", year, semester)
	}
	if err := f.freeClassRoomData.RefreshClassroomOccupancy(ctx); err != nil {
		return fmt.Errorf("failed to refresh classroom occupancy index: %w", err)
	}
	if err := f.cache.Set(ctx, readyKey, Finished, 2*Expire); err != nil {
		return fmt.Errorf("failed to mark classroom occupancy ready for year=%s semester=%s: %w", year, semester, err)
	}
	return nil
}

func classroomOccupancyReadyKey(year, semester string) string {
	return fmt.Sprintf("%s:%s:%s", freeClassroomReadyKeyPrefix, year, semester)
}

func crawledClassroomOccupancyReadyKey(year, semester string, week int) string {
	return fmt.Sprintf("%s:crawler:%s:%s:%d", freeClassroomReadyKeyPrefix, year, semester, week)
}

func unlockFreeClassroomSync(locker lock.Locker, lockKey string) error {
	ok, err := locker.Unlock()
	if err != nil {
		return fmt.Errorf("failed to unlock classroom occupancy sync lock %s: %w", lockKey, err)
	}
	if !ok {
		return fmt.Errorf("failed to unlock classroom occupancy sync lock %s: lock was not released", lockKey)
	}
	return nil
}

func (f *FreeClassroomBiz) SaveFreeClassRoomInfo(ctx context.Context, year, semester string, cwtPairs []model.CTWPair) error {
	if len(cwtPairs) == 0 {
		f.logger.Warnf("no classroom occupancy data to save")
		return nil
	}

	//添加新数据
	err := f.freeClassRoomData.AddClassroomOccupancy(ctx, year, semester, cwtPairs...)
	if err != nil {
		f.logger.Errorf("failed to add classroom occupancy data to es: %v", err)
		return err
	}
	f.logger.Infof("add %d classroom occupancy data to es", len(cwtPairs))
	return nil
}

func (f *FreeClassroomBiz) SearchAvailableClassroom(ctx context.Context, year, semester, stuID string, week, day int, sections []int, wherePrefix string) ([]model.AvailableClassroomStat, error) {
	if err := validateFreeClassroomQuery(year, semester, week, day, sections, wherePrefix); err != nil {
		return nil, err
	}

	year = strings.Split(year, "-")[0]

	// 先获取全部教室
	classroomSet, err := f.getAllClassrooms(ctx, wherePrefix)
	if err != nil {
		return nil, err
	}
	if len(classroomSet) == 0 {
		return []model.AvailableClassroomStat{}, nil
	}

	type crawlResult struct {
		stats map[string][]bool
		err   error
	}
	crawlResultCh := make(chan crawlResult, 1)

	go func() {
		crawlCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 40*time.Second)
		defer cancel()

		freeClassroomMp, err := f.getFreeClassrooms(crawlCtx, year, semester, stuID, week, day, sections, wherePrefix)
		if err != nil {
			crawlResultCh <- crawlResult{err: err}
			return
		}

		crawClassroomStats := make(map[string][]bool, len(classroomSet))
		for _, classroom := range classroomSet {
			crawClassroomStats[classroom] = make([]bool, len(sections))
		}
		secIdx := make(map[int]int, len(sections))
		for k, v := range sections {
			secIdx[v] = k
		}
		for sec, freeClassrooms := range freeClassroomMp {
			idx, ok := secIdx[sec]
			if !ok {
				continue
			}
			for _, freeClassroom := range freeClassrooms {
				if stats, ok := crawClassroomStats[freeClassroom]; ok {
					stats[idx] = true
				}
			}
		}
		crawlResultCh <- crawlResult{stats: crawClassroomStats}
	}()

	localQueryCtx, cancel := context.WithTimeout(ctx, TimeForLocalQuery)
	localFreeClassrooms, localErr := f.queryAvailableClassroomFromLocal(localQueryCtx, year, semester, week, day, sections, wherePrefix, classroomSet)
	cancel()

	if localErr == nil {
		waitForCrawl := 1500 * time.Millisecond
		if hasUniformAvailability(localFreeClassrooms) {
			waitForCrawl = TimeForCrawlFallback
		}
		select {
		case crawResult := <-crawlResultCh:
			if crawResult.err == nil {
				return toSerializableClassroomStats(crawResult.stats), nil
			}
			f.logger.Warnf("realtime free classroom query failed, using local data: %v", crawResult.err)
		case <-time.After(waitForCrawl):
		case <-ctx.Done():
		}
		return toSerializableClassroomStats(localFreeClassrooms), nil
	}

	var realtimeErr error
	select {
	case crawResult := <-crawlResultCh:
		if crawResult.err == nil {
			return toSerializableClassroomStats(crawResult.stats), nil
		}
		realtimeErr = crawResult.err
	case <-time.After(TimeForCrawlFallback):
		realtimeErr = fmt.Errorf("realtime query timed out after %s", TimeForCrawlFallback)
	case <-ctx.Done():
		realtimeErr = ctx.Err()
	}

	err = fmt.Errorf("local free classroom query failed: %v; realtime query failed: %w", localErr, realtimeErr)
	f.logger.Errorf("no free classroom data source available: %v", err)
	return nil, err
}

func validateFreeClassroomQuery(year, semester string, week, day int, sections []int, wherePrefix string) error {
	if strings.TrimSpace(year) == "" {
		return fmt.Errorf("year is required")
	}
	if semester != "1" && semester != "2" && semester != "3" {
		return fmt.Errorf("invalid semester %q", semester)
	}
	if week < 1 || week > 30 {
		return fmt.Errorf("week must be between 1 and 30")
	}
	if day < 1 || day > 7 {
		return fmt.Errorf("day must be between 1 and 7")
	}
	if len(sections) == 0 {
		return fmt.Errorf("sections is required")
	}
	for _, section := range sections {
		if section < 1 || section > 12 {
			return fmt.Errorf("section must be between 1 and 12")
		}
	}
	if strings.TrimSpace(wherePrefix) == "" {
		return fmt.Errorf("wherePrefix is required")
	}
	return nil
}

func (f *FreeClassroomBiz) getAllClassrooms(ctx context.Context, wherePrefix string) ([]string, error) {
	key := fmt.Sprintf("all_classrooms:%s", wherePrefix)

	cacheCtx, cacheCancel := context.WithTimeout(ctx, TimeForCacheRead)
	res, err := f.cache.SMembers(cacheCtx, key)
	cacheCancel()
	if err == nil && len(res) > 0 {
		return res, nil
	}

	rooms, err := f.freeClassRoomData.GetAllClassroom(ctx, wherePrefix)
	if err != nil {
		return nil, err
	}

	_ = f.cache.SAdd(ctx, key, rooms)
	_ = f.cache.SExpire(ctx, key, 7*24*time.Hour)
	return rooms, nil
}

func toSerializableClassroomStats(classroomStats map[string][]bool) []model.AvailableClassroomStat {
	var res = make([]model.AvailableClassroomStat, 0, len(classroomStats))
	for classroom, stats := range classroomStats {
		res = append(res, model.AvailableClassroomStat{
			Classroom:     classroom,
			AvailableStat: stats,
		})
	}
	sort.Slice(res, func(i, j int) bool {
		return naturalLess(res[i].Classroom, res[j].Classroom)
	})
	return res
}

func hasUniformAvailability(classroomStats map[string][]bool) bool {
	if len(classroomStats) == 0 {
		return false
	}

	var (
		base bool
		seen bool
	)
	for _, stats := range classroomStats {
		for _, stat := range stats {
			if !seen {
				base = stat
				seen = true
				continue
			}
			if stat != base {
				return false
			}
		}
	}
	return seen
}

func naturalLess(a, b string) bool {
	ai, bi := 0, 0
	for ai < len(a) && bi < len(b) {
		ar, br := a[ai], b[bi]
		if isDigit(ar) && isDigit(br) {
			an, nextA := readNumber(a, ai)
			bn, nextB := readNumber(b, bi)
			if an != bn {
				return an < bn
			}
			if nextA-ai != nextB-bi {
				return nextA-ai < nextB-bi
			}
			ai, bi = nextA, nextB
			continue
		}

		ar = toLowerASCII(ar)
		br = toLowerASCII(br)
		if ar != br {
			return ar < br
		}
		ai++
		bi++
	}
	return len(a) < len(b)
}

func readNumber(s string, start int) (int64, int) {
	end := start
	for end < len(s) && isDigit(s[end]) {
		end++
	}
	n, err := strconv.ParseInt(s[start:end], 10, 64)
	if err != nil {
		return 0, end
	}
	return n, end
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func toLowerASCII(ch byte) byte {
	if ch >= 'A' && ch <= 'Z' {
		return ch + 'a' - 'A'
	}
	return ch
}

func (f *FreeClassroomBiz) queryAvailableClassroomFromLocal(ctx context.Context, year, semester string, week, day int, sections []int, wherePrefix string, allWheres []string) (map[string][]bool, error) {
	useCrawlerData := false
	crawlerReadyKey := crawledClassroomOccupancyReadyKey(year, semester, week)
	status, crawlerErr := f.cache.Get(ctx, crawlerReadyKey)
	if crawlerErr == nil && status == Finished {
		hasCrawlerData, err := f.freeClassRoomData.HasCrawledClassroomOccupancy(ctx, year, semester, week)
		if err == nil && hasCrawlerData {
			useCrawlerData = true
		} else {
			if err != nil {
				f.logger.Warnf("failed to verify crawler occupancy mirror, falling back to classlist data: %v", err)
			} else {
				f.logger.Warnf("crawler occupancy mirror is marked ready but contains no data; falling back to classlist data")
			}
			if err := f.cache.Del(ctx, crawlerReadyKey); err != nil {
				f.logger.Warnf("failed to clear stale crawler occupancy readiness: %v", err)
			}
		}
	}
	if !useCrawlerData {
		status, err := f.cache.Get(ctx, classroomOccupancyReadyKey(year, semester))
		if err != nil || status != Finished {
			return nil, fmt.Errorf("local classroom occupancy data is not ready for year=%s semester=%s week=%d", year, semester, week)
		}
	}

	var classroomStats = make(map[string][]bool)
	for i, section := range sections {
		var (
			availableClassrooms map[string]bool
			err                 error
		)
		if useCrawlerData {
			availableClassrooms, err = f.freeClassRoomData.QueryAvailableClassroomsFromCrawler(ctx, year, semester, week, day, section, wherePrefix, allWheres)
		} else {
			availableClassrooms, err = f.freeClassRoomData.QueryAvailableClassrooms(ctx, year, semester, week, day, section, wherePrefix, allWheres)
		}
		if i == 0 {
			if err != nil {
				f.logger.Errorf("failed to query available classrooms at the first section: %v", err)
				return nil, err
			}
			for classroom, stat := range availableClassrooms {
				classroomStats[classroom] = make([]bool, len(sections))
				classroomStats[classroom][i] = stat
			}
			continue
		}
		if err != nil {
			f.logger.Warnf("failed to query available classrooms: %v", err)
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
	var campus = 1
	if strings.HasPrefix(wherePrefix, "n") {
		campus = 2
	}

	// 先从缓存拿数据
	freeClassroomCache, err := f.GetFreeClassRoomFromCache(ctx, year, semester, week, campus, day, sections, wherePrefix)
	if err == nil {
		status, readyErr := f.cache.Get(ctx, crawledClassroomOccupancyReadyKey(year, semester, week))
		if readyErr != nil || status != Finished {
			f.startCrawledClassroomMirrorRepair(year, semester, week)
		}
		return freeClassroomCache, nil
	}

	// 到这里就是缓存全部没有命中
	// 预定就是先提前缓存，所以一般都是命中缓存
	// 当然如果缓存重启什么的，就会触发后面的流程

	cookie, err := f.cookieCli.GetCookie(ctx, stuID)
	if err != nil {
		return nil, err
	}

	schedule, err := f.sendReqFindFreeClassRoom(ctx, year, semester, week, cookie)
	if err != nil {
		return nil, err
	}
	freeClassroomMp := selectFreeClassrooms(schedule, campus, day, sections, wherePrefix)

	// 新接口一次返回整周数据，复用本次响应异步预热全部缓存。
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := f.cacheFreeClassroomSchedule(cacheCtx, year, semester, week, schedule); err != nil {
			f.logger.Errorf("failed to persist crawled free classroom schedule: %v", err)
		}
	}()

	return freeClassroomMp, nil
}

type freeClassroomSchedule map[int]map[int][]string

type freeClassroomPeriod struct {
	JC1 string `json:"jc1"`
	JC2 string `json:"jc2"`
	JC3 string `json:"jc3"`
}

func newFreeClassroomSchedule() freeClassroomSchedule {
	schedule := make(freeClassroomSchedule, 7)
	for day := 1; day <= 7; day++ {
		schedule[day] = make(map[int][]string, 12)
		for section := 1; section <= 12; section++ {
			schedule[day][section] = make([]string, 0)
		}
	}
	return schedule
}

func academicTermCode(preYear, semester string) (string, error) {
	if semester != "1" && semester != "2" && semester != "3" {
		return "", fmt.Errorf("invalid semester %q", semester)
	}

	startYearText := strings.TrimSpace(strings.Split(preYear, "-")[0])
	startYear, err := strconv.Atoi(startYearText)
	if err != nil || startYear < 2000 || startYear > 3000 {
		return "", fmt.Errorf("invalid academic year %q", preYear)
	}
	return fmt.Sprintf("%d-%d-%s", startYear, startYear+1, semester), nil
}

func buildFreeClassroomQuery(preYear, semester string, week int) (url.Values, error) {
	if week < 1 || week > 30 {
		return nil, fmt.Errorf("invalid week %d", week)
	}
	termCode, err := academicTermCode(preYear, semester)
	if err != nil {
		return nil, err
	}

	return url.Values{
		"xnxqh":      {termCode},
		"xqbh":       {""},
		"jxqbh":      {""},
		"jxlbh":      {""},
		"jsbh":       {""},
		"jslx":       {""},
		"bjfh":       {"="},
		"rnrs":       {""},
		"yx":         {""},
		"kbjcmsid":   {freeClassroomTimeModelID},
		"selectZc":   {strconv.Itoa(week)},
		"startdate":  {""},
		"enddate":    {""},
		"selectXq":   {"1,2,3,4,5,6,7"},
		"selectJc":   {"0102,0304,0506,0708,0910,1112"},
		"syjs0601id": {""},
		"typewhere":  {"jszq"},
	}, nil
}

func (f *FreeClassroomBiz) sendReqFindFreeClassRoom(ctx context.Context, preYear, semester string, week int, cookie string) (freeClassroomSchedule, error) {
	form, err := buildFreeClassroomQuery(preYear, semester, week)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, freeClassroomQueryURL, strings.NewReader(form.Encode()))
	if err != nil {
		f.logger.Errorf("failed to create request: %v", err)
		return nil, err
	}
	req.Header = http.Header{
		"Accept":           []string{"application/json, text/javascript, */*; q=0.01"},
		"Accept-Language":  []string{"zh-CN,zh;q=0.9"},
		"Content-Type":     []string{"application/x-www-form-urlencoded; charset=UTF-8"},
		"Cookie":           []string{cookie},
		"Origin":           []string{"https://bkzhjw.ccnu.edu.cn"},
		"Referer":          []string{freeClassroomReferer},
		"User-Agent":       []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36"},
		"X-Requested-With": []string{"XMLHttpRequest"},
	}
	resp, err := f.p.NewProxyClient(
		proxy.WithProxyTransport(),
		proxy.WithRedirectPolicy(proxy.RedirectPolicyDefault),
		proxy.WithTimeout(30*time.Second),
	).Do(req)
	if err != nil {
		f.logger.Errorf("failed to send request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := httpx.ReadLimited(resp.Body, 1024)
		bodySummary := strings.Join(strings.Fields(string(body)), " ")
		return nil, fmt.Errorf("free classroom upstream returned HTTP %d: %.300s", resp.StatusCode, bodySummary)
	}
	// 读取 Body 到字节数组
	bodyBytes, err := httpx.ReadResponse(resp)

	if err != nil {
		f.logger.Warnf("failed to read response body: %v", err)
		return nil, fmt.Errorf("failed to read free classroom response: %w", err)
	}

	schedule, err := extractFreeClassroomsFromQuery2(bodyBytes)
	if err != nil {
		f.logger.Errorf("failed to parse response body: %v", err)
		return nil, fmt.Errorf("failed to parse free classroom response: %w", err)
	}

	return schedule, nil
}

func extractFreeClassroomsFromQuery2(rawJSON []byte) (freeClassroomSchedule, error) {
	var response []json.RawMessage
	if err := json.Unmarshal(rawJSON, &response); err != nil {
		return nil, fmt.Errorf("upstream response is not JSON: %w", err)
	}
	if len(response) < 7 {
		return nil, fmt.Errorf("unexpected upstream response length %d", len(response))
	}

	var periods []freeClassroomPeriod
	if err := json.Unmarshal(response[0], &periods); err != nil {
		return nil, fmt.Errorf("invalid period metadata: %w", err)
	}
	if len(periods) == 0 {
		return nil, fmt.Errorf("upstream returned no period metadata")
	}
	var rows [][]json.RawMessage
	if err := json.Unmarshal(response[4], &rows); err != nil {
		return nil, fmt.Errorf("invalid classroom rows: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("upstream returned no classroom rows")
	}
	var dayCount int
	if err := json.Unmarshal(response[5], &dayCount); err != nil {
		return nil, fmt.Errorf("invalid day count: %w", err)
	}
	if dayCount != 7 {
		return nil, fmt.Errorf("invalid day count %d", dayCount)
	}
	var arguments []json.RawMessage
	if err := json.Unmarshal(response[6], &arguments); err != nil {
		return nil, fmt.Errorf("invalid classroom cell metadata: %w", err)
	}

	cellCount := dayCount * len(periods)
	if len(arguments) != cellCount {
		return nil, fmt.Errorf("unexpected classroom grid size: got %d cells, want %d", len(arguments), cellCount)
	}

	schedule := newFreeClassroomSchedule()
	validRows := 0
	for _, row := range rows {
		// 行结构：教室名、网格单元、教室 ID、容量、类型。
		if len(row) < cellCount+4 {
			continue
		}
		var classroom string
		if err := json.Unmarshal(row[0], &classroom); err != nil {
			continue
		}
		classroom = strings.TrimSpace(classroom)
		if classroom == "" {
			continue
		}
		validRows++

		for dayOffset := 0; dayOffset < dayCount; dayOffset++ {
			for periodIndex, period := range periods {
				cell := row[1+dayOffset*len(periods)+periodIndex]
				if !isFreeClassroomCell(cell) {
					continue
				}
				for _, sectionText := range []string{period.JC1, period.JC2, period.JC3} {
					section, err := strconv.Atoi(sectionText)
					if err == nil && section >= 1 && section <= 12 {
						day := dayOffset + 1
						schedule[day][section] = append(schedule[day][section], classroom)
					}
				}
			}
		}
	}
	if validRows == 0 {
		return nil, fmt.Errorf("upstream returned no valid classroom rows")
	}
	return schedule, nil
}

func isFreeClassroomCell(cell json.RawMessage) bool {
	value := strings.TrimSpace(string(cell))
	if value == "" || value == "null" {
		return true
	}

	var text string
	if err := json.Unmarshal(cell, &text); err != nil {
		return false
	}
	return strings.TrimSpace(html.UnescapeString(text)) == ""
}

func selectFreeClassrooms(schedule freeClassroomSchedule, campus, day int, sections []int, wherePrefix string) map[int][]string {
	result := make(map[int][]string, len(sections))
	for _, section := range sections {
		result[section] = make([]string, 0)
		for _, classroom := range schedule[day][section] {
			isSouthLake := strings.HasPrefix(strings.ToLower(classroom), "n")
			if (campus == 2) != isSouthLake {
				continue
			}
			if wherePrefix != "" && !strings.HasPrefix(classroom, wherePrefix) {
				continue
			}
			result[section] = append(result[section], classroom)
		}
	}
	return result
}

func (f *FreeClassroomBiz) GetFreeClassRoomFromCache(ctx context.Context, year, semester string, week, campus, day int, section []int, wherePrefix string) (map[int][]string, error) {
	// 筛选数据
	var freeClassroomMp = make(map[int][]string, len(section))

	var cacheMissCnt int

	for _, sec := range section {
		key := fmt.Sprintf("%s:%s:%s:%d:%d:%d:%d", FreeClassRoomCacheKeyPrefix, year, semester, week, campus, day, sec)
		cacheCtx, cacheCancel := context.WithTimeout(ctx, TimeForCacheRead)
		strData, err := f.cache.Get(cacheCtx, key)
		cacheCancel()
		if err != nil {
			// 缓存未命中
			cacheMissCnt++
			f.logger.Warnf("free classroom cache miss for key %s: %v", key, err)
			continue
		}
		if strings.TrimSpace(strData) == "null" {
			cacheMissCnt++
			f.logger.Warnf("invalid null free classroom cache for key %s", key)
			_ = f.cache.Del(ctx, key)
			continue
		}

		var freeClassrooms []string
		err = json.Unmarshal([]byte(strData), &freeClassrooms)
		if err != nil {
			// 解析失败
			f.logger.Errorf("failed to unmarshal free classroom data from cache for key %s [val=%v]: %v", key, strData, err)
			cacheMissCnt++
			_ = f.cache.Del(ctx, key)
			continue
		}
		if freeClassrooms == nil {
			cacheMissCnt++
			f.logger.Warnf("invalid nil free classroom cache for key %s", key)
			_ = f.cache.Del(ctx, key)
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
	f.logger.Infof("cache miss cnt is %d,total is %d, (year=%v,semester=%v,week=%v,campus=%v,day=%v,sections=%v,wherePrefix=%v)",
		cacheMissCnt, len(section),
		year, semester, week, campus, day, section, wherePrefix)

	// 记录结果
	f.logger.Infof("free classroom map(year=%v,semester=%v,week=%v,campus=%v,day=%v,sections=%v,wherePrefix=%v): %v", year, semester, week,
		campus, day, section, wherePrefix, freeClassroomMp)

	// 任意节次未命中都返回错误，避免上层把缺失节次误判成全不空闲
	if cacheMissCnt > 0 {
		return freeClassroomMp, fmt.Errorf("cache miss cnt is %d,total is %d", cacheMissCnt, len(section))
	}
	return freeClassroomMp, nil
}

// LoadOneWeekFreeClassRoom 加载缓存当前周所有的空教室
func (f *FreeClassroomBiz) LoadOneWeekFreeClassRoom(ctx context.Context, stuID, year, semester string, week int) {
	// 加分布式锁防止重复执行。
	mu := f.lockBuilder.BuildWithExpire("ccnubox_freeClassroom_lock", 3*time.Hour)
	err := mu.Lock()
	if err != nil {
		return
	}
	defer mu.Unlock()

	cookie, err := f.cookieCli.GetCookie(ctx, stuID)
	if err != nil {
		f.logger.Errorf("failed to get cookie for stuId=%v: %v", stuID, err)
		return
	}
	schedule, err := f.sendReqFindFreeClassRoom(ctx, year, semester, week, cookie)
	if err != nil {
		f.logger.Errorf("failed to query weekly free classrooms: %v", err)
		return
	}
	if err := f.cacheFreeClassroomSchedule(ctx, year, semester, week, schedule); err != nil {
		f.logger.Errorf("failed to persist weekly free classrooms: %v", err)
	}
}

func (f *FreeClassroomBiz) cacheFreeClassroomSchedule(ctx context.Context, year, semester string, week int, schedule freeClassroomSchedule) error {
	var cacheErrs []error
	for campus := 1; campus <= 2; campus++ {
		for day := 1; day <= 7; day++ {
			for section := 1; section <= 12; section++ {
				classrooms := selectFreeClassrooms(schedule, campus, day, []int{section}, "")[section]
				data, err := json.Marshal(classrooms)
				if err != nil {
					f.logger.Errorf("failed to marshal free classrooms: %v", err)
					continue
				}

				key := fmt.Sprintf("%s:%s:%s:%d:%d:%d:%d", FreeClassRoomCacheKeyPrefix, year, semester, week, campus, day, section)
				cacheCtx, cancel := context.WithTimeout(ctx, TimeForCache)
				err = f.cache.Set(cacheCtx, key, string(data), 2*Expire)
				cancel()
				if err != nil {
					f.logger.Errorf("set free classroom cache failed (key=%s): %v", key, err)
					cacheErrs = append(cacheErrs, fmt.Errorf("set free classroom cache %s: %w", key, err))
				}
			}
		}
	}
	if len(cacheErrs) > 0 {
		return errors.Join(cacheErrs...)
	}

	return f.persistCrawledClassroomMirror(ctx, year, semester, week, schedule)
}

func (f *FreeClassroomBiz) persistCrawledClassroomMirror(ctx context.Context, year, semester string, week int, schedule freeClassroomSchedule) error {
	allRooms, err := f.freeClassRoomData.GetAllClassroom(ctx, "")
	if err != nil {
		return fmt.Errorf("load classroom catalog for crawler mirror: %w", err)
	}
	if len(allRooms) == 0 {
		return fmt.Errorf("classroom catalog for crawler mirror is empty")
	}
	cwtPairs := buildCrawledClassroomOccupancy(schedule, week, allRooms)
	readyKey := crawledClassroomOccupancyReadyKey(year, semester, week)
	if err := f.cache.Del(ctx, readyKey); err != nil {
		return fmt.Errorf("clear crawler occupancy readiness before replacement: %w", err)
	}
	if err := f.freeClassRoomData.ReplaceCrawledClassroomOccupancy(ctx, year, semester, week, cwtPairs...); err != nil {
		return fmt.Errorf("replace crawled classroom occupancy: %w", err)
	}
	if err := f.cache.Set(ctx, readyKey, Finished, 2*Expire); err != nil {
		return fmt.Errorf("mark crawled classroom occupancy ready: %w", err)
	}
	return nil
}

func (f *FreeClassroomBiz) startCrawledClassroomMirrorRepair(year, semester string, week int) {
	key := fmt.Sprintf("%s:%s:%d", year, semester, week)
	startOnce(&f.crawlerRepairs, key, func() {
		f.repairCrawledClassroomMirrorFromCache(year, semester, week)
	}, func(recovered any) {
		f.logger.Errorf("crawler mirror repair panicked (year=%s semester=%s week=%d): %v\n%s", year, semester, week, recovered, debug.Stack())
	})
}

func startOnce(inFlight *sync.Map, key string, fn func(), onPanic func(any)) bool {
	if _, loaded := inFlight.LoadOrStore(key, struct{}{}); loaded {
		return false
	}
	go func() {
		defer inFlight.Delete(key)
		defer func() {
			if recovered := recover(); recovered != nil && onPanic != nil {
				onPanic(recovered)
			}
		}()
		fn()
	}()
	return true
}

func (f *FreeClassroomBiz) repairCrawledClassroomMirrorFromCache(year, semester string, week int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	lockKey := fmt.Sprintf("ccnubox_freeClassroom_mirror_lock:%s:%s:%d", year, semester, week)
	locker := f.lockBuilder.BuildWithExpire(lockKey, 2*time.Minute)
	if err := locker.Lock(); err != nil {
		return
	}
	defer func() {
		if ok, err := locker.Unlock(); err != nil || !ok {
			f.logger.Errorf("failed to unlock crawler mirror repair %s: %v", lockKey, err)
		}
	}()

	status, err := f.cache.Get(ctx, crawledClassroomOccupancyReadyKey(year, semester, week))
	if err == nil && status == Finished {
		return
	}
	schedule, err := f.loadFreeClassroomScheduleFromCache(ctx, year, semester, week)
	if err != nil {
		f.logger.Errorf("failed to load cached free classroom schedule for ES repair: %v", err)
		return
	}
	if err := f.persistCrawledClassroomMirror(ctx, year, semester, week, schedule); err != nil {
		f.logger.Errorf("failed to repair crawled classroom ES mirror: %v", err)
	}
}

func (f *FreeClassroomBiz) loadFreeClassroomScheduleFromCache(ctx context.Context, year, semester string, week int) (freeClassroomSchedule, error) {
	schedule := newFreeClassroomSchedule()
	for campus := 1; campus <= 2; campus++ {
		for day := 1; day <= 7; day++ {
			for section := 1; section <= 12; section++ {
				key := fmt.Sprintf("%s:%s:%s:%d:%d:%d:%d", FreeClassRoomCacheKeyPrefix, year, semester, week, campus, day, section)
				raw, err := f.cache.Get(ctx, key)
				if err != nil {
					return nil, fmt.Errorf("get cache %s: %w", key, err)
				}
				var rooms []string
				if err := json.Unmarshal([]byte(raw), &rooms); err != nil || rooms == nil {
					return nil, fmt.Errorf("invalid cached classroom list for %s", key)
				}
				schedule[day][section] = append(schedule[day][section], rooms...)
			}
		}
	}
	return schedule, nil
}

func buildCrawledClassroomOccupancy(schedule freeClassroomSchedule, week int, allRooms []string) []model.CTWPair {
	roomSet := make(map[string]struct{}, len(allRooms))
	for _, room := range allRooms {
		room = strings.TrimSpace(room)
		if room != "" {
			roomSet[room] = struct{}{}
		}
	}

	occupiedSections := make(map[string]map[int][]int, len(roomSet))
	for day := 1; day <= 7; day++ {
		for section := 1; section <= 12; section++ {
			freeRooms := make(map[string]struct{}, len(schedule[day][section]))
			for _, room := range schedule[day][section] {
				freeRooms[room] = struct{}{}
			}
			for room := range roomSet {
				if _, free := freeRooms[room]; free {
					continue
				}
				if occupiedSections[room] == nil {
					occupiedSections[room] = make(map[int][]int)
				}
				occupiedSections[room][day] = append(occupiedSections[room][day], section)
			}
		}
	}

	pairs := make([]model.CTWPair, 0, len(occupiedSections)*7)
	for room, days := range occupiedSections {
		for day, sections := range days {
			pairs = append(pairs, model.CTWPair{
				CT:    model.CTime{Day: day, Sections: sections, Weeks: []int{week}},
				Where: room,
			})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Where == pairs[j].Where {
			return pairs[i].CT.Day < pairs[j].CT.Day
		}
		return naturalLess(pairs[i].Where, pairs[j].Where)
	})
	return pairs
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
