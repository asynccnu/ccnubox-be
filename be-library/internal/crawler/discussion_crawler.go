package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/client"
	"github.com/asynccnu/ccnubox-be/be-library/internal/errcode"
	"github.com/asynccnu/ccnubox-be/be-library/pkg/tool"
	"github.com/tidwall/gjson"
)

// test
func (c *Crawler) GetLibraryDiscussionToken(stuid string) (string, error) {
	return c.getDiscussionToken(context.Background(), stuid)
}

func (c *Crawler) getDiscussionToken(ctx context.Context, stuID string) (string, error) {
	return tool.Retry(func() (string, error) {
		timeoutCtx, cancel := context.WithTimeout(ctx, c.waitTime)
		defer cancel()

		token, err := c.ccnu.GetLibraryDiscussionToken(timeoutCtx, stuID)
		if err != nil {
			return "", err
		}

		return token, nil
	})
}

// doDiscussionRequestWithToken 通用HTTP请求函数
func (c *Crawler) doDiscussionRequestWithToken(ctx context.Context, client client.Client, method, url, token string, body io.Reader) (*http.Response, error) {
	return tool.Retry(func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", token)

		resp, err := client.DoWithContext(ctx, req)
		if err != nil {
			return nil, errcode.ErrCrawler
		}

		return resp, nil
	})
}

// GetDiscussion 获取研讨间信息
func (c *Crawler) GetDiscussion(ctx context.Context, stuID string, roomTypeId, venueId, date string) ([]*biz.Discussion, error) {
	URL := "https://kjyy.ccnu.edu.cn/spa/static/api/book/getRoomList"
	req := &getDiscussionInfoReq{
		CurrentPage: 1,
		PageSize:    9,
		RoomTypeId:  roomTypeId,
		SelectDate:  date,
		VenueId:     venueId,
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	token, err := c.getDiscussionToken(ctx, stuID)
	if err != nil {
		return nil, err
	}
	resp, err := c.doDiscussionRequestWithToken(ctx, c.client, "POST", URL, token, bytes.NewBuffer(reqBytes))
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

	pageList := data.Get("pageList")
	if !pageList.Exists() || !pageList.IsArray() {
		return nil, nil
	}

	var result []*biz.Discussion

	pageList.ForEach(func(_, item gjson.Result) bool {
		dis := &biz.Discussion{
			RoomID:   item.Get("id").String(),
			Name:     item.Get("name").String(),
			VenueID:  item.Get("venue").String(),
			RoomType: item.Get("roomType").String(),
			Address:  item.Get("address").String(),
		}

		var disableList []*biz.DisableTime
		roomTimeSlice := item.Get("roomTimeSliceDtoList.0")
		if roomTimeSlice.Exists() {
			roomTimeSlice.Get("disableTime").ForEach(func(_, dt gjson.Result) bool {
				if dt.IsArray() {
					start := strconv.FormatInt(dt.Get("0").Int(), 10)
					end := strconv.FormatInt(dt.Get("1").Int(), 10)

					disableTime := &biz.DisableTime{
						Start: start,
						End:   end,
					}
					disableList = append(disableList, disableTime)
				}
				return true
			})
		}
		dis.DisableList = disableList
		result = append(result, dis)
		return true
	})
	return result, nil
}

// TODO
// ReserveDiscussion 预约研讨间
func (c *Crawler) ReserveDiscussion(ctx context.Context, stuID string, devid, labid, kindid, title, start, end string, list []string) (string, error) {
	return "", nil
}

// TODO:这个方法已经没有了，改成组队模式
func (c *Crawler) SearchUser(ctx context.Context, stuID string, studentid string) (*biz.Search, error) {
	return nil, nil
}
