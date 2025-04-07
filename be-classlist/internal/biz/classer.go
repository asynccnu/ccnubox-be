package biz

import (
	"context"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/classLog"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/pkg/tool"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

type ClassUsecase struct {
	classRepo ClassRepoProxy
	crawler   ClassCrawler
	ccnu      CCNUServiceProxy
	jxbRepo   JxbRepo
	log       *log.Helper
}

func NewClassUsecase(classRepo ClassRepoProxy, crawler ClassCrawler, JxbRepo JxbRepo, Cs CCNUServiceProxy, logger log.Logger) *ClassUsecase {
	return &ClassUsecase{
		classRepo: classRepo,
		crawler:   crawler,
		jxbRepo:   JxbRepo,
		ccnu:      Cs,
		log:       log.NewHelper(logger),
	}
}

func (cluc *ClassUsecase) GetClasses(ctx context.Context, stuID, year, semester string, refresh bool) ([]*model.Class, error) {
	var (
		scs            = make([]*model.StudentCourse, 0)
		classes        = make([]*model.Class, 0)
		classInfos     = make([]*model.ClassInfo, 0)
		SearchFromCCNU = refresh
	)

	if !refresh {
		//直接从数据库中获取课表
		resp1, err := cluc.classRepo.GetClassesFromLocal(ctx, model.GetClassesFromLocalReq{
			StuID:    stuID,
			Year:     year,
			Semester: semester,
		})

		if resp1 != nil && len(resp1.ClassInfos) > 0 {
			classInfos = resp1.ClassInfos
		}

		// 如果数据库中没有
		if err != nil {
			SearchFromCCNU = true

			crawClassInfos, crawScs, err := cluc.getCourseFromCrawler(ctx, stuID, year, semester)
			if err == nil {
				classInfos = crawClassInfos
				scs = crawScs
			}
		}
	} else {
		crawClassInfos, crawScs, err := cluc.getCourseFromCrawler(ctx, stuID, year, semester)
		if err == nil {
			SearchFromCCNU = true
			classInfos = crawClassInfos
			scs = crawScs

			//还要算上手动添加的课程
			//从数据库中获取手动添加的课程
			resp2, err1 := cluc.classRepo.GetAddedClasses(ctx, model.GetAddedClassesReq{
				StudID:   stuID,
				Year:     year,
				Semester: semester,
			})
			if err1 == nil && len(resp2.ClassInfos) > 0 {
				classInfos = append(classInfos, resp2.ClassInfos...)
			}
		} else {
			//如果爬取失败
			SearchFromCCNU = false

			//使用本地数据库做兜底
			resp1, err := cluc.classRepo.GetClassesFromLocal(ctx, model.GetClassesFromLocalReq{
				StuID:    stuID,
				Year:     year,
				Semester: semester,
			})

			if resp1 != nil && len(resp1.ClassInfos) > 0 {
				classInfos = resp1.ClassInfos
			}
			if err != nil {
				cluc.log.Errorf("get classlist[%v %v %v] from DB failed: %v", stuID, year, semester, err)
			}
		}
	}

	//如果所有获取途径均失效，则返回错误
	if len(classInfos) == 0 {
		return nil, errcode.ErrClassNotFound
	}

	//封装class,并获取jxbID
	classes, jxbIDs := convertToClass(classInfos)

	//如果是从CCNU那边查到的，就存储
	if SearchFromCCNU {
		//开个协程来存取
		go func() {
			cluc.classRepo.SaveClass(context.Background(), stuID, year, semester, classInfos, scs)

			//防止ctx因为return就被取消了，所以就改用background，因为这个存取没有精确的要求，所以可以后台完成，用户不需要感知
			if err := cluc.jxbRepo.SaveJxb(context.Background(), stuID, jxbIDs); err != nil {
				cluc.log.Warnw(classLog.Msg, "SaveJxb err",
					classLog.Param, fmt.Sprintf("%v,%v", stuID, jxbIDs),
					classLog.Reason, err)
			}
		}()
	}
	return classes, nil
}

func (cluc *ClassUsecase) AddClass(ctx context.Context, stuID string, info *model.ClassInfo) error {
	sc := &model.StudentCourse{
		StuID:           stuID,
		ClaID:           info.ID,
		Year:            info.Year,
		Semester:        info.Semester,
		IsManuallyAdded: true, //手动添加课程
	}
	//检查是否添加的课程是否已经存在
	if cluc.classRepo.CheckSCIdsExist(ctx, model.CheckSCIdsExistReq{StuID: stuID, Year: info.Year, Semester: info.Semester, ClassId: info.ID}) {
		cluc.log.Errorf("[%v] already exists", info)
		return errcode.ErrClassIsExist
	}
	//添加课程
	err := cluc.classRepo.AddClass(ctx, model.AddClassReq{
		StuID:     stuID,
		Year:      info.Year,
		Semester:  info.Semester,
		ClassInfo: info,
		Sc:        sc,
	})
	if err != nil {
		return err
	}
	return nil
}
func (cluc *ClassUsecase) DeleteClass(ctx context.Context, stuID, year, semester, classId string) error {
	//删除课程
	err := cluc.classRepo.DeleteClass(ctx, model.DeleteClassReq{
		StuID:    stuID,
		Year:     year,
		Semester: semester,
		ClassId:  []string{classId},
	})
	if err != nil {
		cluc.log.Errorf("delete classlist [%v] failed", classId)
		return errcode.ErrClassDelete
	}
	return nil
}
func (cluc *ClassUsecase) GetRecycledClassInfos(ctx context.Context, stuID, year, semester string) ([]*model.ClassInfo, error) {
	//获取回收站的课程ID
	RecycledClassIds, err := cluc.classRepo.GetRecycledIds(ctx, model.GetRecycledIdsReq{
		StuID:    stuID,
		Year:     year,
		Semester: semester,
	})
	if err != nil {
		return nil, err
	}
	classInfos := make([]*model.ClassInfo, 0)
	//从数据库中查询课程
	for _, classId := range RecycledClassIds.Ids {
		resp, err := cluc.classRepo.GetSpecificClassInfo(ctx, model.GetSpecificClassInfoReq{
			StuID:    stuID,
			Year:     year,
			Semester: semester,
			ClassId:  classId})
		if err != nil {
			continue
		}
		classInfos = append(classInfos, resp.ClassInfo)
	}
	return classInfos, nil
}
func (cluc *ClassUsecase) RecoverClassInfo(ctx context.Context, stuID, year, semester, classId string) error {
	//先检查要回复的课程ID是否存在于回收站中
	exist := cluc.classRepo.CheckClassIdIsInRecycledBin(ctx, model.CheckClassIdIsInRecycledBinReq{
		StuID:    stuID,
		Year:     year,
		Semester: semester,
		ClassId:  classId,
	})
	if !exist {
		return errcode.ErrRecycleBinDoNotHaveIt
	}
	//获取该ID的课程信息
	RecycledClassInfo, err := cluc.SearchClass(ctx, classId)
	if err != nil {
		return errcode.ErrRecover
	}
	err = cluc.AddClass(ctx, stuID, RecycledClassInfo)
	if err != nil {
		return errcode.ErrRecover
	}
	//恢复对应的关系
	err = cluc.classRepo.RecoverClassFromRecycledBin(ctx, model.RecoverClassFromRecycleBinReq{
		ClassId: classId,
	})
	if err != nil {
		return errcode.ErrRecover
	}
	return nil
}
func (cluc *ClassUsecase) SearchClass(ctx context.Context, classId string) (*model.ClassInfo, error) {
	resp, err := cluc.classRepo.GetSpecificClassInfo(ctx, model.GetSpecificClassInfoReq{ClassId: classId})
	if err != nil {
		return nil, err
	}
	return resp.ClassInfo, nil
}
func (cluc *ClassUsecase) UpdateClass(ctx context.Context, stuID, year, semester string, newClassInfo *model.ClassInfo, newSc *model.StudentCourse, oldClassId string) error {
	err := cluc.classRepo.UpdateClass(ctx, model.UpdateClassReq{
		StuID:        stuID,
		Year:         year,
		Semester:     semester,
		NewClassInfo: newClassInfo,
		NewSc:        newSc,
		OldClassId:   oldClassId,
	})
	if err != nil {
		return err
	}
	return nil
}
func (cluc *ClassUsecase) CheckSCIdsExist(ctx context.Context, stuID, year, semester, classId string) bool {
	return cluc.classRepo.CheckSCIdsExist(ctx, model.CheckSCIdsExistReq{
		StuID:    stuID,
		Year:     year,
		Semester: semester,
		ClassId:  classId,
	})
}
func (cluc *ClassUsecase) GetAllSchoolClassInfosToOtherService(ctx context.Context, year, semester string, cursor time.Time) []*model.ClassInfo {
	return cluc.classRepo.GetAllSchoolClassInfos(ctx, model.GetAllSchoolClassInfosReq{
		Year:     year,
		Semester: semester,
		Cursor:   cursor,
	}).ClassInfos
}
func (cluc *ClassUsecase) GetStuIdsByJxbId(ctx context.Context, jxbId string) ([]string, error) {
	return cluc.jxbRepo.FindStuIdsByJxbId(ctx, jxbId)
}

func (cluc *ClassUsecase) getCourseFromCrawler(ctx context.Context, stuID string, year string, semester string) ([]*model.ClassInfo, []*model.StudentCourse, error) {
	////测试用的
	//cookie := "JSESSIONID=77CCA81367438A56D3AFF46797E674A4"

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // 10秒超时,防止影响
	defer cancel()                                                 // 确保在函数返回前取消上下文，防止资源泄漏

	getCookieStart := time.Now()

	cookie, err := cluc.ccnu.GetCookie(timeoutCtx, stuID)
	if err != nil {
		cluc.log.Errorf("Error getting cookie(stu_id:%v) from other service", stuID)
		return nil, nil, err
	}

	cluc.log.Infof("获取cookie (stu_id:%v) 从其他服务中,花费了 %v", stuID, time.Since(getCookieStart))

	var stu Student
	if tool.CheckIsUndergraduate(stuID) { //针对是否是本科生，进行分类
		stu = &Undergraduate{}
	} else {
		stu = &GraduateStudent{}
	}
	classinfos, scs, err := stu.GetClass(ctx, stuID, year, semester, cookie, cluc.crawler)
	if err != nil {
		cluc.log.Errorf("craw classlist(stu_id:%v year:%v semester:%v cookie:%v) failed: %v", stuID, year, semester, cookie, err)
		return nil, nil, err
	}
	return classinfos, scs, nil
}

func convertToClass(infos []*model.ClassInfo) ([]*model.Class, []string) {
	if len(infos) == 0 {
		return nil, nil
	}
	Jxbmp := make(map[string]struct{})
	classes := make([]*model.Class, 0, len(infos))
	for _, classInfo := range infos {
		//thisWeek := classInfo.SearchWeek(week)
		class := &model.Class{
			Info: classInfo,
			//ThisWeek: thisWeek && tool.CheckIfThisYear(classInfo.Year, classInfo.Semester),
		}
		if classInfo.JxbId != "" {
			Jxbmp[classInfo.JxbId] = struct{}{}
		}
		classes = append(classes, class)
	}
	jxbIDs := make([]string, 0, len(Jxbmp))
	for k := range Jxbmp {
		jxbIDs = append(jxbIDs, k)
	}
	return classes, jxbIDs
}

// Student 学生接口
type Student interface {
	GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*model.ClassInfo, []*model.StudentCourse, error)
}
type Undergraduate struct{}

func (u *Undergraduate) GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*model.ClassInfo, []*model.StudentCourse, error) {
	var (
		classInfos = make([]*model.ClassInfo, 0)
		scs        = make([]*model.StudentCourse, 0)
	)
	resp, err := craw.GetClassInfosForUndergraduate(ctx, model.GetClassInfosForUndergraduateReq{
		StuID:    stuID,
		Year:     year,
		Semester: semester,
		Cookie:   cookie,
	})
	if err != nil {
		return nil, nil, err
	}
	if resp != nil {
		if resp.ClassInfos != nil {
			classInfos = resp.ClassInfos
		}
		if resp.StudentCourses != nil {
			scs = resp.StudentCourses
		}
	}
	return classInfos, scs, nil
}

type GraduateStudent struct{}

func (g *GraduateStudent) GetClass(ctx context.Context, stuID, year, semester, cookie string, craw ClassCrawler) ([]*model.ClassInfo, []*model.StudentCourse, error) {
	var (
		classInfos = make([]*model.ClassInfo, 0)
		scs        = make([]*model.StudentCourse, 0)
	)
	resp2, err := craw.GetClassInfoForGraduateStudent(ctx, model.GetClassInfoForGraduateStudentReq{
		StuID:    stuID,
		Year:     year,
		Semester: semester,
		Cookie:   cookie,
	})
	if err != nil {
		return nil, nil, err
	}
	if resp2.ClassInfos != nil {
		classInfos = resp2.ClassInfos
	}
	if resp2.StudentCourses != nil {
		scs = resp2.StudentCourses
	}
	return classInfos, scs, nil
}
