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

	"github.com/asynccnu/ccnubox-be/be-library/tool"

	libraryv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/crypto"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
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

var (
	ErrGetSeat = errorx.FormatErrorFunc(libraryv1.ErrorGetSeatError("获取座位失败"))
)

// Crawler 主爬虫结构体
type Crawler struct {
	client *http.Client
	l      logger.Logger
	secret string
}

// NewLibraryCrawler 创建新的图书馆爬虫
func NewLibraryCrawler(client *http.Client, l logger.Logger, secret string) (*Crawler, error) {
	return &Crawler{
		client: client,
		l:      l,
		secret: secret,
	}, nil
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
func (c *Crawler) doSeatRequestWithToken(ctx context.Context, method, url, token string, body io.Reader) (*http.Response, error) {
	return tool.Retry(func() (*http.Response, error) {
		id, sign, ts := crypto.BuildSignWithSecret("POST", c.secret)
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, errorx.Errorf("crawler: create library request failed, err: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Token", token)
		req.Header.Set("LoginType", "PC")
		req.Header.Set("X-Hmac-Request-Key", sign)
		req.Header.Set("X-Request-Date", fmt.Sprintf("%d", ts))
		req.Header.Set("X-Request-Id", id)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.109 Safari/537.36")
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

		resp, err := c.client.Do(req)
		if err != nil {
			c.l.Errorf("crawler: library http request failed, err: %w", err)
			return nil, errorx.Errorf("crawler: library http request failed, err: %w", err)
		}
		return resp, nil
	})
}

// GetSeatInfos 获取不定个给定的房间的座位信息
func (c *Crawler) GetSeatInfos(ctx context.Context, token string, roomIDs []string) (map[string][]*Seat, error) {
	var wg sync.WaitGroup
	results := make(map[string][]*Seat)
	mutex := &sync.Mutex{}

	// 感觉写的有问题
	for _, roomID := range roomIDs {
		wg.Add(1)
		go func() {
			err := func(roomID string) error {
				defer wg.Done()
				seats, err := c.getSeatInfos(ctx, token, roomID)
				if err != nil {
					mutex.Lock()
					results[roomID] = nil
					mutex.Unlock()
					c.l.Errorf("crawler: get seat infos failed, err: %w", err)
					return ErrGetSeat(errorx.Errorf("crawler: get roomid:%s failed", roomID))
				}
				mutex.Lock()
				results[roomID] = seats
				mutex.Unlock()
				return nil
			}(roomID)
			if err != nil {
				mutex.Lock()
				results[roomID] = nil
				mutex.Unlock()
			}
		}()
	}

	wg.Wait()
	return results, nil
}

// getSeatInfos 获取指定房间的座位信息
func (c *Crawler) getSeatInfos(ctx context.Context, token string, roomid string) ([]*Seat, error) {
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

	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	resp, err := c.doSeatRequestWithToken(ctx, "POST", fullURL, token, bytes.NewBuffer(reqBytes))
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

	var result []*Seat

	// 遍历每个 seat
	data.ForEach(func(_, item gjson.Result) bool {
		it := item
		seat := &Seat{
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

func (c *Crawler) GetFreeList(ctx context.Context, token string, seatID string) ([]*FreeTime, error) {
	loc, _ := tool.GetLocation()
	date := time.Now().In(loc).Format("2006-01-02")
	fullURL := fmt.Sprintf("https://kjyy.ccnu.edu.cn/jsq/static/frontApi/res/getTimeLine/%s/%s", seatID, date)

	resp, err := c.doSeatRequestWithToken(ctx, "POST", fullURL, token, nil)
	if err != nil {
		c.l.Errorf("crawler: get freeSeat infos failed, err: %w", err)
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

	var freeList []*FreeTime
	freeListData.ForEach(func(_, item gjson.Result) bool {
		freeTimeLabel := item.Get("label")
		if !freeTimeLabel.Exists() {
			return true
		}
		res := strings.Split(freeTimeLabel.String(), "-")
		if len(res) != 2 {
			return true
		}
		freeTime := &FreeTime{
			Start: res[0],
			End:   res[1],
		}
		freeList = append(freeList, freeTime)
		return true
	})

	return freeList, nil
}

// ReserveSeat 预约座位
func (c *Crawler) ReserveSeat(ctx context.Context, token string, devid, start, end string) (string, error) {
	params := url.Values{}
	params.Add("capToken", "capToken")
	loc, _ := tool.GetLocation()
	date := time.Now().In(loc).Format("2006-01-02")
	path := fmt.Sprintf("%s/%s/%s/%s/%s", ReserveAPIPath, devid, date, start, end)
	fullURL, err := buildURL(path, params)
	if err != nil {
		return "", err
	}
	resp, err := c.doSeatRequestWithToken(ctx, "POST", fullURL, token, nil)
	if err != nil {
		c.l.Errorf("crawler: reserve seat failed, err: %w", err)
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
func (c *Crawler) CancelReserve(ctx context.Context, token string, id string) (string, error) {
	fullURL := fmt.Sprintf("%s/%s", BaseDomain+CancelAPIPath, id)

	resp, err := c.doSeatRequestWithToken(ctx, "POST", fullURL, token, nil)
	if err != nil {
		c.l.Errorf("crawler: cancel reserve failed, err: %w", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var res Response
	err = json.Unmarshal(body, &res)
	if err != nil {
		return "", err
	}
	return res.Message, nil
}

// GetTodayRecord 获取今日预约记录
func (c *Crawler) GetTodayRecord(ctx context.Context, token string) ([]*Record, error) {
	fullURL := "https://kjyy.ccnu.edu.cn/jsq/static/frontApi/user/lastMake"

	records, err := c.getRecord(ctx, token, fullURL, TODAY_RECORD_TYPE)
	if err != nil {
		c.l.Errorf("crawler: get today record failed, err: %w", err)
		return nil, err
	}

	return records, nil
}

// GetHistory 获取历史记录
func (c *Crawler) GetHistory(ctx context.Context, token string) ([]*Record, error) {
	loc, _ := tool.GetLocation()
	today := time.Now().In(loc).Day()
	fullURL := fmt.Sprintf("https://kjyy.ccnu.edu.cn/jsq/static/frontApi/user/history/%d/%d", 0, today)
	records, err := c.getRecord(ctx, token, fullURL, HISTORY_RECORD_TYPE)
	if err != nil {
		c.l.Errorf("crawler: get history record failed, err: %w", err)
		return nil, err
	}
	return records, nil
}

func (c *Crawler) getRecord(ctx context.Context, token string, fullURL string, recordType string) ([]*Record, error) {
	resp, err := c.doSeatRequestWithToken(ctx, "POST", fullURL, token, nil)
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

	var records []*Record

	for _, item := range list.Array() {
		beginStr := item.Get("makeBeginStr").String()
		endStr := item.Get("makeEndStr").String()
		dateStr := item.Get("makeDateStr").String()
		dateTime, err := tool.ParseDateStringToTime(dateStr)
		if err != nil {
			c.l.Errorf("crawler: parsetime failed, err: %w", err)
			continue
		}
		begin, err := tool.ParseTimeStringToTime(fmt.Sprintf("%s %s", dateStr, beginStr))
		if err != nil {
			c.l.Errorf("crawler: parsetime failed, err: %w", err)
			continue
		}
		end, err := tool.ParseTimeStringToTime(fmt.Sprintf("%s %s", dateStr, endStr))
		if err != nil {
			c.l.Errorf("crawler: parsetime failed, err: %w", err)
			continue
		}
		record := &Record{
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
