package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/client"
	"github.com/asynccnu/ccnubox-be/be-library/internal/errcode"
	"github.com/asynccnu/ccnubox-be/be-library/internal/model"
	"github.com/asynccnu/ccnubox-be/be-library/pkg/tool"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tidwall/gjson"
)

// 定义全局URL常量
const (
	BaseDomain = "http://kjyy.ccnu.edu.cn"
)

// API端点路径
var (
	DeviceAPIPath     = "/ClientWeb/pro/ajax/device.aspx"
	ReserveAPIPath    = "/ClientWeb/pro/ajax/reserve.aspx"
	SearchAccountPath = "/ClientWeb/pro/ajax/data/searchAccount.aspx"
)

// Crawler 主爬虫结构体
type Crawler struct {
	log        *log.Helper
	cookiePool *client.CookiePool
	ccnu       biz.CCNUServiceProxy
	waitTime   time.Duration
}

// NewLibraryCrawler 创建新的图书馆爬虫
func NewLibraryCrawler(logger log.Logger, cookiePool *client.CookiePool, ccnu biz.CCNUServiceProxy, waitTime time.Duration) biz.LibraryCrawler {
	return &Crawler{
		log:        log.NewHelper(logger),
		cookiePool: cookiePool,
		ccnu:       ccnu,
		waitTime:   waitTime,
	}
}

func (c *Crawler) getClient(ctx context.Context, stuID string) (*client.CookieClient, error) {
	return tool.Retry(func() (*client.CookieClient, error) {
		timeoutCtx, cancel := context.WithTimeout(ctx, c.waitTime)
		defer cancel()

		cookie, err := c.ccnu.GetLibraryCookie(timeoutCtx, stuID)
		if err != nil {
			return nil, err
		}

		return c.cookiePool.GetClient(cookie)
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

// doRequest 通用HTTP请求函数
func (c *Crawler) doRequest(ctx context.Context, client *client.CookieClient, method, url string, body io.Reader) (*http.Response, error) {
	return tool.Retry(func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "https://account.ccnu.edu.cn")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.109 Safari/537.36")

		resp, err := client.DoWithContext(ctx, req)
		if err != nil {
			return nil, errcode.ErrCrawler
		}

		return resp, nil
	})
}

// GetSeatInfos 获取不定个给定的房间的座位信息
func (c *Crawler) GetSeatInfos(ctx context.Context, stuID string, roomIDs []string) (map[string][]*biz.Seat, error) {
	cli, err := c.getClient(ctx, stuID)
	if err != nil {
		c.log.Errorf("Error getting client(stu_id:%v): %v", stuID, err)
		return nil, err
	}

	var wg sync.WaitGroup
	results := make(map[string][]*biz.Seat)
	mutex := &sync.Mutex{}

	for _, roomID := range roomIDs {
		wg.Add(1)
		go func(roomID string) {
			defer wg.Done()
			seats, err := c.getSeatInfos(ctx, cli, roomID)
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
	fmt.Println("infos:", results)
	return results, nil
}

// getSeatInfos 获取指定房间的座位信息
func (c *Crawler) getSeatInfos(ctx context.Context, client *client.CookieClient, roomid string) ([]*biz.Seat, error) {
	date := time.Now().Format("2006-01-02")

	params := url.Values{}
	params.Add("classkind", "8")
	params.Add("room_id", roomid)
	params.Add("date", date)
	params.Add("act", "get_rsv_sta")

	fullURL, err := buildURL(DeviceAPIPath, params)
	if err != nil {
		return nil, errcode.ErrCrawler
	}

	resp, err := c.doRequest(ctx, client, "GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println("body:", string(body))

	// 用 gjson 解析
	data := gjson.GetBytes(body, "data")
	if !data.Exists() {
		return nil, nil
	}

	var result []*biz.Seat

	// 遍历每个 seat
	data.ForEach(func(_, item gjson.Result) bool {
		seat := &biz.Seat{
			LabName:  item.Get("labName").String(),
			RoomName: item.Get("roomName").String(),
			RoomID:   item.Get("roomId").String(),
			DevID:    item.Get("devId").String(),
			DevName:  item.Get("devName").String(),
		}

		// 提取时间段 ops
		item.Get("ts").ForEach(func(_, op gjson.Result) bool {
			ts := &biz.TimeSlot{
				Start:  op.Get("start").String(),
				End:    op.Get("end").String(),
				State:  op.Get("state").String(),
				Owner:  op.Get("owner").String(),
				Occupy: op.Get("occupy").Bool(),
			}
			seat.Ts = append(seat.Ts, ts)
			return true
		})

		result = append(result, seat)
		return true
	})

	fmt.Println("info:", result)
	return result, nil
}

// test
func (c *Crawler) GetLibraryCookie(stuid string) (string, error) {
	return c.ccnu.GetLibraryCookie(context.Background(), stuid)
}

// ReserveSeat 预约座位
func (c *Crawler) ReserveSeat(ctx context.Context, stuID string, devid, start, end string) (string, error) {
	cli, err := c.getClient(ctx, stuID)
	if err != nil {
		c.log.Errorf("Error getting client(stu_id:%v): %v", stuID, err)
		return "", err
	}

	params := url.Values{}
	params.Add("dev_id", devid)
	params.Add("start", start)
	params.Add("end", end)
	params.Add("act", "set_resv")

	fullURL, err := buildURL(ReserveAPIPath, params)
	if err != nil {
		return "", errcode.ErrCrawler
	}

	resp, err := c.doRequest(ctx, cli, "GET", fullURL, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ReserveResp model.Response
	if err = json.Unmarshal(body, &ReserveResp); err != nil {
		return "", err
	}

	if ReserveResp.Ret != 1 {
		return "", fmt.Errorf(ReserveResp.Msg)
	}

	return ReserveResp.Msg, nil
}

// GetRecord 获取预约记录
func (c *Crawler) GetRecord(ctx context.Context, stuID string) ([]*biz.FutureRecords, error) {
	cli, err := c.getClient(ctx, stuID)
	if err != nil {
		c.log.Errorf("Error getting client(stu_id:%v): %v", stuID, err)
		return nil, err
	}

	params := url.Values{}
	params.Add("act", "get_my_resv")

	fullURL, err := buildURL(ReserveAPIPath, params)
	if err != nil {
		return nil, errcode.ErrCrawler
	}

	resp, err := c.doRequest(ctx, cli, "GET", fullURL, nil)
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

	var result []*biz.FutureRecords

	// 遍历每个record
	for _, item := range data.Array() {
		rawStates := item.Get("states").String()
		html := "<div>" + rawStates + "</div>"

		// 使用goquery提取span内的纯文本
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			continue
		}

		var plainStates []string
		doc.Find("span").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				plainStates = append(plainStates, text)
			}
		})

		record := &biz.FutureRecords{
			ID:       item.Get("id").String(),
			Owner:    item.Get("owner").String(),
			Start:    item.Get("start").String(),
			End:      item.Get("end").String(),
			TimeDesc: item.Get("timeDesc").String(),
			States:   strings.Join(plainStates, ","),
			DevName:  item.Get("devName").String(),
			RoomID:   item.Get("roomId").String(),
			RoomName: item.Get("roomName").String(),
			LabName:  item.Get("labName").String(),
		}

		result = append(result, record)
	}

	return result, nil
}

// GetHistory 获取历史记录
func (c *Crawler) GetHistory(ctx context.Context, stuID string) ([]*biz.HistoryRecords, error) {
	cli, err := c.getClient(ctx, stuID)
	if err != nil {
		c.log.Errorf("Error getting client(stu_id:%v): %v", stuID, err)
		return nil, err
	}

	fullURL := "http://kjyy.ccnu.edu.cn/clientweb/m/a/resvlist.aspx"

	resp, err := c.doRequest(ctx, cli, "GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var records []*biz.HistoryRecords

	doc.Find("li.item-content").Each(func(i int, item *goquery.Selection) {
		place := item.Find(".item-title").Text()
		status := item.Find(".item-after").Text()
		date := item.Find(".item-subtitle").Text()
		submitText := item.Find(".item-text").Text()
		submitParts := strings.Split(submitText, ",")
		if len(submitParts) >= 2 {
			floor := submitParts[0]
			floor = strings.TrimSpace(floor)
			submitTime := submitParts[2]
			submitTime = strings.TrimSpace(submitTime)

			records = append(records, &biz.HistoryRecords{
				Place:      place,
				Floor:      floor,
				Status:     status,
				Date:       date,
				SubmitTime: submitTime,
			})
		}
	})

	return records, nil
}

// GetCreditPoint 获取信用积分
func (c *Crawler) GetCreditPoint(ctx context.Context, stuID string) (*biz.CreditPoints, error) {
	cli, err := c.getClient(ctx, stuID)
	if err != nil {
		c.log.Errorf("Error getting client(stu_id:%v): %v", stuID, err)
		return nil, err
	}

	fullURL := "http://kjyy.ccnu.edu.cn/clientweb/m/a/credit.aspx"

	resp, err := c.doRequest(ctx, cli, "GET", fullURL, nil)
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

// GetDiscussion 获取研讨间信息
func (c *Crawler) GetDiscussion(ctx context.Context, stuID string, classid, date string) ([]*biz.Discussion, error) {
	cli, err := c.getClient(ctx, stuID)
	if err != nil {
		c.log.Errorf("Error getting client(stu_id:%v): %v", stuID, err)
		return nil, err
	}

	params := url.Values{}
	params.Add("classkind", "1")
	params.Add("class_id", classid)
	params.Add("date", date)
	params.Add("act", "get_rsv_sta")

	fullURL, err := buildURL(DeviceAPIPath, params)
	if err != nil {
		return nil, errcode.ErrCrawler
	}

	resp, err := c.doRequest(ctx, cli, "GET", fullURL, nil)
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

	var result []*biz.Discussion

	data.ForEach(func(_, item gjson.Result) bool {
		dis := &biz.Discussion{
			LabID:    item.Get("labId").String(),
			LabName:  item.Get("labName").String(),
			KindID:   item.Get("kindId").String(),
			KindName: item.Get("kindName").String(),
			DevID:    item.Get("devId").String(),
			DevName:  item.Get("devName").String(),
		}

		item.Get("ts").ForEach(func(_, op gjson.Result) bool {
			ts := &biz.DiscussionTS{
				Start:  op.Get("start").String(),
				End:    op.Get("end").String(),
				State:  op.Get("state").String(),
				Title:  op.Get("title").String(),
				Owner:  op.Get("owner").String(),
				Occupy: op.Get("occupy").Bool(),
			}
			dis.TS = append(dis.TS, ts)
			return true
		})

		result = append(result, dis)
		return true
	})

	return result, nil
}

// SearchUser 搜索用户
func (c *Crawler) SearchUser(ctx context.Context, stuID string, studentid string) (*biz.Search, error) {
	cli, err := c.getClient(ctx, stuID)
	if err != nil {
		c.log.Errorf("Error getting client(stu_id:%v): %v", stuID, err)
		return nil, err
	}

	params := url.Values{}
	params.Add("term", studentid)

	fullURL, err := buildURL(SearchAccountPath, params)
	if err != nil {
		return nil, errcode.ErrCrawler
	}

	resp, err := c.doRequest(ctx, cli, "GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var search []*biz.Search
	if err := json.Unmarshal(body, &search); err != nil {
		return nil, err
	}

	return search[0], nil
}

// ReserveDiscussion 预约研讨间
func (c *Crawler) ReserveDiscussion(ctx context.Context, stuID string, devid, labid, kindid, title, start, end string, list []string) (string, error) {
	cli, err := c.getClient(ctx, stuID)
	if err != nil {
		c.log.Errorf("Error getting client(stu_id:%v): %v", stuID, err)
		return "", err
	}

	mbList := "$" + strings.Join(list, ",")

	params := url.Values{}
	params.Add("dev_id", devid)
	params.Add("lab_id", labid)
	params.Add("kind_id", kindid)
	params.Add("min_user", "3")
	params.Add("max_user", "4")
	params.Add("test_name", title)
	params.Add("mb_list", mbList)
	params.Add("start", start)
	params.Add("end", end)
	params.Add("act", "set_resv")

	fullURL, err := buildURL(ReserveAPIPath, params)
	if err != nil {
		return "", errcode.ErrCrawler
	}

	resp, err := c.doRequest(ctx, cli, "GET", fullURL, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ReserveResp model.Response
	if err = json.Unmarshal(body, &ReserveResp); err != nil {
		return "", err
	}

	if ReserveResp.Ret != 1 {
		return "", fmt.Errorf(ReserveResp.Msg)
	}

	return ReserveResp.Msg, nil
}

// CancelReserve 取消预约
func (c *Crawler) CancelReserve(ctx context.Context, stuID string, id string) (string, error) {
	cli, err := c.getClient(ctx, stuID)
	if err != nil {
		c.log.Errorf("Error getting client(stu_id:%v): %v", stuID, err)
		return "", err
	}

	params := url.Values{}
	params.Add("act", "del_resv")
	params.Add("id", id)

	fullURL, err := buildURL(ReserveAPIPath, params)
	if err != nil {
		return "", errcode.ErrCrawler
	}

	resp, err := c.doRequest(ctx, cli, "GET", fullURL, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	jsonRegexp := regexp.MustCompile(`\{[^}]+}`)
	matches := jsonRegexp.FindAll(body, -1)

	var CancelResp model.Response
	for _, m := range matches {
		if err = json.Unmarshal(m, &CancelResp); err != nil {
			continue // 忽略无效块
		}
		if CancelResp.Ret == 1 {
			return CancelResp.Msg, nil
		}
		return "", fmt.Errorf(CancelResp.Msg)
	}

	return CancelResp.Msg, nil
}
