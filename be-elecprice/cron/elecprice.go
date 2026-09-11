package cron

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/asynccnu/ccnubox-be/be-elecprice/conf"
	"github.com/asynccnu/ccnubox-be/be-elecprice/service"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

func getElecpriceEventUrl(roomID string) string {
	value := url.Values{}
	value.Add("room_id", fmt.Sprintf(`"%s"`, roomID))
	return fmt.Sprintf("ccnubox://electricityBillinBalance?%s", value.Encode())
}

// energyDedupeKey 以学生+房间+日期生成稳定消息 ID，同一天内同一房间的低电费提醒重投会被去重。
func energyDedupeKey(studentID, roomID string, day time.Time) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%s", studentID, roomID, day.Format("2006-01-02"))))
	return "energy:" + hex.EncodeToString(sum[:])
}

type ElecpriceController struct {
	feedClient      feedv1.FeedServiceClient
	elecpriceSerice service.ElecpriceService
	stopChan        chan struct{}
	durationTime    time.Duration
	l               logger.Logger
}

func NewElecpriceController(
	feedClient feedv1.FeedServiceClient,
	elecpriceSerice service.ElecpriceService,
	l logger.Logger,
	cfg *conf.ServerConf,
) *ElecpriceController {
	return &ElecpriceController{
		feedClient:      feedClient,
		elecpriceSerice: elecpriceSerice,
		stopChan:        make(chan struct{}),
		durationTime:    time.Duration(cfg.ElecpriceController.DurationTime) * time.Hour,
		l:               l,
	}
}

func (r *ElecpriceController) StartCronTask() {
	go func() {
		ticker := time.NewTicker(r.durationTime)
		for {
			select {
			case <-ticker.C:
				ctx := context.Background()
				err := r.publishMSG(ctx)
				l := r.l.WithContext(ctx)
				if err != nil {
					l.Error("推送消息失败!:", logger.Error(err))

				}

			case <-r.stopChan:
				ticker.Stop()
				return
			}
		}
	}() //定时控制器

}

func (r *ElecpriceController) publishMSG(ctx context.Context) error {
	//由于没有使用注册为路由这里手动写的上下文,每次提前四天进行提醒

	msgs, err := r.elecpriceSerice.GetTobePushMSG(ctx)
	if err != nil {
		return err
	}
	for i := range msgs {
		if msgs[i].Remain != nil {
			url := getElecpriceEventUrl(*msgs[i].RoomID)
			//发送给全体成员
			_, err = r.feedClient.PublicFeedEvent(ctx, &feedv1.PublicFeedEventReq{
				StudentId: msgs[i].StudentId,
				Event: &feedv1.FeedEvent{
					Type:         feedv1.FeedEventType_ENERGY,
					Title:        "电费不足提醒",
					Content:      fmt.Sprintf("房间%s剩余 %s 元，已低于设定阈值 %d 元，请及时充值。", *(msgs[i].RoomName), *(msgs[i].Remain), *(msgs[i].Limit)),
					Url:          url,
					ExtendFields: map[string]string{"url": url},
					DedupeKey:    energyDedupeKey(msgs[i].StudentId, *(msgs[i].RoomID), time.Now()),
				},
			})
		}

	}

	return err
}
