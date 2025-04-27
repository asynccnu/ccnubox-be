package feedback_help

import (
	"fmt"
	feedback_helpv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feedback_help/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
)

// TODO 删除反馈,改为使用木犀反馈中台服务
type FeedbackHelpHandler struct {
	FeedbackHelpClient feedback_helpv1.FeedbackHelpClient //注入的是grpc服务
	Administrators     map[string]struct{}                //这里注入的是管理员权限验证配置
}

func NewFeedbackHelpHandler(FeedbackHelpClient feedback_helpv1.FeedbackHelpClient,
	administrators map[string]struct{}) *FeedbackHelpHandler {
	return &FeedbackHelpHandler{FeedbackHelpClient: FeedbackHelpClient, Administrators: administrators}
}

func (h *FeedbackHelpHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/feedback_help")
	sg.GET("/getQuestion", authMiddleware, ginx.Wrap(h.GetQuestions))
	sg.POST("/createQuestion", authMiddleware, ginx.WrapClaimsAndReq(h.CreateQuestion))
	sg.POST("/changeQuestion", authMiddleware, ginx.WrapClaimsAndReq(h.ChangeQuestion))
	sg.POST("/deleteQuestion", authMiddleware, ginx.WrapClaimsAndReq(h.DeleteQuestion))
	sg.GET("/findQuestionsByName", authMiddleware, ginx.WrapReq(h.FindQuestionsByName))
	sg.POST("/noteQuestion", authMiddleware, ginx.WrapReq(h.NoteQuestion))
}

// @Summary 获取常见问题
// @Description 获取点击数量最多的10个常见问题
// @Tags 帮助与反馈【弃用】
// @Accept  json
// @Produce  json
// @Success 200 {object} web.Response{data=GetQuestionsResp} "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feedback_help/getQuestion [get]
func (h *FeedbackHelpHandler) GetQuestions(c *gin.Context) (web.Response, error) {
	q, err := h.FeedbackHelpClient.GetQuestions(c, &feedback_helpv1.EmptyRequest{})

	if err != nil {
		return web.Response{}, errs.GET_QUESTION_ERROR(err)
	}

	var resp GetQuestionsResp
	for _, question := range q.Questions {
		resp.Questions = append(resp.Questions, FrequentlyAskedQuestion{
			Id:         question.Id,
			Question:   question.Question,
			Answer:     question.Answer,
			ClickTimes: question.ClickTimes,
		})
	}

	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil

}

// @Summary 创建一个问题与答复
// @Description 创建一个常见问题的内容与答复
// @Tags 帮助与反馈【弃用】
// @Accept  json
// @Produce  json
// @Param data body CreateQuestionReq true "创建一个常见问题"
// @Success 200 {object} web.Response "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feedback_help/createQuestion [post]
func (h *FeedbackHelpHandler) CreateQuestion(c *gin.Context, req CreateQuestionReq, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.FeedbackHelpClient.CreateQuestion(c, &feedback_helpv1.CreateQuestionRequest{
		Question: req.Question,
		Anwser:   req.Answer,
	})

	if err != nil {
		return web.Response{}, errs.CREATE_QUESTION_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// @Summary 修改一个问题与答复
// @Description 修改一个常见问题的内容与答复
// @Tags 帮助与反馈【弃用】
// @Accept  json
// @Produce  json
// @Param data body ChangeQuestionReq true "修改常见问题"
// @Success 200 {object} web.Response "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feedback_help/changeQuestion [post]
func (h *FeedbackHelpHandler) ChangeQuestion(c *gin.Context, req ChangeQuestionReq, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.FeedbackHelpClient.ChangeQuestion(c, &feedback_helpv1.UpdateQuestionRequest{
		QuestionId: req.QuestionId,
		Question:   req.Question,
		Anwser:     req.Answer,
	})

	if err != nil {
		return web.Response{}, errs.CHANGE_QUESTION_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// @Summary 删除一个问题与答复
// @Description 删除一个常见问题的内容与答复
// @Tags 帮助与反馈【弃用】
// @Accept  json
// @Produce  json
// @Param data body DeleteQuestionReq true "删除常见问题"
// @Success 200 {object} web.Response "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feedback_help/deleteQuestion [post]
func (h *FeedbackHelpHandler) DeleteQuestion(c *gin.Context, req DeleteQuestionReq, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.FeedbackHelpClient.DeleteQuestion(c, &feedback_helpv1.DeleteQuestionRequest{
		QuestionId: req.QuestionId,
	})

	if err != nil {
		return web.Response{}, errs.DELETE_QUESTION_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// @Summary 搜取问题
// @Description 对常见问题进行模糊搜索
// @Tags 帮助与反馈【弃用】
// @Accept  json
// @Produce  json
// @Param question query string true "问题名称"
// @Success 200 {object} web.Response{date=FindQuestionsByNameResp} "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feedback_help/findQuestionsByName [get]
func (h *FeedbackHelpHandler) FindQuestionsByName(c *gin.Context, req FindQuestionsByNameReq) (web.Response, error) {
	q, err := h.FeedbackHelpClient.FindQuestionByName(c, &feedback_helpv1.FindQuestionByNameRequest{
		Question: req.Question,
	})

	if err != nil {
		return web.Response{
			Code: errs.INTERNAL_SERVER_ERROR_CODE,
			Msg:  "获取失败",
		}, errs.FIND_QUESTIONS_BY_NAME_ERROR(err)
	}

	var resp FindQuestionsByNameResp
	for _, question := range q.Questions {
		resp.Questions = append(resp.Questions, FrequentlyAskedQuestion{
			Id:         question.Id,
			Question:   question.Question,
			Answer:     question.Answer,
			ClickTimes: question.ClickTimes,
		})
	}

	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}

// @Summary 标记问题解决状态
// @Description 标记问题解决状态
// @Tags 帮助与反馈【弃用】
// @Accept  json
// @Produce  json
// @Param data body NoteQuestionReq true "标记问题解决状态"
// @Success 200 {object} web.Response{} "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feedback_help/noteQuestion [post]
func (h *FeedbackHelpHandler) NoteQuestion(c *gin.Context, req NoteQuestionReq) (web.Response, error) {
	_, err := h.FeedbackHelpClient.NoteQuestion(c, &feedback_helpv1.NoteQuestionRequest{
		QuestionId: req.QuestionId,
		IfOver:     req.IfOver,
	})

	if err != nil {
		return web.Response{}, errs.NOTE_QUESTION_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}
func (h *FeedbackHelpHandler) isAdmin(studentId string) bool {
	_, exists := h.Administrators[studentId]
	return exists
}
