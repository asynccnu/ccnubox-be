package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/data/do"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/jinzhu/copier"
)

// MaxNum 每个学期最多允许添加的课程数量
const MaxNum = 20

type ClassInfoRepo struct {
	DB    *ClassInfoDBRepo
	Cache *ClassInfoCacheRepo
}

func NewClassInfoRepo(DB *ClassInfoDBRepo, Cache *ClassInfoCacheRepo) *ClassInfoRepo {
	return &ClassInfoRepo{
		DB:    DB,
		Cache: Cache,
	}
}

type StudentAndCourseRepo struct {
	DB    *StudentAndCourseDBRepo
	Cache *StudentAndCourseCacheRepo
}

func NewStudentAndCourseRepo(DB *StudentAndCourseDBRepo, Cache *StudentAndCourseCacheRepo) *StudentAndCourseRepo {
	return &StudentAndCourseRepo{
		DB:    DB,
		Cache: Cache,
	}
}

type ClassRepo struct {
	ClaRepo *ClassInfoRepo
	Sac     *StudentAndCourseRepo
	TxCtrl  Transaction //控制事务的开启
}

func NewClassRepo(ClaRepo *ClassInfoRepo, TxCtrl Transaction, Sac *StudentAndCourseRepo) *ClassRepo {
	return &ClassRepo{
		ClaRepo: ClaRepo,
		Sac:     Sac,
		TxCtrl:  TxCtrl,
	}
}

// GetClassesFromLocal 从本地获取课程
func (cla ClassRepo) GetClassesFromLocal(ctx context.Context, stuID, year, semester string) ([]*biz.ClassInfo, error) {
	logh := logger.GetLoggerFromCtx(ctx)
	noExpireCtx := context.WithoutCancel(ctx)

	var (
		cacheGet = true
		key      = cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester)
	)

	classInfos, err := cla.ClaRepo.Cache.GetClassInfosFromCache(ctx, key)
	//如果err!=nil(err==redis.Nil)说明该ID第一次进入（redis中没有这个KEY），且未经过数据库，则允许其查数据库，所以要设置cacheGet=false
	//如果err==nil说明其至少经过数据库了，redis中有这个KEY,但可能值为NULL，如果不为NULL，就说明缓存命中了,直接返回没有问题
	//如果为NULL，就说明数据库中没有的数据，其依然在请求，会影响数据库（缓存穿透），我们依然直接返回
	//这时我们就需要直接返回redis中的null，即直接返回nil,而不经过数据库

	if err != nil {
		cacheGet = false
		logh.Warnf("Get Class [%v %v %v] From Cache failed: %v", stuID, year, semester, err)
	}
	if !cacheGet {
		//从数据库中获取
		classInfos, err = cla.ClaRepo.DB.GetClassInfos(ctx, stuID, year, semester)
		if err != nil {
			logh.Errorf("Get Class [%v %v %v] From DB failed: %v", stuID, year, semester, err)
			return nil, errcode.ErrClassFound
		}
		go func() {
			//将课程信息当作整体存入redis
			//注意:如果未获取到，即classInfos为nil，redis仍然会设置key-value，只不过value为NULL
			_ = cla.ClaRepo.Cache.AddClaInfosToCache(noExpireCtx, key, classInfos)
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

	classInfosBiz := make([]*biz.ClassInfo, len(classInfos))
	_ = copier.Copy(&classInfosBiz, &classInfos)

	return classInfosBiz, nil
}

// GetSpecificClassInfo 获取特定课程信息
func (cla ClassRepo) GetSpecificClassInfo(ctx context.Context, classID string) (*biz.ClassInfo, error) {
	classInfo, err := cla.ClaRepo.DB.GetClassInfoFromDB(ctx, classID)
	if err != nil || classInfo == nil {
		return nil, errcode.ErrClassNotFound
	}

	//将do.ClassInfo转换为biz.ClassInfo
	classInfoBiz := new(biz.ClassInfo)
	_ = copier.Copy(&classInfoBiz, &classInfo)
	return classInfoBiz, nil
}

// AddClass 添加课程信息
func (cla ClassRepo) AddClass(ctx context.Context, stuID, year, semester string, classInfo *biz.ClassInfo, sc *biz.StudentCourse) error {
	logh := logger.GetLoggerFromCtx(ctx)
	err := cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester))
	if err != nil {
		return err
	}

	classInfoDo := new(do.ClassInfo)

	scDo := new(do.StudentCourse)

	//将biz.ClassInfo转换为do.ClassInfo
	_ = copier.Copy(&classInfoDo, &classInfo)
	//将biz.StudentCourse转换为do.StudentCourse
	_ = copier.Copy(&scDo, &sc)

	errTx := cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		if err := cla.ClaRepo.DB.AddClassInfoToDB(ctx, classInfoDo); err != nil {
			return errcode.ErrClassUpdate
		}
		// 处理 StudentCourse
		if err := cla.Sac.DB.SaveStudentAndCourseToDB(ctx, scDo); err != nil {
			return errcode.ErrClassUpdate
		}
		cnt, err := cla.Sac.DB.GetClassNum(ctx, stuID, year, semester, sc.IsManuallyAdded)
		if err == nil && cnt > MaxNum {
			return fmt.Errorf("classlist num limit")
		}
		return nil
	})
	if errTx != nil {
		logh.Errorf("Add Class [%v,%v,%v,%+v,%+v] failed:%v", stuID, year, semester, classInfo, sc, errTx)
		return errTx
	}
	go func() {
		//延迟双删
		time.AfterFunc(1*time.Second, func() {
			_ = cla.ClaRepo.Cache.DeleteClassInfoFromCache(context.Background(), cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester))
		})
	}()
	return nil
}

// DeleteClass 删除课程信息
func (cla ClassRepo) DeleteClass(ctx context.Context, stuID, year, semester string, classID []string) error {
	logh := logger.GetLoggerFromCtx(ctx)
	//先删除缓存信息
	err := cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester))
	if err != nil {
		logh.Errorf("Delete Class [%v,%v,%v,%v] from Cache failed:%v", stuID, year, semester, classID, err)
		return err
	}
	// 获取删除课程手动添加的信息
	isManuallyAddedCourse := cla.Sac.DB.CheckManualCourseStatus(ctx, stuID, year, semester, classID[0])

	//删除并添加进回收站
	recycleSetName := cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester)

	err = cla.Sac.Cache.RecycleClassId(ctx, recycleSetName, classID[0], isManuallyAddedCourse)
	if err != nil {
		logh.Errorf("Add Class [%v,%v,%v,%v] To RecycleBin failed:%v", stuID, year, semester, classID, err)
		return err
	}

	//从数据库中删除对应的关系
	errTx := cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		err := cla.Sac.DB.DeleteStudentAndCourseInDB(ctx, stuID, year, semester, classID)
		if err != nil {
			return fmt.Errorf("error deleting student: %w", err)
		}
		return nil
	})
	if errTx != nil {
		logh.Errorf("Delete Class [%v,%v,%v,%v] In DB failed:%v", stuID, year, semester, classID, errTx)
		return errTx
	}
	return nil
}

// GetRecycledIds 获取回收站中的课程ID
func (cla ClassRepo) GetRecycledIds(ctx context.Context, stuID, year, semester string) ([]string, error) {
	recycleKey := cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester)
	classIds, err := cla.Sac.Cache.GetRecycledClassIds(ctx, recycleKey)
	if err != nil {
		return nil, err
	}
	return classIds, nil
}

// IsRecycledCourseManual 检查课程是否为手动添加的回收课程
func (cla ClassRepo) IsRecycledCourseManual(ctx context.Context, stuID, year, semester, classID string) bool {
	recycleKey := cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester)
	return cla.Sac.Cache.IsRecycledCourseManual(ctx, recycleKey, classID)
}

// CheckClassIdIsInRecycledBin 检查课程ID是否在回收站中
func (cla ClassRepo) CheckClassIdIsInRecycledBin(ctx context.Context, stuID, year, semester, classID string) bool {
	RecycledBinKey := cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester)
	return cla.Sac.Cache.CheckRecycleIdIsExist(ctx, RecycledBinKey, classID)
}

// RemoveClassFromRecycledBin 从回收站中删除课程
func (cla ClassRepo) RemoveClassFromRecycledBin(ctx context.Context, stuID, year, semester, classID string) error {
	RecycledBinKey := cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester)
	return cla.Sac.Cache.RemoveClassFromRecycledBin(ctx, RecycledBinKey, classID)
}

// UpdateClass 更新课程信息
func (cla ClassRepo) UpdateClass(ctx context.Context, stuID, year, semester, oldClassID string,
	newClassInfo *biz.ClassInfo, newSc *biz.StudentCourse) error {

	logh := logger.GetLoggerFromCtx(ctx)
	noExpireCtx := context.WithoutCancel(ctx)

	err := cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester))
	if err != nil {
		return err
	}

	newClassInfodo := new(do.ClassInfo)
	newScDo := new(do.StudentCourse)

	_ = copier.Copy(&newClassInfodo, &newClassInfo)
	_ = copier.Copy(&newScDo, &newSc)

	errTx := cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		//添加新的课程信息
		err := cla.ClaRepo.DB.AddClassInfoToDB(ctx, newClassInfodo)
		if err != nil {
			return errcode.ErrClassUpdate
		}
		//删除原本的学生与课程的对应关系
		err = cla.Sac.DB.DeleteStudentAndCourseInDB(ctx, stuID, year, semester, []string{oldClassID})
		if err != nil {
			return errcode.ErrClassUpdate
		}
		//添加新的对应关系
		err = cla.Sac.DB.SaveStudentAndCourseToDB(ctx, newScDo)
		if err != nil {
			return errcode.ErrClassUpdate
		}
		return nil
	})
	if errTx != nil {
		logh.Errorf("Update Class [%v,%v,%v,%v,%+v,%+v] In DB  failed:%v", stuID, year, semester, oldClassID, newClassInfo, newSc, errTx)
		return errTx
	}

	go func() {
		//延迟双删
		time.AfterFunc(1*time.Second, func() {
			_ = cla.ClaRepo.Cache.DeleteClassInfoFromCache(noExpireCtx, cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester))
		})
	}()

	return nil
}

// SaveClass 保存课程[删除原本的，添加新的，主要是为了防止感知不到原本的和新增的之间有差异]
func (cla ClassRepo) SaveClass(ctx context.Context, stuID, year, semester string, classInfos []*biz.ClassInfo, scs []*biz.StudentCourse) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if len(classInfos) == 0 || len(scs) == 0 {
		return errors.New("classInfos or scs is empty")
	}

	key := cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester)

	err := cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, key)
	if err != nil {
		logh.Errorf("Delete Class [%+v] from Cache failed:%v", key, err)
		return err
	}

	defer func() {
		//延迟双删
		time.AfterFunc(1*time.Second, func() {
			_ = cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, key)
		})
	}()

	classInfosdo := make([]*do.ClassInfo, len(classInfos))
	scsdo := make([]*do.StudentCourse, len(scs))

	_ = copier.Copy(&classInfosdo, &classInfos)
	_ = copier.Copy(&scsdo, &scs)

	err = cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		//删除对应的所有关系[只删除非手动添加的]
		err = cla.Sac.DB.DeleteStudentAndCourseByTimeFromDB(ctx, stuID, year, semester)
		if err != nil {
			return err
		}
		//保存课程信息到db
		err = cla.ClaRepo.DB.SaveClassInfosToDB(ctx, classInfosdo)
		if err != nil {
			return err
		}
		//保存新的关系
		err = cla.Sac.DB.SaveManyStudentAndCourseToDB(ctx, scsdo)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logh.Errorf("Save class [%+v] and scs [%v] failed:%v", classInfos, scs, err)
	}
	return err
}

// CheckSCIdsExist 检查学生课程ID是否存在
func (cla ClassRepo) CheckSCIdsExist(ctx context.Context, stuID, year, semester, classID string) bool {
	return cla.Sac.DB.CheckExists(ctx, year, semester, stuID, classID)
}

// GetAllSchoolClassInfos 获取所有学校课程信息
func (cla ClassRepo) GetAllSchoolClassInfos(ctx context.Context, year, semester string, cursor time.Time) []*biz.ClassInfo {
	classInfos, err := cla.ClaRepo.DB.GetAllClassInfos(ctx, year, semester, cursor)
	if err != nil {
		return nil
	}

	classInfosBiz := make([]*biz.ClassInfo, len(classInfos))
	_ = copier.Copy(&classInfosBiz, &classInfos)

	return classInfosBiz
}

// GetAddedClasses 获取学生添加的课程信息
func (cla ClassRepo) GetAddedClasses(ctx context.Context, stuID, year, semester string) ([]*biz.ClassInfo, error) {
	classInfos, err := cla.ClaRepo.DB.GetAddedClassInfos(ctx, stuID, year, semester)
	if err != nil {
		return nil, err
	}

	classInfosBiz := make([]*biz.ClassInfo, len(classInfos))
	_ = copier.Copy(&classInfosBiz, &classInfos)

	return classInfosBiz, nil
}

// IsClassOfficial 检查课程是否为官方课程
func (cla ClassRepo) IsClassOfficial(ctx context.Context, stuID, year, semester, classID string) bool {
	isManuallyAddedCourse := cla.Sac.DB.CheckManualCourseStatus(ctx, stuID, year, semester, classID)
	return !isManuallyAddedCourse
}

func (cla ClassRepo) GetClassNote(ctx context.Context, stuID, year, semester, classID string) string {
	note := cla.Sac.DB.GetCourseNote(ctx, stuID, year, semester, classID)
	return note
}

// UpdateClassNote 插入课程备注
func (cla ClassRepo) UpdateClassNote(ctx context.Context, stuID, year, semester, classID, note string) error {
	logh := logger.GetLoggerFromCtx(ctx)
	err := cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester))
	if err != nil {
		return err
	}

	errTX := cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		err := cla.Sac.DB.UpdateCourseNote(ctx, stuID, year, semester, classID, note)
		if err != nil {
			return errcode.ErrClassUpdate
		}
		return nil
	})

	if errTX != nil {
		logh.Errorf("Update Class [%v,%v,%v,%v] Note %v To DB failed: %v ", stuID, year, semester, classID, note, errTX)
		return errTX
	}

	go func() {
		time.AfterFunc(1*time.Second, func() {
			_ = cla.ClaRepo.Cache.DeleteClassInfoFromCache(context.Background(), cla.Sac.Cache.GenerateClassInfosKey(stuID, year, semester))
		})
	}()

	return nil
}

func (cla ClassRepo) GetClassNatures(ctx context.Context, stuID string) []string {
	classNatures, err := cla.ClaRepo.DB.GetClassNaturesFromDB(ctx, stuID)
	if err != nil {
		return nil
	}
	if len(classNatures) == 0 {
		return nil
	}
	// 去除掉长度为0的元素
	filteredNatures := make([]string, 0, len(classNatures))
	for _, nature := range classNatures {
		if len(nature) > 0 {
			filteredNatures = append(filteredNatures, nature)
		}
	}
	return filteredNatures
}
