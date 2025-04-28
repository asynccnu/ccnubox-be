package card

import (
	cardv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/card/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/ptypes"
)

type CardHandler struct {
	CardClient     cardv1.CardClient
	Administrators map[string]struct{} //这里注入的是管理员权限验证配置
}

func NewCardHandler(CardClient cardv1.CardClient,
	administrators map[string]struct{}) *CardHandler {
	return &CardHandler{CardClient: CardClient, Administrators: administrators}
}

func (h *CardHandler) RegisterRoute(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/card")
	sg.POST("/noteUserKey", authMiddleware, ginx.WrapClaimsAndReq(h.NoteUserKey))
	sg.POST("/updateUserKey", authMiddleware, ginx.WrapClaimsAndReq(h.UpdateUserKey))
	sg.POST("/getRecords", authMiddleware, ginx.WrapClaimsAndReq(h.GetRecords))
}

// NoteUserKey
// @Summary 记录用户的key
// @Description 【弃用】记录用户的key
// @Tags card[Deprecation]
// @Accept json
// @Produce json
// @Param data body NoteUserKeyRequest true "记录用户的key"
// @Success 200 {object} web.Response "成功"
// @Failure 500 {object} web.Response "创建失败"
// @Router /card/noteUserKey [post]
func (h *CardHandler) NoteUserKey(c *gin.Context, req NoteUserKeyRequest, uc ijwt.UserClaims) (web.Response, error) {
	_, err := h.CardClient.CreateUser(c, &cardv1.CreateUserRequest{
		StudentId: uc.StudentId,
		Key:       req.Key,
	})
	if err != nil {
		return web.Response{}, errs.NOTE_USER_KEY_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// UpdateUserKey
// @Summary 更新用户的key
// @Description 【弃用】更新用户的key
// @Tags card[Deprecation]
// @Accept json
// @Produce json
// @Param data body UpdateUserKeyRequest true "更新用户的key"
// @Success 200 {object} web.Response "成功"
// @Failure 500 {object} web.Response "创建失败"
// @Router /card/updateUserKey [post]
func (h *CardHandler) UpdateUserKey(c *gin.Context, req UpdateUserKeyRequest, uc ijwt.UserClaims) (web.Response, error) {
	_, err := h.CardClient.UpdateUserKey(c, &cardv1.UpdateUserKeyRequest{
		StudentId: uc.StudentId,
		Key:       req.Key,
	})
	if err != nil {
		return web.Response{}, errs.UPDATE_USER_KEY_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// GetRecords
// @Summary 获取消费记录
// @Description 【弃用】获取用户消费记录，student_id, start_time, type 必须存在，type 分为 "card"（实体卡）与 "virtual"（虚拟卡）
// @Tags card[Deprecation]
// @Accept json
// @Produce json
// @Param data body GetRecordOfConsumptionRequest true "获取消费记录"
// @Success 200 {object} web.Response{data=GetRecordOfConsumptionResponse} "成功"
// @Failure 500 {object} web.Response "创建失败"
// @Router /card/getRecords [post]
func (h *CardHandler) GetRecords(c *gin.Context, req GetRecordOfConsumptionRequest, uc ijwt.UserClaims) (web.Response, error) {
	resp, err := h.CardClient.GetRecordOfConsumption(c, &cardv1.GetRecordOfConsumptionRequest{
		StudentId: uc.StudentId,
		Key:       req.Key,
		StartTime: req.StartTime,
		Type:      req.Type,
	})
	if err != nil {
		return web.Response{}, errs.GET_RECORDS_ERROR(err)
	}

	var response GetRecordsResp
	for _, record := range resp.Records {
		time, _ := ptypes.Timestamp(record.SMT_DEALDATETIME)
		response.Records = append(response.Records, Records{
			SMT_TIMES:        record.SMT_TIMES,
			SMT_DEALDATETIME: time,
			SMT_ORG_NAME:     record.SMT_ORG_NAME,
			SMT_DEALNAME:     record.SMT_DEALNAME,
			AfterMoney:       record.AfterMoney,
			Money:            record.Money,
		})
	}
	return web.Response{
		Msg:  "Success",
		Data: response,
	}, nil
}
