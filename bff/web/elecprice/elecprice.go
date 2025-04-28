package elecprice

import (
	elecpricev1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/elecprice/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"sort"
	"strconv"
)

type ElecPriceHandler struct {
	ElecPriceClient elecpricev1.ElecpriceServiceClient //注入的是grpc服务
	Administrators  map[string]struct{}                //这里注入的是管理员权限验证配置
}

func NewElecPriceHandler(elecPriceClient elecpricev1.ElecpriceServiceClient,
	administrators map[string]struct{}) *ElecPriceHandler {
	return &ElecPriceHandler{ElecPriceClient: elecPriceClient, Administrators: administrators}
}

func (h *ElecPriceHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/elecprice")
	{
		sg.GET("/getArchitecture", authMiddleware, ginx.WrapClaimsAndReq(h.GetArchitecture))
		sg.GET("/getRoomInfo", authMiddleware, ginx.WrapClaimsAndReq(h.GetRoomInfo))
		sg.GET("/getPrice", authMiddleware, ginx.WrapClaimsAndReq(h.GetPrice))

		sg.PUT("/setStandard", authMiddleware, ginx.WrapClaimsAndReq(h.SetStandard))
		sg.GET("/getStandardList", authMiddleware, ginx.WrapClaimsAndReq(h.GetStandardList))
		sg.DELETE("/cancelStandard", authMiddleware, ginx.WrapClaimsAndReq(h.CancelStandard))
	}
}

// GetArchitecture
// @Summary 获取楼栋信息
// @Description 通过区域获取楼栋信息
// @Tags elecprice
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request query GetArchitectureRequest true "设置电费提醒请求参数"
// @Success 200 {object} web.Response{msg=elecprice.GetArchitectureResponse} "设置成功的返回信息"
// @Failure 500 {object} web.Response{msg=string} "系统异常"
// @Router /elecprice/getArchitecture [get]
func (h *ElecPriceHandler) GetArchitecture(ctx *gin.Context, req GetArchitectureRequest, uc ijwt.UserClaims) (web.Response, error) {
	res, err := h.ElecPriceClient.GetArchitecture(ctx, &elecpricev1.GetArchitectureRequest{
		AreaName: req.AreaName,
	})
	if err != nil {
		return web.Response{}, errs.BAD_ENTITY_ERROR(err)
	}
	var architectureList []*Architecture
	for _, r := range res.ArchitectureList {
		architectureList = append(architectureList, &Architecture{
			ArchitectureName: r.ArchitectureName,
			ArchitectureID:   r.ArchitectureID,
			BaseFloor:        r.BaseFloor,
			TopFloor:         r.TopFloor,
		})
	}

	sort.Slice(architectureList, func(i, j int) bool {
		idI, errI := strconv.Atoi(architectureList[i].ArchitectureID)
		idJ, errJ := strconv.Atoi(architectureList[j].ArchitectureID)

		if errI == nil && errJ == nil {
			return idI < idJ
		}
		return architectureList[i].ArchitectureID < architectureList[j].ArchitectureID
	})

	return web.Response{
		Data: GetArchitectureResponse{
			ArchitectureList: architectureList,
		},
	}, nil
}

// GetRoomInfo
// @Summary 获取房间号和id
// @Description 根据房间号和空调/照明id
// @Tags elecprice
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request query GetRoomInfoRequest true "获取楼栋信息请求参数"
// @Success 200 {object} web.Response{msg=elecprice.GetRoomInfoResponse} "获取成功的返回信息"
// @Failure 500 {object} web.Response{msg=string} "系统异常"
// @Router /elecprice/getRoomInfo [get]
func (h *ElecPriceHandler) GetRoomInfo(ctx *gin.Context, req GetRoomInfoRequest, uc ijwt.UserClaims) (web.Response, error) {
	res, err := h.ElecPriceClient.GetRoomInfo(ctx, &elecpricev1.GetRoomInfoRequest{
		ArchitectureID: req.ArchitectureID,
		Floor:          req.Floor,
	})
	if err != nil {
		return web.Response{}, errs.ELECPRICE_SET_STANDARD_ERROR(err)
	}
	var roomList []*Room
	for _, r := range res.RoomList {
		roomList = append(roomList, &Room{
			RoomID:   r.RoomID,
			RoomName: r.RoomName,
		})
	}

	sort.Slice(roomList, func(i, j int) bool {
		idI, errI := strconv.Atoi(roomList[i].RoomID)
		idJ, errJ := strconv.Atoi(roomList[j].RoomID)

		if errI == nil && errJ == nil {
			return idI < idJ
		}
		return roomList[i].RoomID < roomList[j].RoomID
	})

	return web.Response{
		Data: GetRoomInfoResponse{
			RoomList: roomList,
		},
	}, nil
}

// GetPrice
// @Summary 获取电费
// @Description 根据房间号获取电费信息
// @Tags elecprice
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request query GetPriceRequest true "获取电费请求参数"
// @Success 200 {object} web.Response{msg=elecprice.GetPriceResponse} "获取成功的返回信息"
// @Failure 500 {object} web.Response{msg=string} "系统异常"
// @Router /elecprice/getPrice [get]
func (h *ElecPriceHandler) GetPrice(ctx *gin.Context, req GetPriceRequest, uc ijwt.UserClaims) (web.Response, error) {
	res, err := h.ElecPriceClient.GetPrice(ctx, &elecpricev1.GetPriceRequest{
		RoomId: req.RoomId,
	})
	if err != nil {
		return web.Response{}, errs.ELECPRICE_SET_STANDARD_ERROR(err)
	}
	return web.Response{
		Data: GetPriceResponse{
			Price: &Price{
				RemainMoney:       res.Price.RemainMoney,
				YesterdayUseValue: res.Price.YesterdayUseValue,
				YesterdayUseMoney: res.Price.YesterdayUseMoney,
			},
		},
	}, nil
}

// SetStandard 设置电费
// @Summary 设置电费提醒标准
// @Description 根据区域、楼栋和房间号设置电费提醒的金额标准
// @Tags elecprice
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body SetStandardRequest true "设置电费提醒请求参数"
// @Success 200 {object} web.Response{msg=string} "设置成功的返回信息"
// @Failure 500 {object} web.Response{msg=string} "系统异常"
// @Router /elecprice/setStandard [put]
func (h *ElecPriceHandler) SetStandard(ctx *gin.Context, req SetStandardRequest, uc ijwt.UserClaims) (web.Response, error) {
	_, err := h.ElecPriceClient.SetStandard(ctx, &elecpricev1.SetStandardRequest{
		StudentId: uc.StudentId,
		Standard: &elecpricev1.Standard{
			Limit:    req.Limit,
			RoomName: req.RoomName,
			RoomId:   req.RoomId,
		},
	})
	if err != nil {
		return web.Response{}, errs.ELECPRICE_SET_STANDARD_ERROR(err)
	}

	return web.Response{
		Msg: "设置电费提醒标准成功!",
	}, nil
}

// GetStandardList
// @Summary 获取电费提醒标准
// @Description 获取自己订阅的电费提醒标准
// @Tags elecprice
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} web.Response{msg=elecprice.GetStandardListResponse} "获取成功的返回信息"
// @Failure 500 {object} web.Response{msg=string} "系统异常"
// @Router /elecprice/getStandardList [get]
func (h *ElecPriceHandler) GetStandardList(ctx *gin.Context, req GetStandardListRequest, uc ijwt.UserClaims) (web.Response, error) {
	res, err := h.ElecPriceClient.GetStandardList(ctx, &elecpricev1.GetStandardListRequest{
		StudentId: uc.StudentId,
	})
	if err != nil {
		return web.Response{}, errs.ELECPRICE_GET_STANDARD_LIST_ERROR(err)
	}

	var standards []*StandardResp
	for _, s := range res.Standards {
		standards = append(standards, &StandardResp{
			Limit:    s.Limit,
			RoomName: s.RoomName,
		})
	}

	return web.Response{
		Data: GetStandardListResponse{
			StandardList: standards,
		},
	}, nil
}

// CancelStandard
// @Summary 取消电费提醒标准
// @Description 取消自己订阅的电费提醒
// @Tags elecprice
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body CancelStandardRequest true "取消电费提醒请求参数"
// @Success 200 {object} web.Response{msg=string} "取消成功的返回信息"
// @Failure 500 {object} web.Response{msg=string} "系统异常"
// @Router /elecprice/cancelStandard [delete]
func (h *ElecPriceHandler) CancelStandard(ctx *gin.Context, req CancelStandardRequest, uc ijwt.UserClaims) (web.Response, error) {
	_, err := h.ElecPriceClient.CancelStandard(ctx, &elecpricev1.CancelStandardRequest{
		StudentId: uc.StudentId,
		RoomId:    req.RoomId,
	})
	if err != nil {
		return web.Response{}, errs.ELECPRICE_CANCEL_STANDARD_ERROR(err)
	}

	return web.Response{
		Msg: "取消电费提醒标准成功!",
	}, nil
}
