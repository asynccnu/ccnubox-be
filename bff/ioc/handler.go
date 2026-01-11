package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/bff/pkg/htmlx"
	"github.com/asynccnu/ccnubox-be/bff/web/banner"
	"github.com/asynccnu/ccnubox-be/bff/web/calendar"
	"github.com/asynccnu/ccnubox-be/bff/web/card"
	"github.com/asynccnu/ccnubox-be/bff/web/class"
	"github.com/asynccnu/ccnubox-be/bff/web/classroom"
	"github.com/asynccnu/ccnubox-be/bff/web/department"
	"github.com/asynccnu/ccnubox-be/bff/web/elecprice"
	"github.com/asynccnu/ccnubox-be/bff/web/feed"
	"github.com/asynccnu/ccnubox-be/bff/web/feedback_help"
	"github.com/asynccnu/ccnubox-be/bff/web/grade"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/asynccnu/ccnubox-be/bff/web/infoSum"
	"github.com/asynccnu/ccnubox-be/bff/web/library"
	"github.com/asynccnu/ccnubox-be/bff/web/metrics"
	"github.com/asynccnu/ccnubox-be/bff/web/static"
	"github.com/asynccnu/ccnubox-be/bff/web/swag"
	"github.com/asynccnu/ccnubox-be/bff/web/tube"
	"github.com/asynccnu/ccnubox-be/bff/web/user"
	"github.com/asynccnu/ccnubox-be/bff/web/website"
	bannerv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/banner/v1"
	calendarv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/calendar/v1"
	cardv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/card/v1"
	cs "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classService/v1"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	departmentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/department/v1"
	elecpricev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/elecprice/v1"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	feedbackv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feedback_help/v1"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	infoSumv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/infoSum/v1"
	libraryv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	staticv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/static/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	websitev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/website/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/prometheusx"
	"github.com/ecodeclub/ekit/slice"
	"github.com/qiniu/api.v7/v7/auth/qbox"
	"github.com/redis/go-redis/v9"
)

func InitStaticHandler(
	cfg *conf.TransConf,
	staticClient staticv1.StaticServiceClient) *static.StaticHandler {
	return static.NewStaticHandler(staticClient,
		map[string]htmlx.FileToHTMLConverter{},
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

// InitCalendarHandler 初始化 CalendarHandler
func InitCalendarHandler(
	cfg *conf.TransConf,
	calendarClient calendarv1.CalendarServiceClient) *calendar.CalendarHandler {
	return calendar.NewCalendarHandler(calendarClient,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

// InitBannerHandler 初始化 BannerHandler
func InitBannerHandler(
	cfg *conf.TransConf,
	bannerClient bannerv1.BannerServiceClient, userClient userv1.UserServiceClient) *banner.BannerHandler {
	return banner.NewBannerHandler(bannerClient, userClient,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

// InitWebsiteHandler 初始化 WebsiteHandler
func InitWebsiteHandler(
	cfg *conf.TransConf,
	websiteClient websitev1.WebsiteServiceClient) *website.WebsiteHandler {
	return website.NewWebsiteHandler(websiteClient,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

// InitInfoSumHandler 初始化 InfoSumHandler
func InitInfoSumHandler(
	cfg *conf.TransConf,
	infoSumClient infoSumv1.InfoSumServiceClient) *infoSum.InfoSumHandler {
	return infoSum.NewInfoSumHandler(infoSumClient,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

// InitDepartmentHandler 初始化 DepartmentHandler
func InitDepartmentHandler(
	cfg *conf.TransConf,
	departmentClient departmentv1.DepartmentServiceClient) *department.DepartmentHandler {
	return department.NewDepartmentHandler(departmentClient,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

func InitFeedHandler(
	cfg *conf.TransConf,
	feedServiceClient feedv1.FeedServiceClient) *feed.FeedHandler {
	return feed.NewFeedHandler(feedServiceClient,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

func InitElecpriceHandler(cfg *conf.TransConf, client elecpricev1.ElecpriceServiceClient) *elecprice.ElecPriceHandler {
	return elecprice.NewElecPriceHandler(client,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

func InitClassHandler(cfg *conf.TransConf, client1 classlistv1.ClasserClient, client2 cs.ClassServiceClient) *class.ClassHandler {
	return class.NewClassListHandler(client1, client2,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

func InitClassRoomHandler(client cs.FreeClassroomSvcClient) *classroom.ClassRoomHandler {
	return classroom.NewClassRoomHandler(client)
}
func InitGradeHandler(cfg *conf.TransConf, l logger.Logger, gradeClient gradev1.GradeServiceClient, counterServiceClient counterv1.CounterServiceClient) *grade.GradeHandler {
	return grade.NewGradeHandler(
		gradeClient,
		counterServiceClient,
		l,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) { return element, struct{}{} }),
	)
}

func InitFeedbackHelpHandler(cfg *conf.TransConf, client feedbackv1.FeedbackHelpClient) *feedback_help.FeedbackHelpHandler {
	return feedback_help.NewFeedbackHelpHandler(client,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) { return element, struct{}{} }))
}

func InitCardHandler(cfg *conf.TransConf, client cardv1.CardClient) *card.CardHandler {
	return card.NewCardHandler(client,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) { return element, struct{}{} }))
}

func InitUserHandler(cfg *conf.TransConf, hdl ijwt.Handler, userClient userv1.UserServiceClient) *user.UserHandler {
	return user.NewUserHandler(hdl, userClient)
}

func InitLibraryHandler(cfg *conf.TransConf, client libraryv1.LibraryClient) *library.LibraryHandler {
	return library.NewLibraryHandler(client,
		slice.ToMapV(cfg.Administrators, func(element string) (string, struct{}) { return element, struct{}{} }))
}

func InitTubeHandler(cfg *conf.TransConf, tb *TubePolicies, mac *qbox.Mac) *tube.TubeHandler {
	return tube.NewTubeHandler(tb.defaultPolicy, tb.officialSite, mac, cfg.Oss.DomainName)
}

func InitMetricsHandel(l logger.Logger, redisClient redis.Cmdable, prometheus *prometheusx.PrometheusCounter) *metrics.MetricsHandler {
	return metrics.NewMetricsHandler(l, redisClient, prometheus)
}

func InitSwagHandler() *swag.SwagHandler {
	return swag.NewSwagHandler()
}
