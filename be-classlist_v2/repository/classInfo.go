package repo

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/dao"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// 上层编排
type ClassRepo struct {
	ClaRepo *ClassInfoRepo
	Sac     *StudentCourseRepo
}

// 操作课程表
type ClassInfoRepo struct {
	DB    *dao.ClassInfoDBRepo
	Cache *cache.ClassInfoCacheRepo
}

// 操作学生-课程关联表
// 通过 ID 来构成联系
type StudentCourseRepo struct {
	DB    *dao.StudentCourseDBRepo
	Cache *cache.StudentCourseCacheRepo
}

// GetClassesFromLocal 从本地获取课程
func (cla ClassRepo) GetClassesFromLocal(ctx context.Context, stuID, year, semester string) ([]*biz.ClassInfoBO, error) {
	logh := logger.From(ctx)

	cacheGet := true

	// Cache Aside Pattern: Check cache first
	classInfos, err := cla.ClaRepo.Cache.GetClassInfosFromCache(ctx, stuID, year, semester)
	// 如果err!=nil(err==redis.Nil)说明该ID第一次进入（redis中没有这个KEY），且未经过数据库，则允许其查数据库，所以要设置cacheGet=false
	// 如果err==nil说明其至少经过数据库了，redis中有这个KEY,但可能值为NULL，如果不为NULL，就说明缓存命中了,直接返回没有问题
	// 如果为NULL，就说明数据库中没有的数据，其依然在请求，会影响数据库（缓存穿透），我们依然直接返回
	// 这时我们就需要直接返回redis中的null，即直接返回nil,而不经过数据库
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
	// 检查classInfos是否为空
	// 如果不为空，直接返回就好
	// 如果为空，则说明没有该数据，需要去查询
	// 如果不添加此条件，即便你redis中有值为NULL的话，也不会返回错误，就导致不会去爬取更新，所以需要该条件
	// 添加该条件，能够让查询数据库的操作效率更高，同时也保证了数据的获取
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
