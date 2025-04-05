package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	elecpricev1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/elecprice/v1"
	"github.com/asynccnu/ccnubox-be/be-elecprice/domain"
	"github.com/asynccnu/ccnubox-be/be-elecprice/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-elecprice/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/model"
	"net/url"
	"strconv"
	"sync"
	"time"
)

var (
	INTERNET_ERROR = func(err error) error {
		return errorx.New(elecpricev1.ErrorInternetError("网络错误"), "net", err)
	}
	FIND_CONFIG_ERROR = func(err error) error {
		return errorx.New(elecpricev1.ErrorFindConfigError("获取配置失败"), "dao", err)
	}
	SAVE_CONFIG_ERROR = func(err error) error {
		return errorx.New(elecpricev1.ErrorSaveConfigError("保存配置失败"), "dao", err)
	}
)

type ElecpriceService interface {
	SetStandard(ctx context.Context, r *domain.SetStandardRequest) error
	GetStandardList(ctx context.Context, r *domain.GetStandardListRequest) (*domain.GetStandardListResponse, error)
	CancelStandard(ctx context.Context, r *domain.CancelStandardRequest) error
	GetTobePushMSG(ctx context.Context) ([]*domain.ElectricMSG, error)

	GetArchitecture(ctx context.Context, area string) (domain.ResultArchitectureInfo, error)
	GetRoomInfo(ctx context.Context, archiID string, floor string) (map[string]string, error)
	GetPrice(ctx context.Context, roomid string) (*domain.Prices, error)
}

type elecpriceService struct {
	elecpriceDAO dao.ElecpriceDAO
	l            logger.Logger
}

func NewElecpriceService(elecpriceDAO dao.ElecpriceDAO, l logger.Logger) ElecpriceService {
	return &elecpriceService{elecpriceDAO: elecpriceDAO, l: l}
}

func (s *elecpriceService) SetStandard(ctx context.Context, r *domain.SetStandardRequest) error {
	conf := &model.ElecpriceConfig{
		StudentID: r.StudentId,
		Limit:     r.Standard.Limit,
		RoomName:  r.Standard.RoomName,
		TargetID:  r.Standard.RoomId,
	}

	err := s.elecpriceDAO.Upsert(ctx, r.StudentId, r.Standard.RoomId, conf)
	if err != nil {
		return SAVE_CONFIG_ERROR(err)
	}

	return nil
}

func (s *elecpriceService) GetStandardList(ctx context.Context, r *domain.GetStandardListRequest) (*domain.GetStandardListResponse, error) {
	res, err := s.elecpriceDAO.FindAll(ctx, r.StudentId)
	if err != nil {
		return nil, FIND_CONFIG_ERROR(err)
	}

	var standards []*domain.Standard
	for _, r := range res {
		standards = append(standards, &domain.Standard{
			Limit:    r.Limit,
			RoomId:   r.TargetID,
			RoomName: r.RoomName,
		})
	}

	return &domain.GetStandardListResponse{Standard: standards}, nil
}

func (s *elecpriceService) CancelStandard(ctx context.Context, r *domain.CancelStandardRequest) error {
	return s.elecpriceDAO.Delete(ctx, r.StudentId, r.RoomId)
}

func (s *elecpriceService) GetTobePushMSG(ctx context.Context) ([]*domain.ElectricMSG, error) {
	var (
		resultMsgs []*domain.ElectricMSG       // 存储最终结果
		lastID     int64                 = -1  // 初始游标为 -1，表示从头开始
		limit      int                   = 100 // 每次分页查询的大小
	)

	// 用于控制并发量的通道（令牌池），限制同时运行的 goroutine 数量为 10
	maxConcurrency := 10
	semaphore := make(chan struct{}, maxConcurrency)

	for {
		// 分页获取配置数据
		configs, nextID, err := s.elecpriceDAO.GetConfigsByCursor(ctx, lastID, limit)
		if err != nil {
			return nil, err
		}

		// 如果没有更多数据，跳出循环
		if len(configs) == 0 {
			break
		}

		// 用于并发处理的 goroutine
		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			errChan = make(chan error, len(configs))
		)

		for _, config := range configs {
			wg.Add(1)
			// 获取一个令牌（阻塞直到可用）
			semaphore <- struct{}{}

			go func(cfg model.ElecpriceConfig) {
				defer wg.Done()
				// 释放令牌
				defer func() { <-semaphore }()

				// 获取房间的实时电费
				elecPrice, err := s.GetPrice(ctx, cfg.TargetID)

				if err != nil {
					errChan <- err
					return
				}

				// 转换电费数据为浮点数
				Remain, err := strconv.ParseFloat(elecPrice.RemainMoney, 64)

				// 跳过解析失败的数据
				if err != nil {
					errChan <- fmt.Errorf("解析电费数据失败: %v", err)
					return
				}

				// 检查是否符合用户设定的阈值
				if Remain < float64(cfg.Limit) {
					msg := &domain.ElectricMSG{
						RoomName:  &cfg.RoomName,
						StudentId: cfg.StudentID,
						Remain:    &elecPrice.RemainMoney,
					}

					// 并发安全地添加结果
					mu.Lock()
					resultMsgs = append(resultMsgs, msg)
					mu.Unlock()
				}
			}(config)
		}

		// 等待所有 goroutine 完成
		wg.Wait()
		close(errChan)

		// 检查是否有错误
		for err := range errChan {
			if err != nil {
				// 可以选择返回第一个错误，或者记录日志
				return nil, err
			}
		}

		// 更新游标
		lastID = nextID
	}
	return resultMsgs, nil
}

func (s *elecpriceService) GetArchitecture(ctx context.Context, area string) (domain.ResultArchitectureInfo, error) {
	for name_, code := range ConstantMap {
		if area == name_ {
			body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getArchitectureInfo?Area_ID=%s", code))
			if err != nil {
				return domain.ResultArchitectureInfo{}, INTERNET_ERROR(err)
			}
			var result domain.ResultArchitectureInfo

			err = xml.Unmarshal([]byte(body), &result)
			if err != nil {
				return domain.ResultArchitectureInfo{}, INTERNET_ERROR(err)
			}
			return result, nil

		}
	}
	return domain.ResultArchitectureInfo{}, errors.New("不存在的区域")
}

func (s *elecpriceService) GetRoomInfo(ctx context.Context, archiID string, floor string) (map[string]string, error) {
	body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getRoomInfo?Architecture_ID=%s&Floor=%s", archiID, floor))
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}

	rege := `<RoomNo>(\d+)</RoomNo>\s*<RoomName>(.*?)</RoomName>`
	res, err := matchRegex(body, rege)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}

	return res, nil
}

func (s *elecpriceService) GetPrice(ctx context.Context, roomid string) (*domain.Prices, error) {
	mid, err := s.GetMeterID(ctx, roomid)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}

	price, err := s.GetFinalInfo(ctx, mid)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}

	return price, nil
}

func (s *elecpriceService) GetMeterID(ctx context.Context, RoomID string) (string, error) {
	body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getRoomMeterInfo?Room_ID=%s", RoomID))
	if err != nil {
		return "", INTERNET_ERROR(err)
	}

	rege := `<meterId>(.*?)</meterId>`
	id, err := matchRegexpOneEle(body, rege)
	if err != nil {
		return "", INTERNET_ERROR(err)
	}

	return id, nil
}

func (s *elecpriceService) GetFinalInfo(ctx context.Context, meterID string) (*domain.Prices, error) {
	//取余额
	body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getReserveHKAM?AmMeter_ID=%s", meterID))
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	reg1 := `<remainPower>(.*?)</remainPower>`
	remain, err := matchRegexpOneEle(body, reg1)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}

	//取昨天消费
	encodedDate := url.QueryEscape(time.Now().AddDate(0, 0, -1).Format("2006/1/2"))
	body, err = sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getMeterDayValue?AmMeter_ID=%s&startDate=%s&endDate=%s", meterID, encodedDate, encodedDate))
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	reg2 := `<dayValue>(.*?)</dayValue>`
	dayValue, err := matchRegexpOneEle(body, reg2)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	reg3 := `<dayUseMeony>(.*?)</dayUseMeony>`
	dayUseMeony, err := matchRegexpOneEle(body, reg3)
	if err != nil {
		return nil, INTERNET_ERROR(err)
	}
	finalInfo := &domain.Prices{
		RemainMoney:       remain,
		YesterdayUseMoney: dayUseMeony,
		YesterdayUseValue: dayValue,
	}
	return finalInfo, nil
}
