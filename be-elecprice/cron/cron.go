package cron

import "github.com/asynccnu/ccnubox-be/be-elecprice/service"

type Cron interface {
	StartCronTask()
}

// autoService服务还需要进行一个对表格的清理,如果学号已经超过毕业时间2年应当被自动清理

func NewCron(
	elecpriceController *ElecpriceController,
	proxyGetter service.ProxyGetter,
) []Cron {
	return []Cron{elecpriceController, proxyGetter}
}
