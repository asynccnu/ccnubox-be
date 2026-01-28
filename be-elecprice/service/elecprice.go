package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/cache"
	"golang.org/x/sync/errgroup"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-elecprice/domain"
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/model"
	elecpricev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/elecprice/v1"
	errorx "github.com/asynccnu/ccnubox-be/common/pkg/errorx/rpcerr"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
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
	GetRoomInfo(ctx context.Context, archiID string, floor string) (domain.RoomInfoList, error)
	GetPriceById(ctx context.Context, roomid string) (*domain.PriceInfo, error)
	GetPriceByName(ctx context.Context, roomName string) (*domain.Prices, error)
}

type elecpriceService struct {
	elecpriceDAO dao.ElecpriceDAO
	proxyGetter  ProxyGetter
	cache        cache.ElecPriceCache
	l            logger.Logger
}

func NewElecpriceService(elecpriceDAO dao.ElecpriceDAO, l logger.Logger, c cache.ElecPriceCache,
	proxyGetter ProxyGetter) ElecpriceService {
	return &elecpriceService{elecpriceDAO: elecpriceDAO, l: l, proxyGetter: proxyGetter, cache: c}
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
				elecPrice, err := s.GetPriceById(ctx, cfg.TargetID)

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
	if code, ok := ConstantMap[area]; ok {
		var resp domain.ResultArchitectureInfo

		// 查缓存
		cacheData, err := s.cache.GetArchitectureInfos(ctx, area)
		if err == nil && !s.checkEmptyOrNil(cacheData) {
			err = resp.ArchitectureInfoList.Unmarshal(cacheData)
			if err == nil {
				return resp, nil
			}
		}

		body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getArchitectureInfo?Area_ID=%s", code), false)
		if err != nil {
			return domain.ResultArchitectureInfo{}, INTERNET_ERROR(err)
		}

		err = xml.Unmarshal([]byte(body), &resp)
		if err != nil {
			return domain.ResultArchitectureInfo{}, INTERNET_ERROR(err)
		}

		handleDirtyArch(ctx, &resp, area)
		go func() {
			ctxTm, cancelTm := context.WithTimeout(ctx, 2*time.Second)
			defer cancelTm()
			if er := s.cache.SetArchitectureInfos(ctxTm, area, resp.ArchitectureInfoList.Marshal()); er != nil {
				s.l.Warn("SetArchitectureInfos cache warning", logger.Error(er))
			}
		}()
		return resp, nil
	}
	return domain.ResultArchitectureInfo{}, errors.New("不存在的区域")
}

func (s *elecpriceService) GetRoomInfo(ctx context.Context, archiID string, floor string) (domain.RoomInfoList, error) {
	var resp domain.RoomInfoList

	// 查缓存
	cacheData, err := s.cache.GetRoomInfos(ctx, archiID, floor)
	if err == nil && !s.checkEmptyOrNil(cacheData) {
		resp.Unmarshal(cacheData)
		return resp, nil
	}
	// 不管是缓存未命中还是redis内部错误, 都开始爬虫
	body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getRoomInfo?Architecture_ID=%s&Floor=%s", archiID, floor), false)
	if err != nil {
		return resp, INTERNET_ERROR(err)
	}

	res, err := matchRegex(body, roomInfoReg)
	if err != nil {
		return resp, INTERNET_ERROR(err)
	}
	res = filter(res)
	resp = mergeRoomIds(res)
	setRes := resp

	go func() { // 这个是为后续的缓存存储做准备
		ctxTm, cancelTm := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelTm()
		if er := s.cache.SetRoomInfos(ctxTm, archiID, floor, setRes.Marshal()); er != nil {
			s.l.Warn("SetRoomInfos cache warning", logger.Error(er))
		}
	}() // 缓存存储

	go func() { // 下一步会用到这些details
		ctxTm, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		g, ctxEg := errgroup.WithContext(ctxTm)
		for _, detail := range resp.Rooms {
			detail_ := detail // 捕获变量
			g.Go(func() error {
				if err := s.cache.SetRoomDetail(ctxEg, detail_.RoomName, detail_.Marshal()); err != nil {
					s.l.Warn("SetRoomDetail cache warning", logger.Error(err))
				}
				return nil
			})
		}
		_ = g.Wait() // 等待所有存储完成
	}()

	return resp, nil
}

func (s *elecpriceService) GetPriceByName(ctx context.Context, roomName string) (*domain.Prices, error) {
	var (
		resp   = new(domain.Prices)
		detail domain.RoomInfo
	)
	cacheData, err := s.cache.GetRoomDetail(ctx, roomName)
	if err == nil && !s.checkEmptyOrNil(cacheData) {
		detail.Unmarshal(cacheData)
		if detail.IsUnion() {
			res, er := s.GetPriceById(ctx, detail.Union)
			if er != nil {
				return nil, er
			}
			resp.Union = *res
		} else {
			//g, ctxEg := errgroup.WithContext(ctx)

			//g.Go(func() error {
			res, er := s.GetPriceById(ctx, detail.AC)
			if er != nil {
				return nil, er
			}
			resp.AC = *res
			//return resp,nil
			//})
			//g.Go(func() error {
			res, er = s.GetPriceById(ctx, detail.Light)
			if er != nil {
				return nil, er
			}
			resp.Light = *res
			//return nil
			//})
			//if errW := g.Wait(); errW != nil {
			//	return nil, errW
			//}
		}

		return resp, nil
	}

	return nil, errors.New("")
}
func (s *elecpriceService) GetPriceById(ctx context.Context, roomid string) (*domain.PriceInfo, error) {
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
	body, err := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getRoomMeterInfo?Room_ID=%s", RoomID), false)
	if err != nil {
		return "", INTERNET_ERROR(err)
	}

	id, err := matchRegexpOneEle(body, meterIdReg)
	if err != nil {
		return "", INTERNET_ERROR(err)
	}

	return id, nil
}

func (s *elecpriceService) GetFinalInfo(ctx context.Context, meterID string) (*domain.PriceInfo, error) {
	var (
		remain      string
		dayUseMeony string
		dayValue    string
	)
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		//取余额
		body, err_ := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getReserveHKAM?AmMeter_ID=%s", meterID), false)
		//if err_ != nil {
		//	return nil, INTERNET_ERROR(err_)
		//}
		if err_ != nil {
			return INTERNET_ERROR(err_)
		}
		remain, err_ = matchRegexpOneEle(body, remainPowerReg)
		if err_ != nil {
			return INTERNET_ERROR(err_)
		}
		return nil
	})

	g.Go(func() error {
		//取昨天消费
		encodedDate := url.QueryEscape(time.Now().AddDate(0, 0, -1).Format("2006/1/2"))
		body2, err_2 := sendRequest(ctx, fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getMeterDayValue?AmMeter_ID=%s&startDate=%s&endDate=%s", meterID, encodedDate, encodedDate), true)
		//if err_2 != nil {
		//	return nil, INTERNET_ERROR(err_2)
		//}
		if err_2 != nil {
			return INTERNET_ERROR(err_2)
		}

		dayValue, err_2 = matchRegexpOneEle(body2, dayValueReg)
		if err_2 != nil {
			return INTERNET_ERROR(err_2)
		}
		//
		dayUseMeony, err_2 = matchRegexpOneEle(body2, dayUseMeonyReg)
		if err_2 != nil {
			return INTERNET_ERROR(err_2)
		}
		return nil
	})

	if errW := g.Wait(); errW != nil {
		return nil, errW
	}

	finalInfo := &domain.PriceInfo{
		RemainMoney:       remain,
		YesterdayUseMoney: dayUseMeony,
		YesterdayUseValue: dayValue,
	}
	return finalInfo, nil
}

func addProxyAddrToCtx(ctx context.Context, proxyAddr string) context.Context {
	return context.WithValue(ctx, ProxyAddr, proxyAddr)
}

func (s *elecpriceService) checkEmptyOrNil(value string) bool {
	return value == cache.DataValueNil || value == cache.DataValueEmpty || value == ""
}
