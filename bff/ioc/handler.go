package ioc

import (
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
	bannerv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/banner/v1"
	calendarv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/calendar/v1"
	cardv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/card/v1"
	cs "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/classService/v1"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/classlist/v1"
	counterv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/counter/v1"
	departmentv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/department/v1"
	elecpricev1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/elecprice/v1"
	feedv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/feed/v1"
	feedbackv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/feedback_help/v1"
	gradev1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/grade/v1"
	infoSumv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/infoSum/v1"
	libraryv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/library/v1"
	staticv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/static/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/user/v1"
	websitev1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/website/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/htmlx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/prometheusx"
	"github.com/ecodeclub/ekit/slice"
	"github.com/qiniu/api.v7/v7/auth/qbox"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitStaticHandler(
	staticClient staticv1.StaticServiceClient) *static.StaticHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return static.NewStaticHandler(staticClient,
		map[string]htmlx.FileToHTMLConverter{},
		slice.ToMapV(administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

// InitCalendarHandler 初始化 CalendarHandler
func InitCalendarHandler(
	calendarClient calendarv1.CalendarServiceClient) *calendar.CalendarHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return calendar.NewCalendarHandler(calendarClient,
		slice.ToMapV(administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

// InitBannerHandler 初始化 BannerHandler
func InitBannerHandler(
	bannerClient bannerv1.BannerServiceClient, userClient userv1.UserServiceClient) *banner.BannerHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return banner.NewBannerHandler(bannerClient, userClient,
		slice.ToMapV(administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

// InitWebsiteHandler 初始化 WebsiteHandler
func InitWebsiteHandler(
	websiteClient websitev1.WebsiteServiceClient) *website.WebsiteHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return website.NewWebsiteHandler(websiteClient,
		slice.ToMapV(administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

// InitInfoSumHandler 初始化 InfoSumHandler
func InitInfoSumHandler(
	infoSumClient infoSumv1.InfoSumServiceClient) *infoSum.InfoSumHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return infoSum.NewInfoSumHandler(infoSumClient,
		slice.ToMapV(administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

// InitDepartmentHandler 初始化 DepartmentHandler
func InitDepartmentHandler(
	departmentClient departmentv1.DepartmentServiceClient) *department.DepartmentHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return department.NewDepartmentHandler(departmentClient,
		slice.ToMapV(administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

func InitFeedHandler(
	feedServiceClient feedv1.FeedServiceClient) *feed.FeedHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return feed.NewFeedHandler(feedServiceClient,
		slice.ToMapV(administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

func InitElecpriceHandler(client elecpricev1.ElecpriceServiceClient) *elecprice.ElecPriceHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}

	return elecprice.NewElecPriceHandler(client,
		slice.ToMapV(administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

func InitClassHandler(client1 classlistv1.ClasserClient, client2 cs.ClassServiceClient) *class.ClassHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return class.NewClassListHandler(client1, client2,
		slice.ToMapV(administrators, func(element string) (string, struct{}) {
			return element, struct{}{}
		}))
}

func InitClassRoomHandler(client cs.FreeClassroomSvcClient) *classroom.ClassRoomHandler {
	return classroom.NewClassRoomHandler(client)
}
func InitGradeHandler(l logger.Logger, gradeClient gradev1.GradeServiceClient, counterServiceClient counterv1.CounterServiceClient) *grade.GradeHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return grade.NewGradeHandler(
		gradeClient,
		counterServiceClient,
		l,
		slice.ToMapV(administrators, func(element string) (string, struct{}) { return element, struct{}{} }),
	)
}

func InitFeedbackHelpHandler(client feedbackv1.FeedbackHelpClient) *feedback_help.FeedbackHelpHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return feedback_help.NewFeedbackHelpHandler(client,
		slice.ToMapV(administrators, func(element string) (string, struct{}) { return element, struct{}{} }))
}

func InitCardHandler(client cardv1.CardClient) *card.CardHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return card.NewCardHandler(client,
		slice.ToMapV(administrators, func(element string) (string, struct{}) { return element, struct{}{} }))
}

func InitUserHandler(hdl ijwt.Handler, userClient userv1.UserServiceClient) *user.UserHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return user.NewUserHandler(hdl, userClient)
}

func InitLibraryHandler(client libraryv1.LibraryClient) *library.LibraryHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return library.NewLibraryHandler(client,
		slice.ToMapV(administrators, func(element string) (string, struct{}) { return element, struct{}{} }))
}

func InitTubeHandler(tb *TubePolicies, mac *qbox.Mac) *tube.TubeHandler {
	return tube.NewTubeHandler(tb.defaultPolicy, tb.officialSite, mac, viper.GetString("oss.domainName"))
}

func InitMetricsHandel(l logger.Logger, redisClient redis.Cmdable, prometheus *prometheusx.PrometheusCounter) *metrics.MetricsHandler {
	return metrics.NewMetricsHandler(l, redisClient, prometheus)
}

func InitSwagHandler() *swag.SwagHandler {
	return swag.NewSwagHandler()
}
