package dao

import (
	"context"
	"errors"

	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FeedUserConfigDAO 用来对用户的feed数据进行处理
type FeedUserConfigDAO interface {
	FindOrCreateUserFeedConfig(ctx context.Context, studentId string) (*model.FeedUserConfig, error)
	SaveUserFeedConfig(ctx context.Context, req *model.FeedUserConfig) error
	SetConfigBit(config *uint16, position int)
	ClearConfigBit(config *uint16, position int)
	GetConfigBit(config uint16, position int) bool
	GetStudentIdsByCursor(ctx context.Context, lastID int64, limit int) ([]string, int64, error)
	ChangeConfigBits(ctx context.Context, studentID string, bits map[int]bool, library *bool) (*model.FeedUserConfig, error)
	IsLibraryEnabled(ctx context.Context, studentID string) (bool, error)
	ListLibraryPreferenceChanges(ctx context.Context, afterRevision int64, limit int) ([]model.FeedUserConfigChange, error)
	LatestLibraryPreferenceRevision(ctx context.Context) (int64, error)
	ListLibraryReminderUsers(ctx context.Context, afterID, snapshotRevision int64, limit int) ([]model.FeedUserConfig, error)
}

type feedUserConfigDAO struct {
	gorm *gorm.DB
}

// NewFeedUserConfigDAO 创建一个新的 FeedUserConfigDAO 实例
func NewFeedUserConfigDAO(db *gorm.DB) FeedUserConfigDAO {
	return &feedUserConfigDAO{gorm: db}
}

// FindOrCreateUserFeedConfig 查找或创建 FeedUserConfig
func (dao *feedUserConfigDAO) FindOrCreateUserFeedConfig(ctx context.Context, studentId string) (*model.FeedUserConfig, error) {
	if !tool.IsValidStudentID(studentId) {
		return nil, errorx.New("dao: invalid student id for feed config")
	}
	var allowList model.FeedUserConfig
	err := dao.gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("student_id = ?", studentId).
			First(&allowList).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		allowList = model.FeedUserConfig{
			StudentId:  studentId,
			PushConfig: model.DefaultPushConfig,
		}
		return createUserFeedConfig(tx, &allowList)
	})
	if err != nil {
		return nil, errorx.Errorf("dao: find or create user feed config failed, sid: %s, err: %w", studentId, err)
	}
	return &allowList, nil
}

// SaveUserFeedConfig 保存 FeedUserConfig
func (dao *feedUserConfigDAO) SaveUserFeedConfig(ctx context.Context, req *model.FeedUserConfig) error {
	if req == nil || !tool.IsValidStudentID(req.StudentId) {
		return errorx.New("dao: invalid student id for feed config")
	}
	err := dao.gorm.WithContext(ctx).Save(req).Error
	if err != nil {
		return errorx.Errorf("dao: save user feed config failed, sid: %s, err: %w", req.StudentId, err)
	}
	return nil
}

// 设置指定位置的配置为 1
func (dao *feedUserConfigDAO) SetConfigBit(config *uint16, position int) {
	*config |= (1 << position)
}

// 设置指定位置的配置为 0
func (dao *feedUserConfigDAO) ClearConfigBit(config *uint16, position int) {
	*config &= ^(1 << position)
}

// 获取指定位置的配置值（返回 true 或 false）
func (dao *feedUserConfigDAO) GetConfigBit(config uint16, position int) bool {
	return (config & (1 << position)) != 0
}

func (dao *feedUserConfigDAO) GetStudentIdsByCursor(ctx context.Context, lastID int64, limit int) ([]string, int64, error) {
	var students []struct {
		ID        int64  `gorm:"column:id"`
		StudentId string `gorm:"column:student_id"`
	}

	query := dao.gorm.WithContext(ctx).Model(model.FeedUserConfig{}).
		Where("id > ?", lastID).
		Order("id ASC").
		Limit(limit)

	if err := query.Find(&students).Error; err != nil {
		return nil, 0, errorx.Errorf("dao: get student ids by cursor failed, lastID: %d, limit: %d, err: %w", lastID, limit, err)
	}

	if len(students) == 0 {
		return nil, 0, nil
	}

	var studentIds []string
	for _, student := range students {
		studentIds = append(studentIds, student.StudentId)
	}

	newLastID := students[len(students)-1].ID

	return studentIds, newLastID, nil
}

// 事务中修改 AllowList 的位图
func (dao *feedUserConfigDAO) ChangeConfigBits(
	ctx context.Context,
	studentID string,
	bits map[int]bool,
	library *bool,
) (*model.FeedUserConfig, error) {
	if !tool.IsValidStudentID(studentID) {
		return nil, errorx.New("dao: invalid student id for feed config")
	}
	var config model.FeedUserConfig
	err := dao.gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("student_id = ?", studentID).
			First(&config).Error
		isNew := errors.Is(err, gorm.ErrRecordNotFound)
		if err != nil && !isNew {
			return err
		}
		if isNew {
			config = model.FeedUserConfig{
				StudentId:  studentID,
				PushConfig: model.DefaultPushConfig,
			}
		}

		for position, enabled := range bits {
			if enabled {
				config.PushConfig |= 1 << position
			} else {
				config.PushConfig &^= 1 << position
			}
		}

		libraryChanged := false
		if library != nil {
			wasEnabled := config.PushConfig&(1<<model.LibraryPos) != 0
			libraryChanged = wasEnabled != *library
			if *library {
				config.PushConfig |= 1 << model.LibraryPos
			} else {
				config.PushConfig &^= 1 << model.LibraryPos
			}
		}

		if isNew {
			return createUserFeedConfig(tx, &config)
		}
		if libraryChanged {
			revision, err := allocateLibraryPreferenceRevision(tx)
			if err != nil {
				return err
			}
			change := model.FeedUserConfigChange{
				Revision:       revision,
				StudentId:      studentID,
				LibraryEnabled: *library,
			}
			if err := tx.Create(&change).Error; err != nil {
				return err
			}
			config.LibraryRevision = revision
		}
		return tx.Save(&config).Error
	})
	if err != nil {
		return nil, errorx.Errorf("dao: change feed config transaction failed, sid: %s, err: %w", studentID, err)
	}
	return &config, nil
}

// 新建配置时同时记录初始图书馆偏好，保证后续增量同步不会遗漏用户。
func createUserFeedConfig(tx *gorm.DB, config *model.FeedUserConfig) error {
	revision, err := allocateLibraryPreferenceRevision(tx)
	if err != nil {
		return err
	}
	config.LibraryRevision = revision
	if err = tx.Create(config).Error; err != nil {
		return err
	}
	return tx.Create(&model.FeedUserConfigChange{
		Revision:       revision,
		StudentId:      config.StudentId,
		LibraryEnabled: config.PushConfig&(1<<model.LibraryPos) != 0,
	}).Error
}

// 分配器行锁会一直持有到事务提交，保证版本号顺序与提交顺序一致。
func allocateLibraryPreferenceRevision(tx *gorm.DB) (int64, error) {
	var allocator model.FeedUserConfigRevisionAllocator
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&allocator, model.FeedUserConfigRevisionAllocatorID).Error; err != nil {
		return 0, err
	}
	allocator.Revision++
	if err := tx.Model(&allocator).UpdateColumn("revision", allocator.Revision).Error; err != nil {
		return 0, err
	}
	return allocator.Revision, nil
}

func (dao *feedUserConfigDAO) IsLibraryEnabled(ctx context.Context, studentID string) (bool, error) {
	var config model.FeedUserConfig
	err := dao.gorm.WithContext(ctx).Where("student_id = ?", studentID).First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, errorx.Errorf("dao: get library preference failed, sid: %s, err: %w", studentID, err)
	}
	return config.PushConfig&(1<<model.LibraryPos) != 0, nil
}

func (dao *feedUserConfigDAO) ListLibraryPreferenceChanges(ctx context.Context, afterRevision int64, limit int) ([]model.FeedUserConfigChange, error) {
	var changes []model.FeedUserConfigChange
	err := dao.gorm.WithContext(ctx).
		Where("revision > ?", afterRevision).
		Order("revision ASC").
		Limit(limit).
		Find(&changes).Error
	if err != nil {
		return nil, errorx.Errorf("dao: list library preference changes failed, revision: %d, err: %w", afterRevision, err)
	}
	return changes, nil
}

// 全量分页开始前固定变更水位，水位之后的用户由增量同步重放。
func (dao *feedUserConfigDAO) LatestLibraryPreferenceRevision(ctx context.Context) (int64, error) {
	var allocator model.FeedUserConfigRevisionAllocator
	err := dao.gorm.WithContext(ctx).
		First(&allocator, model.FeedUserConfigRevisionAllocatorID).Error
	if err != nil {
		return 0, errorx.Errorf("dao: get latest library preference revision failed, err: %w", err)
	}
	return allocator.Revision, nil
}

// 只返回水位内未发生后续变更的用户，避免实时 enabled 集合在翻页期间前移。
func (dao *feedUserConfigDAO) ListLibraryReminderUsers(ctx context.Context, afterID, snapshotRevision int64, limit int) ([]model.FeedUserConfig, error) {
	var configs []model.FeedUserConfig
	err := dao.gorm.WithContext(ctx).
		Where("id > ? AND library_revision <= ? AND (push_config & ?) <> 0", afterID, snapshotRevision, uint16(1<<model.LibraryPos)).
		Order("id ASC").
		Limit(limit).
		Find(&configs).Error
	if err != nil {
		return nil, errorx.Errorf("dao: list library reminder users failed, id: %d, snapshot revision: %d, err: %w", afterID, snapshotRevision, err)
	}
	return configs, nil
}
