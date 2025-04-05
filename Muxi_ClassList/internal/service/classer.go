package service

import (
	"context"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/conf"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/errcode"
	model2 "github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/model"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/pkg/tool"
	pb "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classlist/v1" //此处改成了be-api中的,方便其他服务调用.
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

type ClasserService struct {
	pb.UnimplementedClasserServer
	clu       ClassCtrl
	schoolday *conf.SchoolDay
	log       *log.Helper
}

func NewClasserService(clu ClassCtrl, day *conf.SchoolDay, logger log.Logger) *ClasserService {
	return &ClasserService{
		clu:       clu,
		log:       log.NewHelper(logger),
		schoolday: day,
	}
}

func (s *ClasserService) GetClass(ctx context.Context, req *pb.GetClassRequest) (*pb.GetClassResponse, error) {
	if !tool.CheckSY(req.Semester, req.Year) {
		return &pb.GetClassResponse{}, errcode.ErrParam
	}
	pclasses := make([]*pb.Class, 0)
	classes, err := s.clu.GetClasses(ctx, req.GetStuId(), req.GetYear(), req.GetSemester(), req.GetRefresh())
	if err != nil {
		return &pb.GetClassResponse{}, err
	}
	for _, class := range classes {
		pinfo := HandleClass(class.Info)
		var pclass = &pb.Class{
			Info: pinfo,
		}
		pclasses = append(pclasses, pclass)
	}
	return &pb.GetClassResponse{
		Classes: pclasses,
	}, nil
}
func (s *ClasserService) AddClass(ctx context.Context, req *pb.AddClassRequest) (*pb.AddClassResponse, error) {
	if !tool.CheckSY(req.Semester, req.Year) || req.GetWeeks() <= 0 || !tool.CheckIfThisYear(req.Year, req.Semester) {
		return &pb.AddClassResponse{}, errcode.ErrParam
	}
	weekDur := tool.FormatWeeks(tool.ParseWeeks(req.Weeks))
	var classInfo = &model2.ClassInfo{
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
func (s *ClasserService) DeleteClass(ctx context.Context, req *pb.DeleteClassRequest) (*pb.DeleteClassResponse, error) {
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
func (s *ClasserService) UpdateClass(ctx context.Context, req *pb.UpdateClassRequest) (*pb.UpdateClassResponse, error) {
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
	newSc := &model2.StudentCourse{
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
func (s *ClasserService) GetRecycleBinClassInfos(ctx context.Context, req *pb.GetRecycleBinClassRequest) (*pb.GetRecycleBinClassResponse, error) {
	if !tool.CheckSY(req.Semester, req.Year) {
		return &pb.GetRecycleBinClassResponse{}, errcode.ErrParam
	}
	classInfos, err := s.clu.GetRecycledClassInfos(ctx, req.GetStuId(), req.GetYear(), req.GetSemester())
	if err != nil {
		return &pb.GetRecycleBinClassResponse{}, err
	}
	pbClassInfos := make([]*pb.ClassInfo, 0)
	for _, classInfo := range classInfos {
		pbClassInfos = append(pbClassInfos, HandleClass(classInfo))
	}
	return &pb.GetRecycleBinClassResponse{
		ClassInfos: pbClassInfos,
	}, nil
}
func (s *ClasserService) RecoverClass(ctx context.Context, req *pb.RecoverClassRequest) (*pb.RecoverClassResponse, error) {
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
func (s *ClasserService) GetStuIdByJxbId(ctx context.Context, req *pb.GetStuIdByJxbIdRequest) (*pb.GetStuIdByJxbIdResponse, error) {
	stuIds, err := s.clu.GetStuIdsByJxbId(ctx, req.GetJxbId())
	if err != nil {

		return &pb.GetStuIdByJxbIdResponse{}, errcode.ErrGetStuIdByJxbId
	}
	return &pb.GetStuIdByJxbIdResponse{
		StuId: stuIds,
	}, nil
}
func (s *ClasserService) GetAllClassInfo(ctx context.Context, req *pb.GetAllClassInfoRequest) (*pb.GetAllClassInfoResponse, error) {
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
		pbClassInfos = append(pbClassInfos, HandleClass(classInfo))
	}
	return &pb.GetAllClassInfoResponse{
		ClassInfos: pbClassInfos,
		LastTime:   tool.FormatTimeInUTC(classInfos[len(classInfos)-1].CreatedAt),
	}, nil
}

func (s *ClasserService) GetSchoolDay(ctx context.Context, req *pb.GetSchoolDayReq) (*pb.GetSchoolDayResp, error) {
	return &pb.GetSchoolDayResp{
		HolidayTime: s.schoolday.HolidayTime,
		SchoolTime:  s.schoolday.SchoolTime,
	}, nil
}

func HandleClass(info *model2.ClassInfo) *pb.ClassInfo {
	return &pb.ClassInfo{
		Day:          info.Day,
		Teacher:      info.Teacher,
		Where:        info.Where,
		ClassWhen:    info.ClassWhen,
		WeekDuration: info.WeekDuration,
		Classname:    info.Classname,
		Credit:       info.Credit,
		Weeks:        info.Weeks,
		Id:           info.ID,
		Semester:     info.Semester,
		Year:         info.Year,
	}
}
