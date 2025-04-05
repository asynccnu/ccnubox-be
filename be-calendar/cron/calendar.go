package cron

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-calendar/domain"
	"github.com/asynccnu/ccnubox-be/be-calendar/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-calendar/pkg/pdf"
	"github.com/asynccnu/ccnubox-be/be-calendar/pkg/qiniu"
	"github.com/asynccnu/ccnubox-be/be-calendar/pkg/reptile"
	"github.com/asynccnu/ccnubox-be/be-calendar/service"
	"github.com/spf13/viper"
	"strconv"
	"time"
)

type CalendarController struct {
	calendarService service.CalendarService
	qiniu           qiniu.QiniuClient
	reptile         reptile.Reptile
	stopChan        chan struct{}
	cfg             CalendarControllerConfig
	l               logger.Logger
}

// 配置定时任务周期
type CalendarControllerConfig struct {
	DurationTime int64 `yaml:"durationTime"`
}

func NewCalendarController(
	repo service.CalendarService,
	qiniu qiniu.QiniuClient,
	l logger.Logger,
) *CalendarController {
	var cfg CalendarControllerConfig
	if err := viper.UnmarshalKey("calendarController", &cfg); err != nil {
		panic(err)
	}

	return &CalendarController{
		calendarService: repo,
		qiniu:           qiniu,
		reptile:         reptile.NewReptile(),
		stopChan:        make(chan struct{}),
		cfg:             cfg,
		l:               l,
	}
}

func (r *CalendarController) StartCronTask() {
	go func() {
		ticker := time.NewTicker(time.Duration(r.cfg.DurationTime) * time.Hour)
		for {
			select {
			case <-ticker.C:
				err := r.scrapeAndUpload()
				if err != nil {
					r.l.Error("日历爬取错误:", logger.FormatLog("cron", err)...)
				}
			case <-r.stopChan:
				ticker.Stop()
				return
			}
		}
	}() //定时控制器
}

func (r *CalendarController) scrapeAndUpload() error {
	//由于没有使用注册为路由这里手动写的上下文
	ctx := context.Background()
	//获取华师网站日历信息
	calendarInfos, err := r.reptile.GetCalendarLink()
	if err != nil {
		return err
	}
	for _, calendarInfo := range calendarInfos {
		//转化为int类型
		year, err := strconv.Atoi(calendarInfo.Year)
		if err != nil {
			return err
		}

		//检查是否已经爬取过,如果已经爬取过就直接跳过
		_, err = r.calendarService.GetCalendar(ctx, int64(year))
		if err == nil {
			continue
		}

		//爬取以下页面的pdflink和imageLinks
		calendarInfo.PDFLink, calendarInfo.ImageLinks, err = r.reptile.FetchPDFOrImageLinksFromPage(calendarInfo.Link)
		if err != nil {
			return err
		}

		//如果pdf不为空的话就直接获取并存储
		if calendarInfo.PDFLink != "" {
			pdfBytes, err := pdf.GetBytesFromLink(calendarInfo.PDFLink)
			if err != nil {
				return err
			}
			//上传图片并获取返回的链接
			link, err := r.qiniu.Upload(pdfBytes, calendarInfo.Year)
			if err != nil {
				return err
			}
			//存储到数据库中
			err = r.calendarService.SaveCalendar(ctx, &domain.Calendar{
				Year: int64(year),
				Link: link,
			})
			if err != nil {
				return err
			}
		} else if calendarInfo.ImageLinks != nil {

			//如果获取的是images的话
			pdfBytes, err := pdf.CreatePDFfromImageLinks(calendarInfo.ImageLinks)
			if err != nil {
				return err
			}

			//上传图片并获取返回的链接
			link, err := r.qiniu.Upload(pdfBytes, calendarInfo.Year)
			if err != nil {
				return err
			}

			//存储到数据库中
			err = r.calendarService.SaveCalendar(ctx, &domain.Calendar{
				Year: int64(year),
				Link: link,
			})
			if err != nil {
				return err
			}
		}

	}
	return nil
}
