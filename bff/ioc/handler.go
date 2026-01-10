package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/web/class"
	"github.com/asynccnu/ccnubox-be/bff/web/classroom"
	"github.com/asynccnu/ccnubox-be/bff/web/content"
	"github.com/asynccnu/ccnubox-be/bff/web/elecprice"
	"github.com/asynccnu/ccnubox-be/bff/web/feed"
	"github.com/asynccnu/ccnubox-be/bff/web/grade"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/asynccnu/ccnubox-be/bff/web/library"
	"github.com/asynccnu/ccnubox-be/bff/web/metrics"
	"github.com/asynccnu/ccnubox-be/bff/web/swag"
	"github.com/asynccnu/ccnubox-be/bff/web/tube"
	"github.com/asynccnu/ccnubox-be/bff/web/user"
	cs "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classService/v1"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	elecpricev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/elecprice/v1"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	libraryv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/prometheusx"
	"github.com/ecodeclub/ekit/slice"
	"github.com/qiniu/api.v7/v7/auth/qbox"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// InitContentHandler 初始化 ContentHandler
func InitContentHandler(
	contentClient contentv1.ContentServiceClient,
	userClient userv1.UserServiceClient,
) *content.ContentHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return content.NewContentHandler(
		contentClient,
		userClient,
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

func InitUserHandler(hdl ijwt.Handler, userClient userv1.UserServiceClient, gradeClient gradev1.GradeServiceClient,
	classListClient classlistv1.ClasserClient) *user.UserHandler {
	var administrators []string
	err := viper.UnmarshalKey("administrators", &administrators)
	if err != nil {
		panic(err)
	}
	return user.NewUserHandler(hdl, userClient, gradeClient, classListClient)
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
