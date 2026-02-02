package service

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/pkg/tool"
	pb "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1" // 此处改成了api中的,方便其他服务调用.
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/jinzhu/copier"
)

type ClassListService struct {
	pb.UnimplementedClasserServer
	clu       *biz.ClassUsecase
	schoolday *conf.SchoolDay
	logger    logger.Logger
	defaults  *conf.Defaults
}

func NewClasserService(clu *biz.ClassUsecase, day *conf.SchoolDay, logger logger.Logger, defaults *conf.Defaults) *ClassListService {
	return &ClassListService{
		clu:       clu,
		logger:    logger,
		schoolday: day,
		defaults:  defaults,
	}
}

func (s *ClassListService) GetClass(ctx context.Context, req *pb.GetClassRequest) (*pb.GetClassResponse, error) {
	hlog := logger.GlobalLogger.WithContext(ctx)
	hlog = hlog.With(
		logger.String("stu_id", req.GetStuId()),
		logger.String("year", req.GetYear()),
		logger.String("semester", req.GetSemester()),
	)
	ctx = logger.WithLogger(ctx, hlog)

	if s.defaults == nil {
		hlog.Warn("default 参数未在配置文件中配置")
	}

	if req.GetYear() == "" {
		req.Year = s.defaults.Year
		hlog.Warn(fmt.Sprintf("获取 Year 参数为空，使用默认值 %s", req.Year))
	}

	if req.GetSemester() == "" {
		req.Semester = s.defaults.Semester
		hlog.Warn(fmt.Sprintf("获取 Semester 参数为空，使用默认值 %s", req.Semester))
	}

	if !tool.CheckSY(req.Semester, req.Year) {
		return &pb.GetClassResponse{}, errcode.ErrParam
	}
	pclasses := make([]*pb.Class, 0)
	classInfos, lastTime, err := s.clu.GetClasses(ctx, req.GetStuId(), req.Year, req.Semester, req.GetRefresh())
	if err != nil {
		return &pb.GetClassResponse{}, err
	}
	for _, classInfo := range classInfos {
		pinfo := new(pb.ClassInfo)

		_ = copier.Copy(&pinfo, &classInfo)
		// 优先使用 biz 层设置的 IsOfficial
		pinfo.IsOfficial = classInfo.IsOfficial
		// 当 IsOfficial 为 false 时，可能是默认值或手动添加课程，回退到数据库判断以确保本地读取的正确性
		if !pinfo.IsOfficial {
			pinfo.IsOfficial = s.clu.IsClassOfficial(ctx, req.GetStuId(), req.GetYear(), req.GetSemester(), classInfo.ID)
		}

		pclass := &pb.Class{
			Info: pinfo,
		}
		pclasses = append(pclasses, pclass)
	}
	var lastTimeStamp int64

	if lastTime != nil {
		lastTimeStamp = convertToShanghaiTimeStamp(*lastTime)
	} else {
		lastTimeStamp = time.Date(1949, 10, 1, 0, 0, 0, 0, time.Local).Unix()
	}
	return &pb.GetClassResponse{
		Classes:  pclasses,
		LastTime: lastTimeStamp,
	}, nil
}

func (s *ClassListService) AddClass(ctx context.Context, req *pb.AddClassRequest) (*pb.AddClassResponse, error) {
	valLogger := logger.GetLoggerFromCtx(ctx)
	valLogger = valLogger.With(
		logger.String("stu_id", req.GetStuId()),
		logger.String("year", req.GetYear()),
		logger.String("semester", req.GetSemester()),
	)
	ctx = logger.WithLogger(ctx, valLogger)

	if !tool.CheckSY(req.Semester, req.Year) || req.GetWeeks() <= 0 || !tool.CheckIfThisYear(req.Year, req.Semester) {
		return &pb.AddClassResponse{}, errcode.ErrParam
	}
	weekDur := tool.FormatWeeks(tool.ParseWeeks(req.Weeks))
	classInfo := &biz.ClassInfo{
		Day:          req.GetDay(),
		Teacher:      req.GetTeacher(),
		Where:        req.GetWhere(),
		ClassWhen:    req.GetDurClass(),
		WeekDuration: weekDur,
		Classname:    req.GetName(),
		Credit:       req.GetCredit(),
		Weeks:        req.GetWeeks(),
		Semester:     req.GetSemester(),
		Year:         req.GetYear(),
		JxbId:        "unavailable",
	}
	if req.Credit != nil {
		classInfo.Credit = req.GetCredit()
	}

	classInfo.UpdateID()

	err := s.clu.AddClass(ctx, req.GetStuId(), classInfo)
	if err != nil {
		return &pb.AddClassResponse{}, err
	}

	return &pb.AddClassResponse{
		Id:  classInfo.ID,
		Msg: "成功添加",
	}, nil
}

func (s *ClassListService) DeleteClass(ctx context.Context, req *pb.DeleteClassRequest) (*pb.DeleteClassResponse, error) {
	valLogger := logger.GetLoggerFromCtx(ctx)
	valLogger = valLogger.With(
		logger.String("stu_id", req.GetStuId()),
		logger.String("year", req.GetYear()),
		logger.String("semester", req.GetSemester()),
	)
	ctx = logger.WithLogger(ctx, valLogger)

	if !tool.CheckSY(req.Semester, req.Year) {
		return &pb.DeleteClassResponse{}, errcode.ErrParam
	}
	exist := s.clu.CheckSCIdsExist(ctx, req.GetStuId(), req.GetYear(), req.GetSemester(), req.GetId())
	if !exist {
		return &pb.DeleteClassResponse{
			Msg: "该课程不存在",
		}, errcode.ErrSCIDNOTEXIST
	}
	err := s.clu.DeleteClass(ctx, req.GetStuId(), req.GetYear(), req.GetSemester(), req.GetId())
	if err != nil {
		return &pb.DeleteClassResponse{}, err
	}
	return &pb.DeleteClassResponse{
		Msg: "成功删除",
	}, nil
}

func (s *ClassListService) UpdateClass(ctx context.Context, req *pb.UpdateClassRequest) (*pb.UpdateClassResponse, error) {
	valLogger := logger.GetLoggerFromCtx(ctx)
	valLogger = valLogger.With(
		logger.String("stu_id", req.GetStuId()),
		logger.String("year", req.GetYear()),
		logger.String("semester", req.GetSemester()),
	)
	ctx = logger.WithLogger(ctx, valLogger)

	if !tool.CheckSY(req.Semester, req.Year) {
		return &pb.UpdateClassResponse{}, errcode.ErrParam
	}
	exist := s.clu.CheckSCIdsExist(ctx, req.GetStuId(), req.GetYear(), req.GetSemester(), req.GetClassId())
	if !exist {
		return &pb.UpdateClassResponse{
			Msg: "该课程不存在",
		}, errcode.ErrSCIDNOTEXIST
	}
	if !tool.CheckSY(req.Semester, req.GetYear()) {
		return &pb.UpdateClassResponse{}, errcode.ErrParam
	}

	oldclassInfo, err := s.clu.SearchClass(ctx, req.GetClassId())
	if err != nil {
		return &pb.UpdateClassResponse{
			Msg: "修改失败",
		}, err
	}
	if req.Day != nil {
		oldclassInfo.Day = req.GetDay()
	}
	if req.Teacher != nil {
		oldclassInfo.Teacher = req.GetTeacher()
	}
	if req.Where != nil {
		oldclassInfo.Where = req.GetWhere()
	}
	if req.DurClass != nil {
		oldclassInfo.ClassWhen = req.GetDurClass()
	}
	if req.Name != nil {
		oldclassInfo.Classname = req.GetName()
	}
	if req.Weeks != nil {
		oldclassInfo.Weeks = req.GetWeeks()
		weekDur := tool.FormatWeeks(tool.ParseWeeks(req.GetWeeks()))
		oldclassInfo.WeekDuration = weekDur
	}
	if req.Credit != nil {
		oldclassInfo.Credit = req.GetCredit()
	}

	oldclassInfo.UpdateID()
	newSc := &biz.StudentCourse{
		StuID:           req.GetStuId(),
		ClaID:           oldclassInfo.ID,
		Year:            oldclassInfo.Year,
		Semester:        oldclassInfo.Semester,
		IsManuallyAdded: false,
	}
	err = s.clu.UpdateClass(ctx, req.GetStuId(), req.GetYear(), req.GetSemester(), oldclassInfo, newSc, req.GetClassId())
	if err != nil {
		return &pb.UpdateClassResponse{
			Msg: "修改失败",
		}, err
	}
	return &pb.UpdateClassResponse{
		ClassId: oldclassInfo.ID,
		Msg:     "成功修改",
	}, nil
}

func (s *ClassListService) GetRecycleBinClassInfos(ctx context.Context, req *pb.GetRecycleBinClassRequest) (*pb.GetRecycleBinClassResponse, error) {
	valLogger := logger.GetLoggerFromCtx(ctx)
	valLogger = valLogger.With(
		logger.String("stu_id", req.GetStuId()),
		logger.String("year", req.GetYear()),
		logger.String("semester", req.GetSemester()),
	)
	ctx = logger.WithLogger(ctx, valLogger)

	if !tool.CheckSY(req.Semester, req.Year) {
		return &pb.GetRecycleBinClassResponse{}, errcode.ErrParam
	}
	classInfos, err := s.clu.GetRecycledClassInfos(ctx, req.GetStuId(), req.GetYear(), req.GetSemester())
	if err != nil {
		return &pb.GetRecycleBinClassResponse{}, err
	}
	pbClassInfos := make([]*pb.ClassInfo, 0)
	for _, classInfo := range classInfos {
		pbClassInfo := new(pb.ClassInfo)
		_ = copier.Copy(&pbClassInfo, &classInfo)
		pbClassInfo.Note = s.clu.GetClassNote(ctx, req.GetStuId(), req.GetYear(), req.GetSemester(), classInfo.ID)
		pbClassInfos = append(pbClassInfos, pbClassInfo)
	}
	return &pb.GetRecycleBinClassResponse{
		ClassInfos: pbClassInfos,
	}, nil
}

func (s *ClassListService) RecoverClass(ctx context.Context, req *pb.RecoverClassRequest) (*pb.RecoverClassResponse, error) {
	valLogger := logger.GetLoggerFromCtx(ctx)
	valLogger = valLogger.With(
		logger.String("stu_id", req.GetStuId()),
		logger.String("year", req.GetYear()),
		logger.String("semester", req.GetSemester()),
	)
	ctx = logger.WithLogger(ctx, valLogger)

	if !tool.CheckSY(req.Semester, req.Year) {
		return &pb.RecoverClassResponse{
			Msg: "恢复课程失败",
		}, errcode.ErrParam
	}

	err := s.clu.RecoverClassInfo(ctx, req.GetStuId(), req.GetYear(), req.GetSemester(), req.GetClassId())
	if err != nil {
		return &pb.RecoverClassResponse{
			Msg: "恢复课程失败",
		}, err
	}
	return &pb.RecoverClassResponse{
		Msg: "恢复课程成功",
	}, nil
}

func (s *ClassListService) GetStuIdByJxbId(ctx context.Context, req *pb.GetStuIdByJxbIdRequest) (*pb.GetStuIdByJxbIdResponse, error) {
	valLogger := logger.GetLoggerFromCtx(ctx)
	valLogger = valLogger.With(
		logger.String("jxb_id", req.GetJxbId()),
	)
	ctx = logger.WithLogger(ctx, valLogger)

	stuIds, err := s.clu.GetStuIdsByJxbId(ctx, req.GetJxbId())
	if err != nil {
		return &pb.GetStuIdByJxbIdResponse{}, errcode.ErrGetStuIdByJxbId
	}
	return &pb.GetStuIdByJxbIdResponse{
		StuId: stuIds,
	}, nil
}

func (s *ClassListService) GetAllClassInfo(ctx context.Context, req *pb.GetAllClassInfoRequest) (*pb.GetAllClassInfoResponse, error) {
	valLogger := logger.GetLoggerFromCtx(ctx)
	valLogger = valLogger.With(
		logger.String("year", req.GetYear()),
		logger.String("semester", req.GetSemester()),
	)
	ctx = logger.WithLogger(ctx, valLogger)

	cursor, err := time.Parse("2006-01-02T15:04:05.000000", req.Cursor)
	if err != nil {
		return &pb.GetAllClassInfoResponse{}, errcode.ErrParam
	}
	//// 转换为 UTC 时区
	//cursorUTC := cursor.In(time.UTC)

	classInfos := s.clu.GetAllSchoolClassInfosToOtherService(ctx, req.GetYear(), req.GetSemester(), cursor)
	if len(classInfos) == 0 {
		return &pb.GetAllClassInfoResponse{}, nil
	}
	pbClassInfos := make([]*pb.ClassInfo, 0)
	for _, classInfo := range classInfos {
		pbClassInfo := new(pb.ClassInfo)
		_ = copier.Copy(&pbClassInfo, &classInfo)

		pbClassInfos = append(pbClassInfos, pbClassInfo)
	}
	return &pb.GetAllClassInfoResponse{
		ClassInfos: pbClassInfos,
		LastTime:   tool.FormatTimeInUTC(classInfos[len(classInfos)-1].CreatedAt),
	}, nil
}

func (s *ClassListService) GetSchoolDay(ctx context.Context, req *pb.GetSchoolDayReq) (*pb.GetSchoolDayResp, error) {
	return &pb.GetSchoolDayResp{
		HolidayTime: s.schoolday.HolidayTime,
		SchoolTime:  s.schoolday.SchoolTime,
	}, nil
}

func (s *ClassListService) UpdateClassNote(ctx context.Context, req *pb.UpdateClassNoteReq) (*pb.UpdateClassNoteResp, error) {
	if !tool.CheckSY(req.Semester, req.Year) {
		return &pb.UpdateClassNoteResp{}, errcode.ErrParam
	}
	exist := s.clu.CheckSCIdsExist(ctx, req.StuId, req.Year, req.Semester, req.ClassId)
	if !exist {
		return &pb.UpdateClassNoteResp{
			Msg: "该课程不存在",
		}, errcode.ErrClassIsExist
	}
	err := s.clu.UpdateClassNote(ctx, req.StuId, req.Year, req.Semester, req.ClassId, req.Note)
	if err != nil {
		return &pb.UpdateClassNoteResp{
			Msg: "更新课程备注失败",
		}, err
	}
	return &pb.UpdateClassNoteResp{
		Msg: "更新课程备注成功",
	}, nil
}

func (s *ClassListService) DeleteClassNote(ctx context.Context, req *pb.DeleteClassNoteReq) (*pb.DeleteClassNoteResp, error) {
	if !tool.CheckSY(req.Semester, req.Year) {
		return &pb.DeleteClassNoteResp{}, errcode.ErrParam
	}
	exist := s.clu.CheckSCIdsExist(ctx, req.StuId, req.Year, req.Semester, req.ClassId)
	if !exist {
		return &pb.DeleteClassNoteResp{
			Msg: "该课程不存在",
		}, errcode.ErrClassIsExist
	}
	err := s.clu.UpdateClassNote(ctx, req.StuId, req.Year, req.Semester, req.ClassId, "")
	if err != nil {
		return &pb.DeleteClassNoteResp{
			Msg: "删除课程备注失败",
		}, err
	}
	return &pb.DeleteClassNoteResp{
		Msg: "删除课程备注成功",
	}, nil
}

func (s *ClassListService) GetClassNatures(ctx context.Context, req *pb.GetClassNaturesReq) (*pb.GetClassNaturesResp, error) {
	logh := logger.GetLoggerFromCtx(ctx)
	valLogger := logh.With(
		logger.String("stu_id", req.GetStuId()),
	)
	ctx = logger.WithLogger(ctx, valLogger)
	natures := s.clu.GetClassNatures(ctx, req.GetStuId())
	return &pb.GetClassNaturesResp{
		ClassNatures: natures,
	}, nil
}

func convertToShanghaiTimeStamp(t time.Time) int64 {
	return tool.ToShanghaiTime(t).Unix()
}
