package biz

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/pkg/tool"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

const TodayRecordTTL = 30 * time.Minute

type libraryBiz struct {
	crawler          LibraryCrawler
	log              *log.Helper
	SeatRepo         SeatRepo
	RecordRepo       RecordRepo
	CreditPointsRepo CreditPointsRepo
}

func NewLibraryBiz(crawler LibraryCrawler, logger log.Logger, seatRepo SeatRepo, recordRepo RecordRepo, creditPointsRepo CreditPointsRepo) LibraryBiz {
	return &libraryBiz{
		crawler:          crawler,
		log:              log.NewHelper(logger),
		SeatRepo:         seatRepo,
		RecordRepo:       recordRepo,
		CreditPointsRepo: creditPointsRepo,
	}
}

func (b *libraryBiz) GetSeat(ctx context.Context, stuID string, RoomIDs []string) (map[string][]*Seat, error) {
	data, err := b.SeatRepo.GetSeatInfos(ctx, stuID, RoomIDs)
	if err != nil {
		b.log.Errorf("get seats from cache(stu_id:%v) failed: %v", stuID, err)
		return nil, err
	}
	return data, nil
}

func (b *libraryBiz) ReserveSeat(ctx context.Context, stuID, devID, start, end string) (string, error) {
	message, err := b.crawler.ReserveSeat(ctx, stuID, devID, start, end)
	if err != nil {
		b.log.Errorf("reserve seats(stu_id:%v) failed: %v", stuID, err)
		return "", err
	}
	//预约后就需要刷新今日预约记录
	todayRecords, err := b.crawler.GetTodayRecord(ctx, stuID)
	if err != nil {
		b.log.Errorf("crawler today record(stu_id:%v) failed:%v", stuID, err)
		return "", err
	}
	err = b.RecordRepo.UpsertRecords(ctx, stuID, todayRecords)
	if err != nil {
		b.log.Errorf("update today record(stu_id:%v) failed:%v", stuID, err)
		return "", err
	}
	return message, nil
}

func (b *libraryBiz) GetRecordByDate(ctx context.Context, stuID string, dateStrs ...string) ([]*Record, error) {
	date := make([]time.Time, 0, len(dateStrs))
	for _, s := range dateStrs {
		d, err := tool.ParseDateStringToTime(s)
		if err != nil {
			b.log.Errorf("parse time(stuID:%s) error:%v", stuID, err)
			continue
		}
		date = append(date, d)
	}

	needHistoryCrawler := false
	needTodayCrawler := false

	lastUpdate, err := b.RecordRepo.GetRecordUpdateTime(ctx, stuID)
	//如果key不存在，说明之前没有更新过，直接更新
	if err != nil {
		if errors.Is(err, redis.Nil) {
			needHistoryCrawler = true
			needTodayCrawler = true
		} else {
			return nil, err
		}
	} else {
		lastUpdateTime, err := tool.ParseTimeStringToTime(lastUpdate)
		if err != nil {
			return nil, err
		}

		//判断今日预约记录是否要刷新
		for _, d := range date {
			if tool.IsSameDay(d, time.Now()) && time.Since(lastUpdateTime) > TodayRecordTTL {
				needTodayCrawler = true
				break
			}
		}
		//判断历史数据是否需要刷新
		if !tool.IsSameDay(lastUpdateTime, time.Now()) {
			needHistoryCrawler = true
		}
	}

	var record []*Record
	if needHistoryCrawler {
		historyRecord, err := b.crawler.GetHistory(ctx, stuID)
		if err != nil {
			return nil, err
		}
		record = append(record, historyRecord...)
	}
	if needTodayCrawler {
		todayRecord, err := b.crawler.GetTodayRecord(ctx, stuID)
		if err != nil {
			return nil, err
		}
		record = append(record, todayRecord...)
	}
	if len(record) > 0 {
		err = b.RecordRepo.UpsertRecords(ctx, stuID, record)
		if err != nil {
			b.log.Errorf("update today record(stu_id:%v) failed:%v", stuID, err)
			return nil, err
		}
	}

	res, err := b.RecordRepo.ListRecords(ctx, stuID, date...)
	if err != nil {
		return nil, err
	}
	return res, nil

}

func (b *libraryBiz) GetCreditPoint(ctx context.Context, stuID string) (*CreditPoints, error) {
	creditPoints, err := b.crawler.GetCreditPoint(ctx, stuID)
	if err != nil {
		b.log.Errorf("get credit points(stu_id:%v) failed: %v", stuID, err)
		return nil, err
	}
	// 去重并持久化
	if err = b.CreditPointsRepo.UpsertCreditPoint(ctx, stuID, creditPoints); err != nil {
		b.log.Warnf("persist credit points(stu_id:%v) failed: %v", stuID, err)
		return nil, err
	}
	// 从数据库读取去重后的数据
	result, err := b.CreditPointsRepo.ListCreditPoint(ctx, stuID)
	if err != nil {
		b.log.Errorf("list credit points(stu_id:%v) failed: %v", stuID, err)
		return nil, err
	}
	return result, nil
}

func (b *libraryBiz) GetDiscussion(ctx context.Context, stuID, roomTypeID, venueID, date string) ([]*Discussion, error) {
	discussions, err := b.crawler.GetDiscussion(ctx, stuID, roomTypeID, venueID, date)
	if err != nil {
		b.log.Errorf("get discussions(stu_id:%v) failed: %v", stuID, err)
		return nil, err
	}
	return discussions, nil
}

func (b *libraryBiz) SearchUser(ctx context.Context, stuID, studentID string) (*Search, error) {
	user, err := b.crawler.SearchUser(ctx, stuID, studentID)
	if err != nil {
		b.log.Errorf("search user(stu_id:%v for student_id:%v) failed: %v", stuID, studentID, err)
		return nil, err
	}
	return user, nil
}

func (b *libraryBiz) ReserveDiscussion(ctx context.Context, stuID, devID, labID, kindID, title, start, end string, list []string) (string, error) {
	message, err := b.crawler.ReserveDiscussion(ctx, stuID, devID, labID, kindID, title, start, end, list)
	if err != nil {
		b.log.Errorf("reserve discussion(stu_id:%v) failed: %v", stuID, err)
		return "", err
	}
	return message, nil
}

func (b *libraryBiz) CancelReserve(ctx context.Context, stuID, id string) (string, error) {
	message, err := b.crawler.CancelReserve(ctx, stuID, id)
	if err != nil {
		b.log.Errorf("cancel reserve(stu_id:%v id:%v) failed: %v", stuID, id, err)
		return "", err
	}
	return message, nil
}

// 2025-09-02 20:00
func (b *libraryBiz) ReserveSeatRandomly(ctx context.Context, stuID, start, end string, roomIDs []string) (string, bool, error) {
	qStart, _ := tool.ParseTodayTimeStringToUnix(start)
	qEnd, _ := tool.ParseTodayTimeStringToUnix(end)

	// 查找空闲预约
	seatDevID, isExist, err := b.SeatRepo.FindFirstAvailableSeat(ctx, qStart, qEnd, roomIDs, stuID)
	if err != nil {
		return "", false, err
	}
	if !isExist {
		return "", false, nil
	}

	//要把时间转换成分钟
	startTime, _ := tool.ParseTodayTimeStringToTime(start)
	startMinute := strconv.Itoa(tool.ParseTimeToMinute(startTime))
	endTime, _ := tool.ParseTodayTimeStringToTime(end)
	endMinute := strconv.Itoa(tool.ParseTimeToMinute(endTime))

	// 执行预约操作
	msg, err := b.ReserveSeat(ctx, stuID, seatDevID, startMinute, endMinute)
	if err != nil {
		b.log.Errorf("Randomly reserve(stu_id:%v seatid:%v) failed: %v", stuID, seatDevID, err)
		return "", false, err
	}
	return msg, true, nil
}
