package biz

import (
	"context"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

const MaxNum = 20

type ClassRepo struct {
	ClaRepo *ClassInfoRepo
	Sac     *StudentAndCourseRepo
	TxCtrl  Transaction //控制事务的开启
	log     *log.Helper
}

func NewClassRepo(ClaRepo *ClassInfoRepo, TxCtrl Transaction, Sac *StudentAndCourseRepo, logger log.Logger) *ClassRepo {
	return &ClassRepo{
		ClaRepo: ClaRepo,
		Sac:     Sac,
		log:     log.NewHelper(logger),
		TxCtrl:  TxCtrl,
	}
}

// GetClassesFromLocal 从本地获取课程
func (cla ClassRepo) GetClassesFromLocal(ctx context.Context, req model.GetClassesFromLocalReq) (*model.GetClassesFromLocalResp, error) {
	var (
		cacheGet = true
		key      = GenerateClassInfosKey(req.StuID, req.Year, req.Semester)
	)

	classInfos, err := cla.ClaRepo.Cache.GetClassInfosFromCache(ctx, key)
	//如果err!=nil(err==redis.Nil)说明该ID第一次进入（redis中没有这个KEY），且未经过数据库，则允许其查数据库，所以要设置cacheGet=false
	//如果err==nil说明其至少经过数据库了，redis中有这个KEY,但可能值为NULL，如果不为NULL，就说明缓存命中了,直接返回没有问题
	//如果为NULL，就说明数据库中没有的数据，其依然在请求，会影响数据库（缓存穿透），我们依然直接返回
	//这时我们就需要直接返回redis中的null，即直接返回nil,而不经过数据库

	if err != nil {
		cacheGet = false
		cla.log.Warnf("Get Class [%+v] From Cache failed: %v", req, err)
	}
	if !cacheGet {
		//从数据库中获取
		classInfos, err = cla.ClaRepo.DB.GetClassInfos(ctx, req.StuID, req.Year, req.Semester)
		if err != nil {
			cla.log.Errorf("Get Class [%+v] From DB failed: %v", req, err)
			return nil, errcode.ErrClassFound
		}
		go func() {
			//将课程信息当作整体存入redis
			//注意:如果未获取到，即classInfos为nil，redis仍然会设置key-value，只不过value为NULL
			_ = cla.ClaRepo.Cache.AddClaInfosToCache(context.Background(), key, classInfos)
		}()
	}
	//检查classInfos是否为空
	//如果不为空，直接返回就好
	//如果为空，则说明没有该数据，需要去查询
	//如果不添加此条件，即便你redis中有值为NULL的话，也不会返回错误，就导致不会去爬取更新，所以需要该条件
	//添加该条件，能够让查询数据库的操作效率更高，同时也保证了数据的获取
	if len(classInfos) == 0 {
		return nil, errcode.ErrClassNotFound
	}
	return &model.GetClassesFromLocalResp{ClassInfos: classInfos}, nil
}

func (cla ClassRepo) GetSpecificClassInfo(ctx context.Context, req model.GetSpecificClassInfoReq) (*model.GetSpecificClassInfoResp, error) {
	classInfo, err := cla.ClaRepo.DB.GetClassInfoFromDB(ctx, req.ClassId)
	if err != nil || classInfo == nil {
		return nil, errcode.ErrClassNotFound
	}
	return &model.GetSpecificClassInfoResp{ClassInfo: classInfo}, nil
}
func (cla ClassRepo) AddClass(ctx context.Context, req model.AddClassReq) error {
	err := cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, GenerateClassInfosKey(req.StuID, req.Year, req.Semester))
	if err != nil {
		return err
	}
	errTx := cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		if err := cla.ClaRepo.DB.AddClassInfoToDB(ctx, req.ClassInfo); err != nil {
			return errcode.ErrClassUpdate
		}
		// 处理 StudentCourse
		if err := cla.Sac.DB.SaveStudentAndCourseToDB(ctx, req.Sc); err != nil {
			return errcode.ErrClassUpdate
		}
		cnt, err := cla.Sac.DB.GetClassNum(ctx, req.StuID, req.Year, req.Semester, req.Sc.IsManuallyAdded)
		if err == nil && cnt > MaxNum {
			return fmt.Errorf("classlist num limit")
		}
		return nil
	})
	if errTx != nil {
		cla.log.Errorf("Add Class [%+v] failed:%v", req, errTx)
		return errTx
	}
	go func() {
		//延迟双删
		time.AfterFunc(1*time.Second, func() {
			_ = cla.ClaRepo.Cache.DeleteClassInfoFromCache(context.Background(), GenerateClassInfosKey(req.StuID, req.Year, req.Semester))
		})
	}()
	return nil
}
func (cla ClassRepo) DeleteClass(ctx context.Context, req model.DeleteClassReq) error {
	//先删除缓存信息
	err := cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, GenerateClassInfosKey(req.StuID, req.Year, req.Semester))
	if err != nil {
		cla.log.Errorf("Delete Class [%+v] from Cache failed:%v", req, err)
		return err
	}
	//删除并添加进回收站
	recycleSetName := GenerateRecycleSetName(req.StuID, req.Year, req.Semester)
	err = cla.Sac.Cache.RecycleClassId(ctx, recycleSetName, req.ClassId...)
	if err != nil {
		cla.log.Errorf("Add Class [%+v] To RecycleBin failed:%v", req, err)
		return err
	}
	//从数据库中删除对应的关系
	errTx := cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		err := cla.Sac.DB.DeleteStudentAndCourseInDB(ctx, req.StuID, req.Year, req.Semester, req.ClassId)
		if err != nil {
			return fmt.Errorf("error deleting student: %w", err)
		}
		return nil
	})
	if errTx != nil {
		cla.log.Errorf("Delete Class [%+v] In DB failed:%v", req, errTx)
		return errTx
	}
	return nil
}
func (cla ClassRepo) GetRecycledIds(ctx context.Context, req model.GetRecycledIdsReq) (*model.GetRecycledIdsResp, error) {
	recycleKey := GenerateRecycleSetName(req.StuID, req.Year, req.Semester)
	classIds, err := cla.Sac.Cache.GetRecycledClassIds(ctx, recycleKey)
	if err != nil {
		return nil, err
	}
	return &model.GetRecycledIdsResp{Ids: classIds}, nil
}
func (cla ClassRepo) CheckClassIdIsInRecycledBin(ctx context.Context, req model.CheckClassIdIsInRecycledBinReq) bool {

	RecycledBinKey := GenerateRecycleSetName(req.StuID, req.Year, req.Semester)
	return cla.Sac.Cache.CheckRecycleIdIsExist(ctx, RecycledBinKey, req.ClassId)
}
func (cla ClassRepo) RecoverClassFromRecycledBin(ctx context.Context, req model.RecoverClassFromRecycleBinReq) error {
	RecycledBinKey := GenerateRecycleSetName(req.StuID, req.Year, req.Semester)
	return cla.Sac.Cache.RemoveClassFromRecycledBin(ctx, RecycledBinKey, req.ClassId)
}
func (cla ClassRepo) UpdateClass(ctx context.Context, req model.UpdateClassReq) error {
	err := cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, GenerateClassInfosKey(req.StuID, req.Year, req.Semester))
	if err != nil {
		return err
	}
	errTx := cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		//添加新的课程信息
		err := cla.ClaRepo.DB.AddClassInfoToDB(ctx, req.NewClassInfo)
		if err != nil {
			return errcode.ErrClassUpdate
		}
		//删除原本的学生与课程的对应关系
		err = cla.Sac.DB.DeleteStudentAndCourseInDB(ctx, req.StuID, req.Year, req.Semester, []string{req.OldClassId})
		if err != nil {
			return errcode.ErrClassUpdate
		}
		//添加新的对应关系
		err = cla.Sac.DB.SaveStudentAndCourseToDB(ctx, req.NewSc)
		if err != nil {
			return errcode.ErrClassUpdate
		}
		return nil
	})
	if errTx != nil {
		cla.log.Errorf("Update Class [%+v] In DB  failed:%v", req, errTx)
		return errTx
	}

	go func() {
		//延迟双删
		time.AfterFunc(1*time.Second, func() {
			_ = cla.ClaRepo.Cache.DeleteClassInfoFromCache(context.Background(), GenerateClassInfosKey(req.StuID, req.Year, req.Semester))
		})
	}()

	return nil
}

// SaveClass 保存课程[删除原本的，添加新的，主要是为了防止感知不到原本的和新增的之间有差异]
func (cla ClassRepo) SaveClass(ctx context.Context, stuID, year, semester string, classInfos []*model.ClassInfo, scs []*model.StudentCourse) {
	key := GenerateClassInfosKey(stuID, year, semester)

	_ = cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, key)

	err := cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		//删除对应的所有关系[只删除非手动添加的]
		err := cla.Sac.DB.DeleteStudentAndCourseByTimeFromDB(ctx, stuID, year, semester)
		if err != nil {
			return err
		}
		//保存课程信息到db
		err = cla.ClaRepo.DB.SaveClassInfosToDB(ctx, classInfos)
		if err != nil {
			return err
		}
		//保存新的关系
		err = cla.Sac.DB.SaveManyStudentAndCourseToDB(ctx, scs)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		cla.log.Errorf("Save class [%+v] and scs [%v] failed:%v", classInfos, scs, err)
	}

	go func() {
		//延迟双删
		time.AfterFunc(1*time.Second, func() {
			_ = cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, key)
		})
	}()
}

func (cla ClassRepo) CheckSCIdsExist(ctx context.Context, req model.CheckSCIdsExistReq) bool {
	return cla.Sac.DB.CheckExists(ctx, req.Year, req.Semester, req.StuID, req.ClassId)
}
func (cla ClassRepo) GetAllSchoolClassInfos(ctx context.Context, req model.GetAllSchoolClassInfosReq) *model.GetAllSchoolClassInfosResp {
	classInfos, err := cla.ClaRepo.DB.GetAllClassInfos(ctx, req.Year, req.Semester, req.Cursor)
	if err != nil {
		return nil
	}
	return &model.GetAllSchoolClassInfosResp{ClassInfos: classInfos}
}

func (cla ClassRepo) GetAddedClasses(ctx context.Context, req model.GetAddedClassesReq) (*model.GetAddedClassesResp, error) {
	classInfos, err := cla.ClaRepo.DB.GetAddedClassInfos(ctx, req.StudID, req.Year, req.Semester)
	if err != nil {
		return nil, err
	}
	return &model.GetAddedClassesResp{ClassInfos: classInfos}, nil
}

func GenerateRecycleSetName(stuId, xnm, xqm string) string {
	return fmt.Sprintf("Recycle:%s:%s:%s", stuId, xnm, xqm)
}
func GenerateClassInfosKey(stuId, xnm, xqm string) string {
	return fmt.Sprintf("ClassInfos:%s:%s:%s", stuId, xnm, xqm)
}
