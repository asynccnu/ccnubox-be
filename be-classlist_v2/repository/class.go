package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/errcode"
	bizModel "github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/pkg/transaction"
	repoModel "github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
	"github.com/avast/retry-go"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/dao"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// MaxNum 每个学期最多允许添加的课程数量
const MaxNum = 20

// 一表一个 DAO/cache 结构体
// 一业务一个 repo 结构体 （面对上一层的业务）
type ClassRepo struct {
	ClaInfoDAO     *dao.ClassInfoDAO
	ClaInfoCache   *cache.ClassInfoCache
	StuCourseDAO   *dao.StudentCourseDAO
	StuCourseCache *cache.StudentCourseCache
}

// GetClassesFromLocal 从本地获取课程
func (cla ClassRepo) GetClassesFromLocal(ctx context.Context, stuID, year, semester string) ([]*bizModel.ClassInfoBO, error) {
	logh := logger.From(ctx)
	cacheGet := true

	// 1.先检查缓存
	classInfos, err := cla.ClaInfoCache.GetClassInfosFromCache(ctx, stuID, year, semester)
	// 如果err!=nil(err==redis.Nil)说明该ID第一次进入（redis中没有这个KEY），且未经过数据库，则允许其查数据库，所以要设置cacheGet=false
	// 如果err==nil说明其至少经过数据库了，redis中有这个KEY,但可能值为NULL，如果不为NULL，就说明缓存命中了,直接返回没有问题
	// 如果为NULL，就说明数据库中没有的数据，其依然在请求，会影响数据库（缓存穿透），我们依然直接返回
	// 这时我们就需要直接返回redis中的null，即直接返回nil,而不经过数据库
	// 该ID第一次查询，允许其查数据库
	if err != nil {
		cacheGet = false
		logh.Warnf("Get Class [%v %v %v] From Cache failed: %v", stuID, year, semester, err)
	}
	if !cacheGet {
		// 缓存未命中，从数据库中获取数据
		classInfos, err = cla.ClaInfoDAO.GetClassInfos(ctx, stuID, year, semester)
		if err != nil {
			logh.Errorf("Get Class [%v %v %v] From DB failed: %v", stuID, year, semester, err)
			return nil, errcode.ErrClassFound
		}

		// 在从数据库读取数据之后，同步写入缓存。
		// 如果 classInfos 是 nil 或空数据，Redis 仍然会写入一个 NULL 值，用来防止缓存穿透。
		if err := cla.ClaInfoCache.AddClaInfosToCache(ctx, stuID, year, semester, classInfos); err != nil {
			logh.Warnf("Failed to populate cache for [%v %v %v]: %v", stuID, year, semester, err)
			// 继续执行 —— 即使写入缓存失败，也要返回数据。
		}
	}
	// 检查classInfos是否为空
	// 如果不为空，直接返回就好
	// 如果为空，则说明没有该数据，需要去查询
	// 如果不添加此条件，即便你redis中有值为NULL的话，也不会返回错误，就导致不会去爬取更新，所以需要该条件
	// 添加该条件，能够让查询数据库的操作效率更高，同时也保证了数据的获取
	if len(classInfos) == 0 {
		return nil, errcode.ErrClassNotFound
	}

	classInfosBiz := make([]*bizModel.ClassInfoBO, 0, len(classInfos))
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

// AddClass 添加课程信息
func (cla ClassRepo) AddClass(ctx context.Context, stuID, year, semester string, classInfo *bizModel.ClassInfoBO, sc *bizModel.StudentCourseBO) error {
	logh := logger.GetLoggerFromCtx(ctx)

	// 类型转换
	classInfoDo, scDo := classInfoBOToDO(classInfo), studentCourseBOToDO(sc)

	// Cache Aside Pattern: Update database first
	errTx := transaction.InTx(cla.ClaInfoDAO.GetDB(ctx), ctx, func(ctx context.Context) error {
		if err := cla.ClaInfoDAO.AddClassInfoToDB(ctx, classInfoDo); err != nil {
			return errcode.ErrClassUpdate
		}
		// 处理 StudentCourse
		if err := cla.StuCourseDAO.SaveStudentAndCourseToDB(ctx, scDo); err != nil {
			return errcode.ErrClassUpdate
		}
		cnt, err := cla.StuCourseDAO.GetClassNum(ctx, stuID, year, semester, sc.IsManuallyAdded)
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
			return cla.ClaInfoCache.DeleteClassInfoFromCache(ctx, stuID, year, semester)
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
func (cla ClassRepo) SaveClass(ctx context.Context, stuID, year, semester string, classInfos []*bizModel.ClassInfoBO, scs []*bizModel.StudentCourseBO) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if len(classInfos) == 0 || len(scs) == 0 {
		return errors.New("classInfos or scs is empty")
	}

	classInfosdo := make([]*repoModel.ClassInfo, 0, len(classInfos))
	scsdo := make([]*repoModel.StudentCourse, 0, len(scs))

	for _, classInfo := range classInfos {
		classInfosdo = append(classInfosdo, classInfoBOToDO(classInfo))
	}
	for _, sc := range scs {
		scsdo = append(scsdo, studentCourseBOToDO(sc))
	}

	// Cache Aside Pattern: Update database first
	err := transaction.InTx(cla.ClaInfoDAO.GetDB(ctx), ctx, func(ctx context.Context) error {
		// 删除对应的所有关系[只删除官方课程]
		err := cla.StuCourseDAO.DeleteStudentAndCourseByTimeFromDB(ctx, stuID, year, semester)
		if err != nil {
			return err
		}
		// 保存课程信息到db
		err = cla.ClaInfoDAO.SaveClassInfosToDB(ctx, classInfosdo)
		if err != nil {
			return err
		}
		// 保存新的关系
		err = cla.StuCourseDAO.SaveManyStudentAndCourseToDB(ctx, scsdo)
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
			return cla.ClaInfoCache.DeleteClassInfoFromCache(ctx, stuID, year, semester)
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
			return cla.StuCourseCache.DeleteAllClassMetaData(ctx, stuID, year, semester)
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

func (cla ClassRepo) fillClassMetaData(ctx context.Context, stuID, year, semester string, classInfosBiz []*bizModel.ClassInfoBO) {
	if len(classInfosBiz) == 0 {
		return
	}
	logh := logger.From(ctx)

	// 收集所有需要查询的claIds
	claIds := make([]string, len(classInfosBiz))
	for i, classInfo := range classInfosBiz {
		claIds[i] = classInfo.ID
	}

	// 尝试从缓存获取指定课程的元数据
	metaDataMap, err := cla.StuCourseCache.GetSelectClassMetaData(ctx, stuID, year, semester, claIds)
	if err != nil || len(metaDataMap) < len(classInfosBiz) {
		logh.Warnf("Get ClassMetaData from cache failed: %v", err)
		// 缓存未命中，从数据库获取
		metaDataMapDO := cla.StuCourseDAO.GetClassMetaData(ctx, stuID, year, semester, claIds)

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
		cacheMetaDataMap := make(map[string]*repoModel.ClassMetaData)
		for claId, metaDataDO := range metaDataMapDO {
			cacheMetaDataMap[claId] = &metaDataDO
		}
		if len(cacheMetaDataMap) > 0 {
			if err := cla.StuCourseCache.SetAllClassMetaData(ctx, stuID, year, semester, cacheMetaDataMap); err != nil {
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
