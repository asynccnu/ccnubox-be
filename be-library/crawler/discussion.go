package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/tidwall/gjson"
)

func (c *Crawler) doDiscussionRequestWithToken(ctx context.Context, method, url, token string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	return c.client.Do(req)
}

func (c *Crawler) GetDiscussion(ctx context.Context, token string, roomTypeId, venueId, date string) ([]*Discussion, error) {
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
	resp, err := c.doDiscussionRequestWithToken(ctx, "POST", URL, token, bytes.NewBuffer(reqBytes))
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

	var result []*Discussion

	pageList.ForEach(func(_, item gjson.Result) bool {
		dis := &Discussion{
			RoomID:   item.Get("id").String(),
			Name:     item.Get("name").String(),
			VenueID:  item.Get("venue").String(),
			RoomType: item.Get("roomType").String(),
			Address:  item.Get("address").String(),
		}

		var disableList []*DisableTime
		roomTimeSlice := item.Get("roomTimeSliceDtoList.0")
		if roomTimeSlice.Exists() {
			roomTimeSlice.Get("disableTime").ForEach(func(_, dt gjson.Result) bool {
				if dt.IsArray() {
					start := strconv.FormatInt(dt.Get("0").Int(), 10)
					end := strconv.FormatInt(dt.Get("1").Int(), 10)
					disableTime := &DisableTime{
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

func (c *Crawler) ReserveDiscussion(ctx context.Context, token string, devid, labid, kindid, title, start, end string, list []string) (string, error) {
	return "", nil
}
