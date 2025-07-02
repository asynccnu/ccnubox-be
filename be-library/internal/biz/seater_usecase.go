package biz

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

// libraryUsecase struct + methods + NewLibraryUsecase

type libraryUsecase struct {
	crawler LibraryCrawler
	ccnu    CCNUServiceProxy
	Que     DelayQueue

	waitTime time.Duration
	log      *log.Helper
}

func NewLibraryUsecase(crawler LibraryCrawler, ccnu CCNUServiceProxy, logger log.Logger, cf *conf.Server,
	que DelayQueue) LibraryUsecase {
	waitTime := 1200 * time.Millisecond

	if cf.Grpc.Timeout.Seconds > 0 {
		waitTime = cf.Grpc.Timeout.AsDuration()
	}

	uc := &libraryUsecase{
		crawler:  crawler,
		ccnu:     ccnu,
		log:      log.NewHelper(logger),
		waitTime: waitTime,
		Que:      que,
	}

	go func() {
		if err := uc.Que.Consume("be-library-refresh-retry", uc.handleRetryMsg); err != nil {
			uc.log.Errorf("Error consuming retry message: %v", err)
		}
	}()

	return uc
}

// 爬取座位信息
func (u *libraryUsecase) GetSeatFromCrawler(ctx context.Context, stuID string, RoomID string) ([]*Seat, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, u.waitTime)
	defer cancel()

	getCookieStart := time.Now()

	cookie, err := u.ccnu.GetCookie(timeoutCtx, stuID)
	if err != nil {
		u.log.Errorf("Error getting cookie(stu_id:%v) from other service: %v", stuID, err)
		return nil, err
	}

	u.log.Infof("Get cookie (stu_id:%v) from other service,cost %v", stuID, time.Since(getCookieStart))

	seats, err := u.crawler.GetSeatInfos(ctx, RoomID)
	if err != nil {
		u.log.Errorf("craw seats(stu_id:%v cookie:%v) failed: %v", stuID, cookie, err)
		return nil, err
	}
	return seats, nil
}

// 处理重试消息
func (u *libraryUsecase) handleRetryMsg(key, val []byte) {

}
