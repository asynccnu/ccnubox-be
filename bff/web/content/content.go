package content

import (
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/gin-gonic/gin"
)

// ContentHandler 处理与 content 相关的 API 请求
type ContentHandler struct {
	contentClient  contentv1.ContentServiceClient
	userClient     userv1.UserServiceClient
	gradeClient    gradev1.GradeServiceClient
	Administrators map[string]struct{}
}

// NewContentHandler 创建一个新的 ContentHandler 实例
func NewContentHandler(contentClient contentv1.ContentServiceClient,
	userClient userv1.UserServiceClient,
	gradeClient gradev1.GradeServiceClient,
	administrators map[string]struct{},
) *ContentHandler {
	return &ContentHandler{contentClient: contentClient, gradeClient: gradeClient, userClient: userClient, Administrators: administrators}
}

// RegisterRoutes 注册与 content 相关的路由
func (h *ContentHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	h.RegisterBannerRoute(s, authMiddleware)
	h.RegisterCalendarRoute(s, authMiddleware)
	h.RegisterDepartmentRoute(s, authMiddleware)
	h.RegisterInfoSumRoute(s, authMiddleware)
	h.RegisterWebsiteRoute(s, authMiddleware)
}

func (h *ContentHandler) isAdmin(studentId string) bool {
	_, exists := h.Administrators[studentId]
	return exists
}
