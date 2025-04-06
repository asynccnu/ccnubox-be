package biz

import (
	"context"
	v1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classlist/v1"
	clog "github.com/asynccnu/ccnubox-be/be-class/internal/log"
	"github.com/asynccnu/ccnubox-be/be-class/internal/model"
)

type EsProxy interface {
	AddClassInfo(ctx context.Context, classInfo ...model.ClassInfo) error
	ClearClassInfo(ctx context.Context, xnm, xqm string)
	SearchClassInfo(ctx context.Context, keyWords string, xnm, xqm string) ([]model.ClassInfo, error)
}

type ClassListService interface {
	GetAllSchoolClassInfos(ctx context.Context, xnm, xqm, cursor string) ([]model.ClassInfo, string, error)
	AddClassInfoToClassListService(ctx context.Context, req *v1.AddClassRequest) (*v1.AddClassResponse, error)
}
type ClassSerivceUserCase struct {
	es EsProxy
	cs ClassListService
}

func NewClassSerivceUserCase(es EsProxy, cs ClassListService) *ClassSerivceUserCase {
	return &ClassSerivceUserCase{
		es: es,
		cs: cs,
	}
}

func (c *ClassSerivceUserCase) AddClassInfoToClassListService(ctx context.Context, request *v1.AddClassRequest) (*v1.AddClassResponse, error) {
	return c.cs.AddClassInfoToClassListService(ctx, request)
}

func (c *ClassSerivceUserCase) SearchClassInfo(ctx context.Context, keyWords string, xnm, xqm string) ([]model.ClassInfo, error) {
	return c.es.SearchClassInfo(ctx, keyWords, xnm, xqm)
}

func (c *ClassSerivceUserCase) AddClassInfosToES(ctx context.Context, xnm, xqm string) {
	//xnm, xqm := tool.GetXnmAndXqm()
	reqTime := "1949-10-01T00:00:00.000000"
	for {
		classInfos, lastTime, err := c.cs.GetAllSchoolClassInfos(ctx, xnm, xqm, reqTime)
		if len(classInfos) == 0 {
			return
		}
		if err != nil {
			clog.LogPrinter.Errorf("failed to get all classlist")
			return
		}
		err1 := c.es.AddClassInfo(ctx, classInfos...)
		if err1 != nil {
			clog.LogPrinter.Errorf("add classlist[%v] failed: %v", classInfos, err)
		}
		clog.LogPrinter.Infof("es has save %d classes", len(classInfos))
		reqTime = lastTime
	}
}
func (c *ClassSerivceUserCase) DeleteSchoolClassInfosFromES(ctx context.Context, xnm, xqm string) {
	//xnm, xqm := tool.GetXnmAndXqm()
	c.es.ClearClassInfo(ctx, xnm, xqm)
}
