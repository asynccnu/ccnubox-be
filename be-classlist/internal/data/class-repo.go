package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/avast/retry-go"
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
func (cla ClassRepo) GetClassesFromLocal(ctx context.Context, stuID, year, semester string) ([]*biz.ClassInfoBO, error) {
	logh := logger.GetLoggerFromCtx(ctx)

	var (
		cacheGet = true
	)

	// Cache Aside Pattern: Check cache first
	classInfos, err := cla.ClaRepo.Cache.GetClassInfosFromCache(ctx, stuID, year, semester)
	//如果err!=nil(err==redis.Nil)说明该ID第一次进入（redis中没有这个KEY），且未经过数据库，则允许其查数据库，所以要设置cacheGet=false
	//如果err==nil说明其至少经过数据库了，redis中有这个KEY,但可能值为NULL，如果不为NULL，就说明缓存命中了,直接返回没有问题
	//如果为NULL，就说明数据库中没有的数据，其依然在请求，会影响数据库（缓存穿透），我们依然直接返回
	//这时我们就需要直接返回redis中的null，即直接返回nil,而不经过数据库

	if err != nil {
		cacheGet = false
		logh.Warnf("Get Class [%v %v %v] From Cache failed: %v", stuID, year, semester, err)
	}
	if !cacheGet {
		// Cache miss: Load from database
		classInfos, err = cla.ClaRepo.DB.GetClassInfos(ctx, stuID, year, semester)
		if err != nil {
			logh.Errorf("Get Class [%v %v %v] From DB failed: %v", stuID, year, semester, err)
			return nil, errcode.ErrClassFound
		}

		// Populate cache synchronously after database read
		// Note: If classInfos is nil/empty, redis will still set the key-value with NULL value to prevent cache penetration
		if err := cla.ClaRepo.Cache.AddClaInfosToCache(ctx, stuID, year, semester, classInfos); err != nil {
			logh.Warnf("Failed to populate cache for [%v %v %v]: %v", stuID, year, semester, err)
			// Continue - return data even if cache population fails
		}
	}
	//检查classInfos是否为空
	//如果不为空，直接返回就好
	//如果为空，则说明没有该数据，需要去查询
	//如果不添加此条件，即便你redis中有值为NULL的话，也不会返回错误，就导致不会去爬取更新，所以需要该条件
	//添加该条件，能够让查询数据库的操作效率更高，同时也保证了数据的获取
	if len(classInfos) == 0 {
		return nil, errcode.ErrClassNotFound
	}

	classInfosBiz := make([]*biz.ClassInfoBO, 0, len(classInfos))
	for _, classInfo := range classInfos {
		if classInfo == nil {
			continue
		}
		classInfosBiz = append(classInfosBiz, classInfoDOToBO(classInfo, nil))
	}

	// 设置metaData
	cla.fillClassMetaData(ctx, stuID, year, semester, classInfosBiz)
	return classInfosBiz, nil
}

func (cla ClassRepo) CacheClass(ctx context.Context, stuID, year, semester string) {
	logh := logger.GetLoggerFromCtx(ctx)
	classInfos, err := cla.ClaRepo.DB.GetClassInfos(ctx, stuID, year, semester)
	if err != nil {
		logh.Errorf("Get Class [%v %v %v] From DB failed: %v", stuID, year, semester, err)
		return
	}
	if err := cla.ClaRepo.Cache.AddClaInfosToCache(ctx, stuID, year, semester, classInfos); err != nil {
		logh.Warnf("Failed to populate cache for [%v %v %v]: %v", stuID, year, semester, err)
	} else {
		// 添加缓存失败，就尝试删除
		if err := cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, stuID, year, semester); err != nil {
			logh.Warnf("Failed to delete cache for [%v %v %v]: %v", stuID, year, semester, err)
		}
	}

	claIds := make([]string, len(classInfos))
	for i, classInfo := range classInfos {
		claIds[i] = classInfo.ID
	}

	metaDataMapDO := cla.Sac.DB.GetClassMetaData(ctx, stuID, year, semester, claIds)
	if len(metaDataMapDO) == 0 {
		// 如果获取为空，可能是失败
		// 删除缓存
		if err := cla.Sac.Cache.DeleteAllClassMetaData(ctx, stuID, year, semester); err != nil {
			logh.Warnf("Failed to delete all class meta data cache for [%v %v %v]: %v", stuID, year, semester, err)
		}
	} else {
		// 将数据库查询结果写入缓存
		cacheMetaDataMap := make(map[string]*ClassMetaData)
		for claId, metaDataDO := range metaDataMapDO {
			cacheMetaDataMap[claId] = &metaDataDO
		}
		if err := cla.Sac.Cache.SetAllClassMetaData(ctx, stuID, year, semester, cacheMetaDataMap); err != nil {
			logh.Warnf("Failed to cache all ClassMetaData for [%v %v %v]: %v", stuID, year, semester, err)
		}
	}
}

// fillClassMetaData 填充课程元数据
func (cla ClassRepo) fillClassMetaData(ctx context.Context, stuID, year, semester string, classInfosBiz []*biz.ClassInfoBO) {
	if len(classInfosBiz) == 0 {
		return
	}
	logh := logger.GetLoggerFromCtx(ctx)

	// 收集所有需要查询的claIds
	claIds := make([]string, len(classInfosBiz))
	for i, classInfo := range classInfosBiz {
		claIds[i] = classInfo.ID
	}

	// 尝试从缓存获取指定课程的元数据
	metaDataMap, err := cla.Sac.Cache.GetSelectClassMetaData(ctx, stuID, year, semester, claIds)
	if err != nil || len(metaDataMap) < len(classInfosBiz) {
		logh.Warnf("Get ClassMetaData from cache failed: %v", err)
		// 缓存未命中，从数据库获取
		metaDataMapDO := cla.Sac.DB.GetClassMetaData(ctx, stuID, year, semester, claIds)

		// 填充到classInfosBiz
		for i := range classInfosBiz {
			if classInfosBiz[i] == nil {
				continue
			}
			if metaData, ok := metaDataMapDO[classInfosBiz[i].ID]; ok {
				classInfosBiz[i].MetaData = metaDataDOToBO(metaData)
			}
		}

		// 将数据库查询结果写入缓存
		cacheMetaDataMap := make(map[string]*ClassMetaData)
		for claId, metaDataDO := range metaDataMapDO {
			cacheMetaDataMap[claId] = &metaDataDO
		}
		if len(cacheMetaDataMap) > 0 {
			if err := cla.Sac.Cache.SetAllClassMetaData(ctx, stuID, year, semester, cacheMetaDataMap); err != nil {
				logh.Warnf("Failed to cache ClassMetaData: %v", err)
			}
		}
		return
	}

	// 缓存命中，直接使用缓存数据
	for i := range classInfosBiz {
		if metaData, ok := metaDataMap[classInfosBiz[i].ID]; ok {
			classInfosBiz[i].MetaData = metaDataDOToBO(*metaData)
		}
	}
}

// GetSpecificClassInfo 获取特定课程信息
func (cla ClassRepo) GetSpecificClassInfo(ctx context.Context, stuID, year, semester, classID string) (*biz.ClassInfoBO, error) {
	classInfo, err := cla.ClaRepo.DB.GetClassInfoFromDB(ctx, classID)
	if err != nil || classInfo == nil {
		return nil, errcode.ErrClassNotFound
	}

	//将ClassInfo转换为biz.ClassInfoBO
	classInfoBiz := classInfoDOToBO(classInfo, nil)
	// 设置metaData
	cla.fillClassMetaData(ctx, stuID, year, semester, []*biz.ClassInfoBO{classInfoBiz})
	return classInfoBiz, nil
}

// AddClass 添加课程信息
func (cla ClassRepo) AddClass(ctx context.Context, stuID, year, semester string, classInfo *biz.ClassInfoBO, sc *biz.StudentCourse) error {
	logh := logger.GetLoggerFromCtx(ctx)

	// 类型转换
	classInfoDo, scDo := classInfoBOToDO(classInfo), studentCourseBOToDO(sc)

	// Cache Aside Pattern: Update database first
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

	// Invalidate cache synchronously after successful transaction with retry
	err := retry.Do(
		func() error {
			return cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, stuID, year, semester)
		},
		retry.Attempts(5),
		retry.OnRetry(func(n uint, err error) {
			logh.Warnf("Retry %d: Failed to invalidate cache for [%v %v %v]: %v", n+1, stuID, year, semester, err)
		}),
	)
	if err != nil {
		logh.Warnf("Failed to invalidate cache after retries for [%v %v %v]: %v", stuID, year, semester, err)
		// Don't return error - database write succeeded
	}

	return nil
}

// DeleteClass 删除课程信息（仅从本地删除，不处理回收站）
func (cla ClassRepo) DeleteClass(ctx context.Context, stuID, year, semester string, classInfo *biz.ClassInfoBO) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if classInfo == nil {
		return errcode.ErrClassNotFound
	}

	// Cache Aside Pattern: Update database first
	errTx := cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		err := cla.Sac.DB.DeleteStudentAndCourseInDB(ctx, stuID, year, semester, classInfo.ID)
		if err != nil {
			return fmt.Errorf("error deleting student: %w", err)
		}
		return nil
	})
	if errTx != nil {
		logh.Errorf("Delete Class [%v,%v,%v,%v] In DB failed:%v", stuID, year, semester, classInfo.ID, errTx)
		return errTx
	}

	// Invalidate cache synchronously after successful transaction with retry
	err := retry.Do(
		func() error {
			return cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, stuID, year, semester)
		},
		retry.Attempts(5),
		retry.OnRetry(func(n uint, err error) {
			logh.Warnf("Retry %d: Failed to invalidate cache for [%v %v %v]: %v", n+1, stuID, year, semester, err)
		}),
	)
	if err != nil {
		logh.Warnf("Failed to invalidate cache after retries for [%v %v %v]: %v", stuID, year, semester, err)
		// Don't return error - database write succeeded
	}

	// 删除metaData的缓存
	err = retry.Do(
		func() error {
			return cla.Sac.Cache.DeleteClassMetaData(ctx, stuID, classInfo.ID, year, semester)
		},
		retry.Attempts(5),
		retry.OnRetry(func(n uint, err error) {
			logh.Warnf("Retry %d: Failed to invalidate cache for [%v %v %v]: %v", n+1, stuID, year, semester, err)
		}),
	)
	if err != nil {
		logh.Warnf("Failed to invalidate cache after retries for [%v %v %v]: %v", stuID, year, semester, err)
		// Don't return error - database write succeeded
	}

	return nil
}

// UpdateClass 更新课程信息
func (cla ClassRepo) UpdateClass(ctx context.Context, stuID, year, semester, oldClassID string,
	newClassInfo *biz.ClassInfoBO, newSc *biz.StudentCourse) error {

	logh := logger.GetLoggerFromCtx(ctx)

	newClassInfodo, newScDo := classInfoBOToDO(newClassInfo), studentCourseBOToDO(newSc)

	// Cache Aside Pattern: Update database first
	errTx := cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		//添加新的课程信息
		err := cla.ClaRepo.DB.AddClassInfoToDB(ctx, newClassInfodo)
		if err != nil {
			return errcode.ErrClassUpdate
		}
		//删除原本的学生与课程的对应关系
		err = cla.Sac.DB.DeleteStudentAndCourseInDB(ctx, stuID, year, semester, oldClassID)
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

	// Invalidate cache synchronously after successful transaction with retry
	err := retry.Do(
		func() error {
			return cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, stuID, year, semester)
		},
		retry.Attempts(5),
		retry.OnRetry(func(n uint, err error) {
			logh.Warnf("Retry %d: Failed to invalidate cache for [%v %v %v]: %v", n+1, stuID, year, semester, err)
		}),
	)
	if err != nil {
		logh.Warnf("Failed to invalidate cache after retries for [%v %v %v]: %v", stuID, year, semester, err)
		// Don't return error - database write succeeded
	}

	// 删除metaData的缓存
	err = retry.Do(
		func() error {
			return cla.Sac.Cache.DeleteClassMetaData(ctx, stuID, oldClassID, year, semester)
		},
		retry.Attempts(5),
		retry.OnRetry(func(n uint, err error) {
			logh.Warnf("Retry %d: Failed to invalidate cache for [%v %v %v]: %v", n+1, stuID, year, semester, err)
		}),
	)
	if err != nil {
		logh.Warnf("Failed to invalidate cache after retries for [%v %v %v]: %v", stuID, year, semester, err)
		// Don't return error - database write succeeded
	}

	return nil
}

// SaveClass 保存课程[删除原本的，添加新的，主要是为了防止感知不到原本的和新增的之间有差异]
func (cla ClassRepo) SaveClass(ctx context.Context, stuID, year, semester string, classInfos []*biz.ClassInfoBO, scs []*biz.StudentCourse) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if len(classInfos) == 0 || len(scs) == 0 {
		return errors.New("classInfos or scs is empty")
	}

	classInfosdo := make([]*ClassInfo, 0, len(classInfos))
	scsdo := make([]*StudentCourse, 0, len(scs))

	for _, classInfo := range classInfos {
		classInfosdo = append(classInfosdo, classInfoBOToDO(classInfo))
	}
	for _, sc := range scs {
		scsdo = append(scsdo, studentCourseBOToDO(sc))
	}

	// Cache Aside Pattern: Update database first
	err := cla.TxCtrl.InTx(ctx, func(ctx context.Context) error {
		//删除对应的所有关系[只删除官方课程]
		err := cla.Sac.DB.DeleteStudentAndCourseByTimeFromDB(ctx, stuID, year, semester)
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
		return err
	}

	// Invalidate cache synchronously after successful transaction with retry
	err = retry.Do(
		func() error {
			return cla.ClaRepo.Cache.DeleteClassInfoFromCache(ctx, stuID, year, semester)
		},
		retry.Attempts(5),
		retry.OnRetry(func(n uint, err error) {
			logh.Warnf("Retry %d: Failed to invalidate cache for [%v %v %v]: %v", n+1, stuID, year, semester, err)
		}),
	)
	if err != nil {
		logh.Warnf("Failed to invalidate cache after retries for [%v %v %v]: %v", stuID, year, semester, err)
		// Don't return error - database write succeeded
	}

	// 删除metaData的缓存
	err = retry.Do(
		func() error {
			return cla.Sac.Cache.DeleteAllClassMetaData(ctx, stuID, year, semester)
		},
		retry.Attempts(5),
		retry.OnRetry(func(n uint, err error) {
			logh.Warnf("Retry %d: Failed to invalidate cache for [%v %v %v]: %v", n+1, stuID, year, semester, err)
		}),
	)
	if err != nil {
		logh.Warnf("Failed to invalidate cache after retries for [%v %v %v]: %v", stuID, year, semester, err)
		// Don't return error - database write succeeded
	}

	return nil
}

// CheckSCIdsExist 检查学生课程ID是否存在
func (cla ClassRepo) CheckSCIdsExist(ctx context.Context, stuID, year, semester, classID string) bool {
	return cla.Sac.DB.CheckExists(ctx, year, semester, stuID, classID)
}

// GetAllSchoolClassInfos 获取所有学校课程信息
func (cla ClassRepo) GetAllSchoolClassInfos(ctx context.Context, year, semester string, cursor time.Time) []*biz.ClassInfoBO {
	classInfos, err := cla.ClaRepo.DB.GetAllClassInfos(ctx, year, semester, cursor)
	if err != nil {
		return nil
	}

	classInfosBiz := make([]*biz.ClassInfoBO, 0, len(classInfos))
	for _, classInfo := range classInfos {
		classInfosBiz = append(classInfosBiz, classInfoDOToBO(classInfo, nil))
	}
	return classInfosBiz
}

// GetAddedClasses 获取学生添加的课程信息
func (cla ClassRepo) GetAddedClasses(ctx context.Context, stuID, year, semester string) ([]*biz.ClassInfoBO, error) {
	classInfos, err := cla.ClaRepo.DB.GetAddedClassInfos(ctx, stuID, year, semester)
	if err != nil {
		return nil, err
	}

	classInfosBiz := make([]*biz.ClassInfoBO, 0, len(classInfos))
	for _, classInfo := range classInfos {
		if classInfo == nil {
			continue
		}

		classInfosBiz = append(classInfosBiz, classInfoDOToBO(classInfo, nil))
	}
	cla.fillClassMetaData(ctx, stuID, year, semester, classInfosBiz)
	return classInfosBiz, nil
}

// GetClassMetaData 获取单个课程元信息
func (cla ClassRepo) GetClassMetaData(ctx context.Context, stuID, year, semester, classID string) (biz.ClassMetaDataBO, error) {
	logh := logger.GetLoggerFromCtx(ctx)

	// 尝试从缓存获取单个元数据
	metaDataCache, err := cla.Sac.Cache.GetClassMetaData(ctx, stuID, classID, year, semester)
	if err != nil {
		logh.Warnf("Get ClassMetaData from cache failed: %v", err)
		// 缓存未命中，从数据库获取
		metaDataMapDO := cla.Sac.DB.GetClassMetaData(ctx, stuID, year, semester, []string{classID})
		if len(metaDataMapDO) == 0 {
			return biz.ClassMetaDataBO{}, fmt.Errorf("metadata not found")
		}

		metaDataDO, ok := metaDataMapDO[classID]
		if !ok {
			return biz.ClassMetaDataBO{}, fmt.Errorf("metadata not found for classID: %s", classID)
		}

		// 将数据库查询结果写入缓存
		if err := cla.Sac.Cache.SetClassMetaData(ctx, stuID, classID, year, semester, &metaDataDO); err != nil {
			logh.Warnf("Failed to cache ClassMetaData: %v", err)
		}

		return metaDataDOToBO(metaDataDO), nil
	}

	// 缓存命中，直接返回
	return metaDataDOToBO(*metaDataCache), nil
}

// UpdateClassNote 插入课程备注
func (cla ClassRepo) UpdateClassNote(ctx context.Context, stuID, year, semester, classID, note string) error {
	logh := logger.GetLoggerFromCtx(ctx)

	// Cache Aside Pattern: Update database first
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

	// Invalidate metadata cache for this specific class
	err := retry.Do(
		func() error {
			return cla.Sac.Cache.DeleteClassMetaData(ctx, stuID, classID, year, semester)
		},
		retry.Attempts(5),
		retry.OnRetry(func(n uint, err error) {
			logh.Warnf("Retry %d: Failed to invalidate metadata cache for [%v %v %v %v]: %v", n+1, stuID, year, semester, classID, err)
		}),
	)
	if err != nil {
		logh.Warnf("Failed to invalidate metadata cache after retries for [%v %v %v %v]: %v", stuID, year, semester, classID, err)
		// Don't return error - database write succeeded
	}

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

func (cla ClassRepo) GetStudentIDs(ctx context.Context, lastStuID string, size int) ([]string, error) {
	return cla.Sac.DB.GetStudentIDs(ctx, lastStuID, size)
}
