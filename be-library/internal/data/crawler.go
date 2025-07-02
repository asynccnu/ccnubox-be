package data

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/errcode"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tidwall/gjson"
)

type Crawler struct {
	log    *log.Helper
	client *http.Client
}

func NewLibraryCrawler(logger log.Logger) biz.LibraryCrawler {
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

func (c *Crawler) GetSeatInfos(ctx context.Context, roomid string) ([]*biz.Seat, error) {
	baseURL := "http://kjyy.ccnu.edu.cn/ClientWeb/pro/ajax/device.aspx"

	params := url.Values{}
	params.Set("classkind", "8")
	params.Set("room_id", "101699187")
	params.Set("date", "2025-07-02")
	params.Set("act", "get_rsv_sta")

	fullURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Fatal("创建请求失败:", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errcode.ErrCrawler
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
		seat := &biz.Seat{
			Name:     item.Get("name").String(),
			DevID:    item.Get("devId").String(),
			KindName: item.Get("kindName").String(),
		}

		// 提取时间段 ops
		item.Get("ops").ForEach(func(_, op gjson.Result) bool {
			ts := &biz.TimeSlot{
				Start: op.Get("start").String(),
				End:   op.Get("end").String(),
				Owner: op.Get("owner").String(), // 可能是 null，这里会是 ""
			}
			seat.Ts = append(seat.Ts, ts)
			return true
		})

		result = append(result, seat)
		return true
	})

	return result, nil
}
