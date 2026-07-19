package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/asynccnu/ccnubox-be/be-library/tool"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/tidwall/gjson"
)

func (c *Crawler) doDiscussionRequestWithToken(ctx context.Context, method, url, token string, body []byte) (*http.Response, error) {
	return tool.Retry(func() (*http.Response, error) {
		var requestBody *bytes.Reader
		if body == nil {
			requestBody = bytes.NewReader(nil)
		} else {
			requestBody = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, requestBody)
		if err != nil {
			return nil, errorx.Errorf("create discussion request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Authorization", token)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		return c.client.Do(req)
	})
}

func (c *Crawler) GetDiscussion(ctx context.Context, token string, roomTypeId, venueId, date string) ([]*Discussion, error) {
	const pageSize = 10
	URL := c.baseURL + "/spa/static/api/book/getRoomList"
	var result []*Discussion
	for page := 1; page <= 100; page++ {
		reqBytes, err := json.Marshal(&getDiscussionInfoReq{
			CurrentPage: page,
			PageSize:    pageSize,
			RoomTypeId:  roomTypeId,
			SelectDate:  date,
			VenueId:     venueId,
		})
		if err != nil {
			return nil, errorx.Errorf("encode discussion request: %w", err)
		}
		resp, err := c.doDiscussionRequestWithToken(ctx, http.MethodPost, URL, token, reqBytes)
		if err != nil {
			return nil, errorx.Errorf("request discussion page %d: %w", page, err)
		}
		body, readErr := readSuccessfulResponse(resp)
		resp.Body.Close()
		if readErr != nil {
			return nil, errorx.Errorf("read discussion page %d: %w", page, readErr)
		}

		data := gjson.GetBytes(body, "data")
		pageList := data.Get("pageList")
		if !pageList.IsArray() {
			return result, nil
		}
		pageList.ForEach(func(_, item gjson.Result) bool {
			dis := &Discussion{
				RoomID:   item.Get("id").String(),
				Name:     item.Get("name").String(),
				VenueID:  firstNonEmpty(item.Get("venueId").String(), item.Get("venue.id").String(), item.Get("venue").String()),
				RoomType: firstNonEmpty(item.Get("roomType.name").String(), item.Get("roomType").String()),
				Address:  item.Get("address").String(),
			}

			item.Get("roomTimeSliceDtoList").ForEach(func(_, roomTimeSlice gjson.Result) bool {
				roomTimeSlice.Get("disableTime").ForEach(func(_, dt gjson.Result) bool {
					if dt.IsArray() && len(dt.Array()) >= 2 {
						dis.DisableList = append(dis.DisableList, &DisableTime{
							Start: strconv.FormatInt(dt.Get("0").Int(), 10),
							End:   strconv.FormatInt(dt.Get("1").Int(), 10),
						})
					}
					return true
				})
				return true
			})
			result = append(result, dis)
			return true
		})

		totalPage := int(data.Get("totalPage").Int())
		if pageList.Get("#").Int() == 0 || (totalPage > 0 && page >= totalPage) || (totalPage == 0 && len(pageList.Array()) < pageSize) {
			break
		}
	}
	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" && value != "{}" {
			return value
		}
	}
	return ""
}

func (c *Crawler) ReserveDiscussion(context.Context, string, string, string, string, string, string, string, []string) (string, error) {
	return "", errorx.New("discussion reservation is unavailable in the current library system")
}
