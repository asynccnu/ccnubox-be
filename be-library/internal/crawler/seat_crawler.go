package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/client"
	"github.com/asynccnu/ccnubox-be/be-library/internal/errcode"
	"github.com/asynccnu/ccnubox-be/be-library/internal/model"
	"github.com/asynccnu/ccnubox-be/be-library/pkg/tool"
	"github.com/asynccnu/ccnubox-be/common/pkg/crypto"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tidwall/gjson"
)

// 定义全局URL常量
const (
	BaseDomain = "https://kjyy.ccnu.edu.cn"
)

const (
	TODAY_RECORD_TYPE   = "today"
	HISTORY_RECORD_TYPE = "history"
)

// API端点路径
var (
	DeviceAPIPath  = "/jsq/static/frontApi/res/freeSeatIdsDuration"
	ReserveAPIPath = "/jsq/static/frontApi/make/freeBook"
	CancelAPIPath  = "/jsq/static/frontApi/make/cancel"
)

// Crawler 主爬虫结构体
type Crawler struct {
	log      *log.Helper
	ccnu     biz.CCNUServiceProxy
	waitTime time.Duration
	client   client.Client
	secret   string
}

// NewLibraryCrawler 创建新的图书馆爬虫
func NewLibraryCrawler(logger log.Logger, ccnu biz.CCNUServiceProxy, waitTime time.Duration, client client.Client, secret string) biz.LibraryCrawler {
	return &Crawler{
		log:      log.NewHelper(logger),
		ccnu:     ccnu,
		waitTime: waitTime,
		client:   client,
		secret:   secret,
	}
}

func (c *Crawler) getSeatToken(ctx context.Context, stuID string) (string, error) {
	return tool.Retry(func() (string, error) {
		timeoutCtx, cancel := context.WithTimeout(ctx, c.waitTime)
		defer cancel()

		token, err := c.ccnu.GetLibrarySeatToken(timeoutCtx, stuID)
		if err != nil {
			return "", err
		}

		return token, nil
	})
}

// buildURL 构建带参数的URL
func buildURL(path string, params url.Values) (string, error) {
	baseURL := BaseDomain + path

	// 创建URL对象
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// 添加传入的query到URL
	u.RawQuery = params.Encode()

	return u.String(), nil
}

// doSeatRequestWithToken 通用HTTP请求函数
func (c *Crawler) doSeatRequestWithToken(ctx context.Context, client client.Client, method, url, token string, body io.Reader) (*http.Response, error) {
	return tool.Retry(func() (*http.Response, error) {
		id, sign, ts := crypto.BuildSignWithSecret("POST", c.secret)
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Token", token)
		req.Header.Set("LoginType", "PC")
		req.Header.Set("X-Hmac-Request-Key", sign)
		req.Header.Set("X-Request-Date", fmt.Sprintf("%d", ts))
		req.Header.Set("X-Request-Id", id)
		resp, err := client.DoWithContext(ctx, req)
		if err != nil {
			return nil, errcode.ErrCrawler
		}

		return resp, nil
	})
}

// GetSeatInfos 获取不定个给定的房间的座位信息
func (c *Crawler) GetSeatInfos(ctx context.Context, stuID string, roomIDs []string) (map[string][]*biz.Seat, error) {
	var wg sync.WaitGroup
	results := make(map[string][]*biz.Seat)
	mutex := &sync.Mutex{}

	for _, roomID := range roomIDs {
		wg.Add(1)
		go func(roomID string) {
			defer wg.Done()
			seats, err := c.getSeatInfos(ctx, roomID, stuID)
			if err != nil {
				c.log.Errorf("获取房间 %s 座位失败: %v", roomID, err)
				mutex.Lock()
				results[roomID] = nil
				mutex.Unlock()
				return // todo错误处理
			}
			mutex.Lock()
			results[roomID] = seats
			mutex.Unlock()
		}(roomID)
	}

	wg.Wait()
	return results, nil
}

// getSeatInfos 获取指定房间的座位信息
func (c *Crawler) getSeatInfos(ctx context.Context, roomid string, stuID string) ([]*biz.Seat, error) {
	var date string
	var reqData getSeatInfoReq
	loc, _ := tool.GetLocation()
	//如果是22:00之后就只能查询第二天的座位信息(此时学校系统这一天的座位已经查不到了）
	if time.Now().In(loc).Hour() >= 22 {
		date = time.Now().In(loc).AddDate(0, 0, 1).Format("2006-01-02")
		reqData = getSeatInfoReq{
			BeginMinute: -1,
			EndMinute:   0,
			MinMinute:   0,
		}
	} else {
		date = time.Now().In(loc).Format("2006-01-02")
		beginMinute := tool.ParseTimeToMinute(time.Now().In(loc))
		reqData = getSeatInfoReq{
			BeginMinute: beginMinute,
			EndMinute:   0,
			MinMinute:   0,
		}
	}

	fullURL := fmt.Sprintf("%s/%s/%s", BaseDomain+DeviceAPIPath, roomid, date)
	token, err := c.getSeatToken(ctx, stuID)
	if err != nil {
		return nil, err
	}

	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	resp, err := c.doSeatRequestWithToken(ctx, c.client, "POST", fullURL, token, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 用 gjson 解析
	data := gjson.GetBytes(body, "data")
	if !data.Exists() {
		return nil, nil
	}

	var result []*biz.Seat

	// 遍历每个 seat
	data.ForEach(func(_, item gjson.Result) bool {
		it := item
		seat := &biz.Seat{
			ID:        it.Get("id").String(),
			Label:     it.Get("label").String(),
			Name:      it.Get("name").String(),
			Status:    it.Get("status").String(),
			AfterFree: it.Get("afterFree").Bool(),
		}
		result = append(result, seat)
		return true
	})

	return result, nil
}

func (c *Crawler) GetFreeList(ctx context.Context, seatID string, stuID string) ([]*biz.FreeTime, error) {
	loc, _ := tool.GetLocation()
	date := time.Now().In(loc).Format("2006-01-02")
	fullURL := fmt.Sprintf("https://kjyy.ccnu.edu.cn/jsq/static/frontApi/res/getTimeLine/%s/%s", seatID, date)
	token, err := c.getSeatToken(ctx, stuID)
	if err != nil {
		return nil, err
	}

	resp, err := c.doSeatRequestWithToken(ctx, c.client, "POST", fullURL, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	data := gjson.GetBytes(body, "data")
	if !data.Exists() {
		return nil, nil
	}

	freeListData := data.Get("freeList")
	if !freeListData.IsArray() {
		return nil, nil
	}

	var freeList []*biz.FreeTime
	freeListData.ForEach(func(_, item gjson.Result) bool {
		freeTimeLabel := item.Get("label")
		if !freeTimeLabel.Exists() {
			return true
		}
		res := strings.Split(freeTimeLabel.String(), "-")
		if len(res) != 2 {
			return true
		}
		freeTime := &biz.FreeTime{
			Start: res[0],
			End:   res[1],
		}
		freeList = append(freeList, freeTime)
		return true
	})

	return freeList, nil
}

// test
func (c *Crawler) GetLibrarySeatToken(stuid string) (string, error) {
	return c.ccnu.GetLibrarySeatToken(context.Background(), stuid)
}

// ReserveSeat 预约座位
func (c *Crawler) ReserveSeat(ctx context.Context, stuID string, devid, start, end string) (string, error) {
	params := url.Values{}
	params.Add("capToken", "capToken")
	loc, _ := tool.GetLocation()
	date := time.Now().In(loc).Format("2006-01-02")
	path := fmt.Sprintf("%s/%s/%s/%s/%s", ReserveAPIPath, devid, date, start, end)
	fullURL, err := buildURL(path, params)
	if err != nil {
		return "", errcode.ErrCrawler
	}
	token, err := c.getSeatToken(ctx, stuID)
	if err != nil {
		return "", err
	}
	resp, err := c.doSeatRequestWithToken(ctx, c.client, "POST", fullURL, token, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	data := gjson.GetBytes(body, "data")
	if !data.Exists() {
		return "", nil
	}

	return data.Get("message").String(), nil

}

// CancelReserve 取消预约
func (c *Crawler) CancelReserve(ctx context.Context, stuID string, id string) (string, error) {
	fullURL := fmt.Sprintf("%s/%s", BaseDomain+CancelAPIPath, id)

	token, err := c.getSeatToken(ctx, stuID)
	if err != nil {
		c.log.Errorf("Error getting client(stu_id:%v): %v", stuID, err)
		return "", err
	}
	resp, err := c.doSeatRequestWithToken(ctx, c.client, "POST", fullURL, token, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var res model.Response
	err = json.Unmarshal(body, &res)
	if err != nil {
		return "", err
	}
	return res.Message, nil
}

// GetRecord 获取今日预约记录
func (c *Crawler) GetTodayRecord(ctx context.Context, stuID string) ([]*biz.Record, error) {
	fullURL := "https://kjyy.ccnu.edu.cn/jsq/static/frontApi/user/lastMake"

	records, err := c.getRecord(ctx, stuID, fullURL, TODAY_RECORD_TYPE)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// GetHistory 获取历史记录
func (c *Crawler) GetHistory(ctx context.Context, stuID string) ([]*biz.Record, error) {
	loc, _ := tool.GetLocation()
	today := time.Now().In(loc).Day()
	fullURL := fmt.Sprintf("https://kjyy.ccnu.edu.cn/jsq/static/frontApi/user/history/%d/%d", 0, today)
	records, err := c.getRecord(ctx, stuID, fullURL, HISTORY_RECORD_TYPE)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (c *Crawler) getRecord(ctx context.Context, stuID string, fullURL string, recordType string) ([]*biz.Record, error) {
	token, err := c.getSeatToken(ctx, stuID)
	if err != nil {
		return nil, err
	}
	resp, err := c.doSeatRequestWithToken(ctx, c.client, "POST", fullURL, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 用 gjson 解析
	data := gjson.GetBytes(body, "data")
	if !data.Exists() {
		return nil, nil
	}

	var list gjson.Result
	switch recordType {
	case TODAY_RECORD_TYPE:
		list = data
	case HISTORY_RECORD_TYPE:
		list = data.Get("list")
	}

	if !list.IsArray() {
		return nil, nil
	}

	var records []*biz.Record

	for _, item := range list.Array() {
		beginStr := item.Get("makeBeginStr").String()
		endStr := item.Get("makeEndStr").String()
		dateStr := item.Get("makeDateStr").String()
		dateTime, err := tool.ParseDateStringToTime(dateStr)
		if err != nil {
			c.log.Errorf("解析日期失败：%s", dateStr)
			continue
		}
		begin, err := tool.ParseTimeStringToTime(fmt.Sprintf("%s %s", dateStr, beginStr))
		if err != nil {
			c.log.Errorf("解析时间失败：%s", beginStr)
			continue
		}
		end, err := tool.ParseTimeStringToTime(fmt.Sprintf("%s %s", dateStr, endStr))
		if err != nil {
			c.log.Errorf("解析时间失败：%s", endStr)
			continue
		}
		record := &biz.Record{
			ID:        item.Get("id").String(),
			RoomID:    item.Get("roomId").String(),
			RoomName:  item.Get("roomName").String(),
			SeatID:    item.Get("seatId").String(),
			SeatLabel: item.Get("seatLabel").String(),
			MakeBegin: begin,
			MakeEnd:   end,
			MakeDate:  dateTime,
			BuildName: item.Get("buildName").String(),
			FloorName: item.Get("floorName").String(),
			Message:   item.Get("message").String(),
			Status:    item.Get("status").String(),
		}
		records = append(records, record)
	}

	return records, nil
}

// TODO：改成了违约记录
func (c *Crawler) GetCreditPoint(ctx context.Context, stuID string) (*biz.CreditPoints, error) {
	fullURL := "http://kjyy.ccnu.edu.cn/clientweb/m/a/credit.aspx"

	token, err := c.getSeatToken(ctx, stuID)
	if err != nil {
		return nil, err
	}
	resp, err := c.doSeatRequestWithToken(ctx, c.client, "GET", fullURL, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var summary *biz.CreditSummary
	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		tds := s.Find("td")
		if tds.Length() >= 3 {
			summary = &biz.CreditSummary{
				System: strings.TrimSpace(tds.Eq(0).Text()),
				Remain: strings.TrimSpace(tds.Eq(1).Text()),
				Total:  strings.TrimSpace(tds.Eq(2).Text()),
			}
		}
	})

	var records []*biz.CreditRecord
	doc.Find("#my_resv_list li").Each(func(i int, s *goquery.Selection) {
		record := &biz.CreditRecord{
			Title:    strings.TrimSpace(s.Find(".item-title").Text()),
			Subtitle: strings.TrimSpace(s.Find(".item-subtitle").Text()),
			Location: strings.TrimSpace(s.Find(".item-text").Text()),
		}
		records = append(records, record)
	})

	result := &biz.CreditPoints{
		Summary: summary,
		Records: records,
	}

	return result, nil
}
