package service

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/usecase"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/pkg/tool"
	pb "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1" // 此处改成了api中的,方便其他服务调用.
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	ctool "github.com/asynccnu/ccnubox-be/common/tool"
)

type ClassListService struct {
	clu  *usecase.ClassUsecase
	conf *conf.ServerConf
}

func NewClasserService(clu *usecase.ClassUsecase, conf *conf.ServerConf) *ClassListService {
	return &ClassListService{
		clu:  clu,
		conf: conf,
	}
}

func (s *ClassListService) GetClass(ctx context.Context, req *pb.GetClassRequest) (*pb.GetClassResponse, error) {
	hlog := logger.From(ctx)
	hlog = hlog.With(
		logger.String("stu_id", req.GetStuId()),
		logger.String("year", req.GetYear()),
		logger.String("semester", req.GetSemester()),
	)
	ctx = logger.WithLogger(ctx, hlog)

	defaultYear, defaultSemester := ctool.GetCurrentAcademicYearAndSemesterStr(time.Now())

	if req.GetYear() == "" {
		req.Year = defaultYear
		hlog.Warn(fmt.Sprintf("获取 Year 参数为空，使用默认值 %s", req.Year))
	}

	if req.GetSemester() == "" {
		req.Semester = defaultSemester
		hlog.Warn(fmt.Sprintf("获取 Semester 参数为空，使用默认值 %s", req.Semester))
	}

	if !tool.CheckSY(req.Semester, req.Year) {
		return &pb.GetClassResponse{}, errcode.ErrParam
	}

	classInfos, lastTime, err := s.clu.GetClasses(ctx, req.StuId, req.Year, req.Semester, req.Refresh)
	if err != nil {
		return &pb.GetClassResponse{}, err
	}

	pbClassInfos := make([]*pb.Class, 0, len(classInfos))
	for _, classInfo := range classInfos {
		pbClassInfo := classInfoBOToPb(classInfo)
		pbClassInfos = append(pbClassInfos, &pb.Class{
			Info: pbClassInfo,
		})
	}

	var lastTimeStamp int64

	if lastTime != nil {
		lastTimeStamp = convertToShanghaiTimeStamp(*lastTime)
	} else {
		lastTimeStamp = time.Date(1949, 10, 1, 0, 0, 0, 0, time.Local).Unix()
	}
	return &pb.GetClassResponse{
		Classes:  pbClassInfos,
		LastTime: lastTimeStamp,
	}, nil
}

func convertToShanghaiTimeStamp(t time.Time) int64 {
	return tool.ToShanghaiTime(t).Unix()
}
