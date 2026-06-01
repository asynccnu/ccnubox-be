package repo

import (
	"context"
	"errors"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz"
	bizModel "github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/pkg/transaction"
	repoModel "github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
	"github.com/avast/retry-go"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/dao"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// MaxNum 每个学期最多允许添加的课程数量
const MaxNum = 20

// 一表一个 DAO/cache 结构体
// 一业务一个 repo 结构体 （面对上一层的业务）
//
// 注：该 repo 保留 log 字段，**仅用于** Cache Aside Pattern 下 best-effort 缓存失效
// 的业务决策观测日志（retry warn）。纯错误路径一律用 errorx 包装上抛。
type ClassRepo struct {
	ClaInfoDAO     *dao.ClassInfoDAO
	ClaInfoCache   *cache.ClassInfoCache
	StuCourseDAO   *dao.StudentCourseDAO
	StuCourseCache *cache.StudentCourseCache
	log            logger.Logger
}

func NewClassRepo(ClaInfoDAO *dao.ClassInfoDAO, ClaInfoCache *cache.ClassInfoCache, StuCourseDAO *dao.StudentCourseDAO, StuCourseCache *cache.StudentCourseCache, l logger.Logger) biz.ClassRepo {
	return &ClassRepo{
		ClaInfoDAO:     ClaInfoDAO,
		ClaInfoCache:   ClaInfoCache,
		StuCourseDAO:   StuCourseDAO,
		StuCourseCache: StuCourseCache,
		log:            l,
	}
}

// GetClassesFromLocal 从本地获取课程
func (cla ClassRepo) GetClassesFromLocal(ctx context.Context, stuID, year, semester string) ([]*bizModel.ClassInfoBO, error) {
	// 1. 先检查缓存
	classInfos, err := cla.ClaInfoCache.GetClassInfosFromCache(ctx, stuID, year, semester)
	cacheHit := err == nil
	// 缓存未命中走 DB 并回写缓存
	// 缓存值可能是 RedisNull（防穿透），此时 classInfos 为 nil、err 为 nil，直接走到最后返回 ErrClassNotFound
	if !cacheHit {
		classInfos, err = cla.ClaInfoDAO.GetClassInfos(ctx, stuID, year, semester)
		if err != nil {
			return nil, errorx.Errorf("repo.class.GetClassesFromLocal: stuID=%s, year=%s, semester=%s: %w",
				stuID, year, semester, err)
		}

		// 在从数据库读取数据之后，同步写入缓存
		// 如果 classInfos 是 nil 或空数据，Redis 仍然会写入一个 NULL 值，用来防止缓存穿透
		if cacheErr := cla.ClaInfoCache.AddClaInfosToCache(ctx, stuID, year, semester, classInfos); cacheErr != nil {
			// best-effort：缓存回写失败不影响返回
			cla.log.WithContext(ctx).Warnf("repo.class.GetClassesFromLocal: populate cache failed: %+v", cacheErr)
		}
	}

	if len(classInfos) == 0 {
		return nil, errorx.Errorf("repo.class.GetClassesFromLocal: stuID=%s, year=%s, semester=%s: %w",
			stuID, year, semester, biz.ErrClassNotFound)
	}

	classInfosBiz := make([]*bizModel.ClassInfoBO, 0, len(classInfos))
	for _, classInfo := range classInfos {
		if classInfo == nil {
			continue
		}
		classInfosBiz = append(classInfosBiz, classInfoDOToBO(classInfo, nil))
	}

	// 设置 metaData
	cla.fillClassMetaData(ctx, stuID, year, semester, classInfosBiz)
	return classInfosBiz, nil
}

// AddClass 添加课程信息
func (cla ClassRepo) AddClass(ctx context.Context, stuID, year, semester string, classInfo *bizModel.ClassInfoBO, sc *bizModel.StudentCourseBO) error {
	classInfoDo, scDo := classInfoBOToDO(classInfo), studentCourseBOToDO(sc)

	// Cache Aside Pattern: Update database first
	errTx := transaction.InTx(cla.ClaInfoDAO.GetDB(ctx), ctx, func(ctx context.Context) error {
		if err := cla.ClaInfoDAO.AddClassInfoToDB(ctx, classInfoDo); err != nil {
			return err
		}
		if err := cla.StuCourseDAO.SaveStudentAndCourseToDB(ctx, scDo); err != nil {
			return err
		}
		cnt, err := cla.StuCourseDAO.GetClassNum(ctx, stuID, year, semester, sc.IsManuallyAdded)
		if err != nil {
			return err
		}
		if cnt > MaxNum {
			return errorx.Errorf("repo.class.AddClass: classlist num limit, count=%d, max=%d: %w",
				cnt, MaxNum, biz.ErrClassUpdateRejected)
		}
		return nil
	})
	if errTx != nil {
		return errorx.Errorf("repo.class.AddClass: stuID=%s, year=%s, semester=%s: %w",
			stuID, year, semester, errTx)
	}

	// 缓存失效（best-effort，DB 已成功）
	cla.invalidateClassInfoCacheBestEffort(ctx, stuID, year, semester)
	return nil
}

// 防止重复加课
func (cla ClassRepo) AddedCourseExists(ctx context.Context, stuID, year, semester, classID string) bool {
	return cla.StuCourseDAO.AddedCourseExists(ctx, stuID, year, semester, classID)
}

// 批次删除手动添加的课
func (cla ClassRepo) DeleteAddedClasses(ctx context.Context, stuID, year, semester string, classIDs []string) error {
	errTx := transaction.InTx(cla.ClaInfoDAO.GetDB(ctx), ctx, func(ctx context.Context) error {
		if err := cla.StuCourseDAO.DeleteAddedStudentCourses(ctx, stuID, year, semester, classIDs); err != nil {
			return err
		}
		if err := cla.ClaInfoDAO.DeleteAddedClassInfos(ctx, classIDs); err != nil {
			return err
		}
		return nil
	})
	if errTx != nil {
		return errorx.Errorf("repo.class.DeleteAddedClasses: stuID=%s, year=%s, semester=%s, classIDs=%v: %w",
			stuID, year, semester, classIDs, errTx)
	}
	cla.invalidateClassInfoCacheBestEffort(ctx, stuID, year, semester)
	cla.invalidateMetaDataCacheBestEffort(ctx, stuID, year, semester)
	return nil
}

func (cla ClassRepo) UpdateAddedClass(ctx context.Context, stuID, year, semester, oldClassID string, classInfo *bizModel.ClassInfoBO, sc *bizModel.StudentCourseBO) error {
	classInfoDo, scDo := classInfoBOToDO(classInfo), studentCourseBOToDO(sc)

	errTx := transaction.InTx(cla.ClaInfoDAO.GetDB(ctx), ctx, func(ctx context.Context) error {
		if err := cla.ClaInfoDAO.UpsertClassInfoToDB(ctx, classInfoDo); err != nil {
			return err
		}
		if err := cla.StuCourseDAO.DeleteAddedStudentCourses(ctx, stuID, year, semester, []string{oldClassID}); err != nil {
			return err
		}
		if err := cla.StuCourseDAO.SaveStudentAndCourseToDB(ctx, scDo); err != nil {
			return err
		}
		if oldClassID != classInfo.ID {
			if err := cla.ClaInfoDAO.DeleteAddedClassInfos(ctx, []string{oldClassID}); err != nil {
				return err
			}
		}
		return nil
	})
	if errTx != nil {
		return errorx.Errorf("repo.class.UpdateAddedClass: stuID=%s, year=%s, semester=%s, oldClassID=%s, newClassID=%s: %w",
			stuID, year, semester, oldClassID, classInfo.ID, errTx)
	}
	cla.invalidateClassInfoCacheBestEffort(ctx, stuID, year, semester)
	cla.invalidateMetaDataCacheBestEffort(ctx, stuID, year, semester)
	return nil
}

func (cla ClassRepo) UpdateClassNote(ctx context.Context, stuID, year, semester, classID, note string) error {
	errTx := transaction.InTx(cla.StuCourseDAO.GetDB(ctx), ctx, func(ctx context.Context) error {
		return cla.StuCourseDAO.UpdateCourseNote(ctx, stuID, year, semester, classID, note)
	})
	if errTx != nil {
		return errorx.Errorf("repo.class.UpdateClassNote: stuID=%s, year=%s, semester=%s, classID=%s: %w",
			stuID, year, semester, classID, errTx)
	}
	cla.invalidateMetaDataCacheBestEffort(ctx, stuID, year, semester)
	return nil
}

// SaveClass 保存课程[删除原本的，添加新的，主要是为了防止感知不到原本的和新增的之间有差异]
func (cla ClassRepo) SaveClass(ctx context.Context, stuID, year, semester string, classInfos []*bizModel.ClassInfoBO, scs []*bizModel.StudentCourseBO) error {
	if len(classInfos) == 0 || len(scs) == 0 {
		return errorx.Errorf("repo.class.SaveClass: classInfos or scs is empty: %w", errors.New("empty input"))
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
		if err := cla.StuCourseDAO.DeleteStudentAndCourseByTimeFromDB(ctx, stuID, year, semester); err != nil {
			return err
		}
		if err := cla.ClaInfoDAO.SaveClassInfosToDB(ctx, classInfosdo); err != nil {
			return err
		}
		if err := cla.StuCourseDAO.SaveManyStudentAndCourseToDB(ctx, scsdo); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return errorx.Errorf("repo.class.SaveClass: stuID=%s, year=%s, semester=%s, classCount=%d: %w",
			stuID, year, semester, len(classInfos), err)
	}

	// 缓存失效（best-effort，DB 已成功）
	cla.invalidateClassInfoCacheBestEffort(ctx, stuID, year, semester)
	cla.invalidateMetaDataCacheBestEffort(ctx, stuID, year, semester)
	return nil
}

// GetAddedClasses 获取学生添加的课程信息
func (cla ClassRepo) GetAddedClasses(ctx context.Context, stuID, year, semester string) ([]*bizModel.ClassInfoBO, error) {
	classInfos, err := cla.ClaInfoDAO.GetAddedClassInfos(ctx, stuID, year, semester)
	if err != nil {
		return nil, errorx.Errorf("repo.class.GetAddedClasses: stuID=%s, year=%s, semester=%s: %w",
			stuID, year, semester, err)
	}

	classInfosBiz := make([]*bizModel.ClassInfoBO, 0, len(classInfos))
	for _, classInfo := range classInfos {
		if classInfo == nil {
			continue
		}
		classInfosBiz = append(classInfosBiz, classInfoDOToBO(classInfo, nil))
	}
	cla.fillClassMetaData(ctx, stuID, year, semester, classInfosBiz)
	return classInfosBiz, nil
}

func (cla ClassRepo) GetClassNatures(ctx context.Context, stuID string) ([]string, error) {
	classNatures, err := cla.ClaInfoDAO.GetClassNatures(ctx, stuID)
	if err != nil {
		return nil, errorx.Errorf("repo.class.GetClassNatures: stuID=%s: %w", stuID, err)
	}

	filteredNatures := make([]string, 0, len(classNatures))
	for _, nature := range classNatures {
		if nature != "" {
			filteredNatures = append(filteredNatures, nature)
		}
	}
	return filteredNatures, nil
}

func (cla ClassRepo) fillClassMetaData(ctx context.Context, stuID, year, semester string, classInfosBiz []*bizModel.ClassInfoBO) {
	if len(classInfosBiz) == 0 {
		return
	}

	claIds := make([]string, len(classInfosBiz))
	for i, classInfo := range classInfosBiz {
		claIds[i] = classInfo.ID
	}

	metaDataMap, err := cla.StuCourseCache.GetSelectClassMetaData(ctx, stuID, year, semester, claIds)
	if err != nil || len(metaDataMap) < len(classInfosBiz) {
		// 缓存未命中或不完整，从数据库补齐
		metaDataMapDO, dbErr := cla.StuCourseDAO.GetClassMetaData(ctx, stuID, year, semester, claIds)
		if dbErr != nil {
			// DB 失败：不污染缓存，MetaData 保持零值；repo 层就地 warn 作为业务降级观测点
			cla.log.WithContext(ctx).Warnf("repo.class.fillClassMetaData: stuID=%s year=%s semester=%s, GetClassMetaData failed, skip cache refill: %+v",
				stuID, year, semester, dbErr)
			return
		}

		for i := range classInfosBiz {
			if classInfosBiz[i] == nil {
				continue
			}
			if metaData, ok := metaDataMapDO[classInfosBiz[i].ID]; ok {
				classInfosBiz[i].MetaData = metaDataDOToBO(metaData)
			}
		}

		// 回写缓存（best-effort）
		cacheMetaDataMap := make(map[string]*repoModel.ClassMetaData)
		for claId, metaDataDO := range metaDataMapDO {
			cacheMetaDataMap[claId] = &metaDataDO
		}
		if len(cacheMetaDataMap) > 0 {
			if cacheErr := cla.StuCourseCache.SetAllClassMetaData(ctx, stuID, year, semester, cacheMetaDataMap); cacheErr != nil {
				cla.log.WithContext(ctx).Warnf("repo.class.fillClassMetaData: populate cache failed: %+v", cacheErr)
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

// invalidateClassInfoCacheBestEffort 缓存失效：DB 已成功，这里失败不返回错误，仅记录 warn
func (cla ClassRepo) invalidateClassInfoCacheBestEffort(ctx context.Context, stuID, year, semester string) {
	err := retry.Do(
		func() error {
			return cla.ClaInfoCache.DeleteClassInfoFromCache(ctx, stuID, year, semester)
		},
		retry.Attempts(5),
	)
	if err != nil {
		cla.log.WithContext(ctx).Warnf("repo.class.invalidateClassInfoCache: stuID=%s year=%s semester=%s: %+v",
			stuID, year, semester, err)
	}
}

// invalidateMetaDataCacheBestEffort 同上，针对 metaData 缓存
func (cla ClassRepo) invalidateMetaDataCacheBestEffort(ctx context.Context, stuID, year, semester string) {
	err := retry.Do(
		func() error {
			return cla.StuCourseCache.DeleteAllClassMetaData(ctx, stuID, year, semester)
		},
		retry.Attempts(5),
	)
	if err != nil {
		cla.log.WithContext(ctx).Warnf("repo.class.invalidateMetaDataCache: stuID=%s year=%s semester=%s: %+v",
			stuID, year, semester, err)
	}
}
