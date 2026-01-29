package grpc

import (
	"github.com/asynccnu/ccnubox-be/be-content/service"                        // 替换为calendar的服务路径
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1" // 替换为calendar的proto包路径
	"google.golang.org/grpc"
)

type ContentServiceServer struct {
	contentv1.UnimplementedContentServiceServer
	svcCalendar   service.CalendarService
	svcBanner     service.BannerService
	svcDepartment service.DepartmentService
	svcWebsite    service.WebsiteService
	svcInfoSum    service.InfoSumService
}

func NewCalendarServiceServer(
	svcCalendar service.CalendarService,
	svcBanner service.BannerService,
	svcDepartment service.DepartmentService,
	svcWebsite service.WebsiteService,
	svcInfoSum service.InfoSumService,
) *ContentServiceServer {
	return &ContentServiceServer{
		svcCalendar:   svcCalendar,
		svcBanner:     svcBanner,
		svcDepartment: svcDepartment,
		svcWebsite:    svcWebsite,
		svcInfoSum:    svcInfoSum,
	}
}

// 注册为grpc服务
func (c *ContentServiceServer) Register(server grpc.ServiceRegistrar) {
	contentv1.RegisterContentServiceServer(server, c)
}
