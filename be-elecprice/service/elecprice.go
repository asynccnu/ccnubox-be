package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-elecprice/domain"
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/model"
	elecpricev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/elecprice/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"golang.org/x/sync/errgroup"
)

var (
	INTERNET_ERROR    = errorx.FormatErrorFunc(elecpricev1.ErrorInternetError("网络错误"))
	FIND_CONFIG_ERROR = errorx.FormatErrorFunc(elecpricev1.ErrorFindConfigError("获取配置失败"))
	SAVE_CONFIG_ERROR = errorx.FormatErrorFunc(elecpricev1.ErrorSaveConfigError("保存配置失败"))
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
	ProxyService ProxyService
	cache        cache.ElecPriceCache
	l            logger.Logger
}

func NewElecpriceService(elecpriceDAO dao.ElecpriceDAO, l logger.Logger, c cache.ElecPriceCache,
	ProxyService ProxyService) ElecpriceService {
	return &elecpriceService{elecpriceDAO: elecpriceDAO, l: l, ProxyService: ProxyService, cache: c}
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
		return SAVE_CONFIG_ERROR(errorx.Errorf("service: upsert standard failed, sid: %s, rid: %s, err: %w", r.StudentId, r.Standard.RoomId, err))
	}

	return nil
}

func (s *elecpriceService) GetStandardList(ctx context.Context, r *domain.GetStandardListRequest) (*domain.GetStandardListResponse, error) {
	res, err := s.elecpriceDAO.FindAll(ctx, r.StudentId)
	if err != nil {
		return nil, FIND_CONFIG_ERROR(errorx.Errorf("service: find standard list failed, sid: %s, err: %w", r.StudentId, err))
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
	err := s.elecpriceDAO.Delete(ctx, r.StudentId, r.RoomId)
	if err != nil {
		return errorx.Errorf("service: delete standard failed, sid: %s, rid: %s, err: %w", r.StudentId, r.RoomId, err)
	}
	return nil
}

func (s *elecpriceService) GetTobePushMSG(ctx context.Context) ([]*domain.ElectricMSG, error) {
	var (
		resultMsgs []*domain.ElectricMSG
		lastID     int64 = -1
		limit      int   = 100
	)

	maxConcurrency := 10
	semaphore := make(chan struct{}, maxConcurrency)

	for {
		configs, nextID, err := s.elecpriceDAO.GetConfigsByCursor(ctx, lastID, limit)
		if err != nil {
			return nil, errorx.Errorf("service: get configs by cursor failed, lastID: %d, err: %w", lastID, err)
		}

		if len(configs) == 0 {
			break
		}

		var (
			wg sync.WaitGroup
			mu sync.Mutex
		)

		for _, config := range configs {
			wg.Add(1)
			semaphore <- struct{}{}

			go func(cfg model.ElecpriceConfig) {
				defer wg.Done()
				defer func() { <-semaphore }()

				elecPrice, err := s.GetPriceById(ctx, cfg.TargetID)
				if err != nil {
					s.l.Error("service: push worker get price failed", logger.String("rid", cfg.TargetID), logger.Error(err))
					return
				}

				Remain, err := strconv.ParseFloat(elecPrice.RemainMoney, 64)
				if err != nil {
					s.l.Warn("service: push worker parse float failed", logger.String("rid", cfg.TargetID), logger.Error(err))
					return
				}

				if Remain < float64(cfg.Limit) {
					msg := &domain.ElectricMSG{
						RoomName:  &cfg.RoomName,
						StudentId: cfg.StudentID,
						Remain:    &elecPrice.RemainMoney,
						Limit:     &cfg.Limit,
						RoomID:    &cfg.TargetID,
					}
					mu.Lock()
					resultMsgs = append(resultMsgs, msg)
					mu.Unlock()
				}
			}(config)
		}

		wg.Wait()
		lastID = nextID
	}
	return resultMsgs, nil
}

func (s *elecpriceService) GetArchitecture(ctx context.Context, area string) (domain.ResultArchitectureInfo, error) {
	code, ok := ConstantMap[area]
	if !ok {
		return domain.ResultArchitectureInfo{}, errorx.Errorf("service: area not found in constant map: %s", area)
	}

	var resp domain.ResultArchitectureInfo

	// 1. 尝试从缓存获取
	cacheData, err := s.cache.GetArchitectureInfos(ctx, area)
	if err == nil && !s.checkEmptyOrNil(cacheData) {
		if er := resp.ArchitectureInfoList.Unmarshal(cacheData); er == nil {
			return resp, nil
		}
	}

	// 2. 爬取数据
	apiURL := fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getArchitectureInfo?Area_ID=%s", code)
	body, err := sendRequest(ctx, apiURL, false)
	if err != nil {
		return domain.ResultArchitectureInfo{}, INTERNET_ERROR(errorx.Errorf("service: request architecture info failed, area: %s, err: %w", area, err))
	}

	if err = xml.Unmarshal([]byte(body), &resp); err != nil {
		return domain.ResultArchitectureInfo{}, INTERNET_ERROR(errorx.Errorf("service: unmarshal architecture xml failed, area: %s, body: %s, err: %w", area, body, err))
	}

	handleDirtyArch(ctx, &resp, area)

	// 3. 异步回填缓存
	go func() {
		ctxTm, cancelTm := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelTm()
		if er := s.cache.SetArchitectureInfos(ctxTm, area, resp.ArchitectureInfoList.Marshal()); er != nil {
			s.l.Warn("service: async set architecture cache warning", logger.Error(er))
		}
	}()

	return resp, nil
}

func (s *elecpriceService) GetRoomInfo(ctx context.Context, archiID string, floor string) (domain.RoomInfoList, error) {
	var resp domain.RoomInfoList

	// 1. 尝试缓存
	cacheData, err := s.cache.GetRoomInfos(ctx, archiID, floor)
	if err == nil && !s.checkEmptyOrNil(cacheData) {
		resp.Unmarshal(cacheData)
		return resp, nil
	}

	// 2. 爬取
	apiURL := fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getRoomInfo?Architecture_ID=%s&Floor=%s", archiID, floor)
	body, err := sendRequest(ctx, apiURL, false)
	if err != nil {
		return resp, INTERNET_ERROR(errorx.Errorf("service: request room info failed, archiID: %s, floor: %s, err: %w", archiID, floor, err))
	}

	res, err := matchRegex(body, roomInfoReg)
	if err != nil {
		return resp, INTERNET_ERROR(errorx.Errorf("service: match room regex failed, body: %s, err: %w", body, err))
	}

	res = filter(res)
	resp = mergeRoomIds(res)
	setRes := resp

	// 3. 异步回填缓存 (List)
	go func() {
		ctxTm, cancelTm := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelTm()
		if er := s.cache.SetRoomInfos(ctxTm, archiID, floor, setRes.Marshal()); er != nil {
			s.l.Warn("service: async set room info cache warning", logger.Error(er))
		}
	}()

	// 4. 异步回填缓存 (Detail)
	go func() {
		ctxTm, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		g, ctxEg := errgroup.WithContext(ctxTm)
		for _, detail := range resp.Rooms {
			detail_ := detail
			g.Go(func() error {
				if err := s.cache.SetRoomDetail(ctxEg, detail_.RoomName, detail_.Marshal()); err != nil {
					s.l.Warn("service: async set room detail cache warning", logger.String("room", detail_.RoomName), logger.Error(err))
				}
				return nil
			})
		}
		_ = g.Wait()
	}()

	return resp, nil
}

func (s *elecpriceService) GetPriceByName(ctx context.Context, roomName string) (*domain.Prices, error) {
	var (
		resp   = new(domain.Prices)
		detail domain.RoomInfo
	)

	cacheData, err := s.cache.GetRoomDetail(ctx, roomName)
	if err != nil {
		return nil, errorx.Errorf("service: get room detail from cache failed, room: %s, err: %w", roomName, err)
	}

	if s.checkEmptyOrNil(cacheData) {
		return nil, errorx.Errorf("service: room detail not found in cache, room: %s", roomName)
	}

	detail.Unmarshal(cacheData)
	if detail.IsUnion() {
		res, er := s.GetPriceById(ctx, detail.Union)
		if er != nil {
			return nil, er
		}
		resp.Union = *res
	} else {
		// 并行获取空调和照明电费 (优化为并行)
		g, gCtx := errgroup.WithContext(ctx)
		var acRes, lightRes *domain.PriceInfo

		g.Go(func() error {
			var er error
			acRes, er = s.GetPriceById(gCtx, detail.AC)
			return er
		})
		g.Go(func() error {
			var er error
			lightRes, er = s.GetPriceById(gCtx, detail.Light)
			return er
		})

		if er := g.Wait(); er != nil {
			return nil, er
		}
		resp.AC = *acRes
		resp.Light = *lightRes
	}

	return resp, nil
}

func (s *elecpriceService) GetPriceById(ctx context.Context, roomid string) (*domain.PriceInfo, error) {
	mid, err := s.GetMeterID(ctx, roomid)
	if err != nil {
		return nil, err
	}

	price, err := s.GetFinalInfo(ctx, mid)
	if err != nil {
		return nil, err
	}

	return price, nil
}

func (s *elecpriceService) GetMeterID(ctx context.Context, RoomID string) (string, error) {
	apiURL := fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getRoomMeterInfo?Room_ID=%s", RoomID)
	body, err := sendRequest(ctx, apiURL, false)
	if err != nil {
		return "", INTERNET_ERROR(errorx.Errorf("service: request meter id failed, rid: %s, err: %w", RoomID, err))
	}

	id, err := matchRegexpOneEle(body, meterIdReg)
	if err != nil {
		return "", INTERNET_ERROR(errorx.Errorf("service: parse meter id regex failed, rid: %s, body: %s, err: %w", RoomID, body, err))
	}

	return id, nil
}

func (s *elecpriceService) GetFinalInfo(ctx context.Context, meterID string) (*domain.PriceInfo, error) {
	var (
		remain struct{ RemainMoney string }
		dayUse struct {
			DayUseMoney string
			DayUseValue string
		}
	)
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		apiURL := fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getReserveHKAM?AmMeter_ID=%s", meterID)
		body, err_ := sendRequest(ctx, apiURL, false)
		if err_ != nil {
			return INTERNET_ERROR(errorx.Errorf("service: request reserve HKAM failed, mid: %s, err: %w", meterID, err_))
		}
		remain.RemainMoney, err_ = matchRegexpOneEle(body, remainPowerReg)
		if err_ != nil {
			return INTERNET_ERROR(errorx.Errorf("service: parse remain money failed, mid: %s, body: %s, err: %w", meterID, body, err_))
		}
		return nil
	})

	g.Go(func() error {
		encodedDate := url.QueryEscape(time.Now().AddDate(0, 0, -1).Format("2006/1/2"))
		apiURL := fmt.Sprintf("https://jnb.ccnu.edu.cn/ICBS/PurchaseWebService.asmx/getMeterDayValue?AmMeter_ID=%s&startDate=%s&endDate=%s", meterID, encodedDate, encodedDate)
		body, err := sendRequest(ctx, apiURL, true)
		if err != nil {
			return INTERNET_ERROR(errorx.Errorf("service: request meter day value failed, mid: %s, date: %s, err: %w", meterID, encodedDate, err))
		}

		dayUse.DayUseValue, err = matchRegexpOneEle(body, dayValueReg)
		if err != nil {
			return INTERNET_ERROR(errorx.Errorf("service: parse day use value failed, mid: %s, body: %s, err: %w", meterID, body, err))
		}
		dayUse.DayUseMoney, err = matchRegexpOneEle(body, dayUseMeonyReg)
		if err != nil {
			return INTERNET_ERROR(errorx.Errorf("service: parse day use money failed, mid: %s, body: %s, err: %w", meterID, body, err))
		}
		return nil
	})

	if errW := g.Wait(); errW != nil {
		return nil, errW
	}

	return &domain.PriceInfo{
		RemainMoney:       remain.RemainMoney,
		YesterdayUseMoney: dayUse.DayUseMoney,
		YesterdayUseValue: dayUse.DayUseValue,
	}, nil
}

func (s *elecpriceService) checkEmptyOrNil(value string) bool {
	return value == cache.DataValueNil || value == cache.DataValueEmpty || value == ""
}
