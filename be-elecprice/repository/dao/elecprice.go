package dao

import (
	"context"
	"errors"

	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
)

// ElecpriceDAO 数据库操作的集合
type ElecpriceDAO interface {
	FindAll(ctx context.Context, studengId string) ([]model.ElecpriceConfig, error)
	Delete(ctx context.Context, studentId string, roomId string) error
	GetConfigsByCursor(ctx context.Context, lastID int64, limit int) ([]model.ElecpriceConfig, int64, error)
	IsNotFoundError(err error) bool
	Upsert(ctx context.Context, studentId string, roomId string, ec *model.ElecpriceConfig) error
}

type elecpriceDAO struct {
	db *gorm.DB
}

// NewElecpriceDAO 构建数据库操作实例
func NewElecpriceDAO(db *gorm.DB) ElecpriceDAO {
	return &elecpriceDAO{db: db}
}

func (d *elecpriceDAO) FindAll(ctx context.Context, studentId string) ([]model.ElecpriceConfig, error) {
	var configs []model.ElecpriceConfig
	err := d.db.WithContext(ctx).Where("student_id = ?", studentId).Find(&configs).Error
	if err != nil {
		return nil, errorx.Errorf("dao: find all configs failed, student_id: %s, err: %w", studentId, err)
	}

	return configs, nil
}

func (d *elecpriceDAO) GetConfigsByCursor(ctx context.Context, lastID int64, limit int) ([]model.ElecpriceConfig, int64, error) {
	// 分页查询数据
	var configs []model.ElecpriceConfig
	query := d.db.WithContext(ctx).
		Model(model.ElecpriceConfig{}).
		Order("id ASC"). // 按 id 排序，确保数据有序
		Limit(limit)

	// 如果提供了游标（lastID），则从该游标之后开始查询
	if lastID != -1 {
		query = query.Where("id > ?", lastID)
	}

	err := query.Scan(&configs).Error
	if err != nil {
		return nil, -1, errorx.Errorf("dao: scan configs by cursor failed, lastID: %d, limit: %d, err: %w", lastID, limit, err)
	}

	// 如果没有数据，直接返回
	if len(configs) == 0 {
		return nil, -1, nil
	}

	return configs, configs[len(configs)-1].ID, nil
}

func (d *elecpriceDAO) IsNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func (d *elecpriceDAO) Delete(ctx context.Context, studentId string, roomId string) error {
	err := d.db.WithContext(ctx).Where("target_id = ? and student_id = ?", roomId, studentId).Delete(&model.ElecpriceConfig{}).Error
	if err != nil {
		return errorx.Errorf("dao: delete config failed, student_id: %s, room_id: %s, err: %w", studentId, roomId, err)
	}
	return nil
}

func (d *elecpriceDAO) Upsert(ctx context.Context, studentId string, roomId string, ec *model.ElecpriceConfig) error {
	var old model.ElecpriceConfig
	// 原有逻辑：First 错误未处理，依赖 old 零值判断
	d.db.Where("student_id = ? and target_id = ?", studentId, roomId).First(&old)

	if old.RoomName == ec.RoomName {
		err := d.db.Model(&old).Updates(ec).Error
		if err != nil {
			return errorx.Errorf("dao: upsert(update) config failed, student_id: %s, room_id: %s, err: %w", studentId, roomId, err)
		}
		return nil
	}

	err := d.db.Create(ec).Error
	if err != nil {
		return errorx.Errorf("dao: upsert(create) config failed, student_id: %s, room_id: %s, err: %w", studentId, roomId, err)
	}

	return nil
}
