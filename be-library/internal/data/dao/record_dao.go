package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-library/internal/data/DO"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RecordDAO struct {
	db *gorm.DB
}

func NewRecordDAO(db *gorm.DB) *RecordDAO {
	return &RecordDAO{db: db}
}

func (d *RecordDAO) SyncFutureRecords(ctx context.Context, stuID string, records []DO.FutureRecord) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("stu_id = ?", stuID).Delete(&DO.FutureRecord{}).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		if err := tx.Create(&records).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *RecordDAO) ListFutureRecords(ctx context.Context, stuID string) ([]DO.FutureRecord, error) {
	var records []DO.FutureRecord
	err := d.db.WithContext(ctx).
		Where("stu_id = ?", stuID).
		Order("start DESC").
		Find(&records).Error
	return records, err
}

func (d *RecordDAO) UpsertHistoryRecords(ctx context.Context, records []DO.HistoryRecord) error {
	if len(records) == 0 {
		return nil
	}

	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "stu_id"},
				{Name: "submit_time"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"place", "floor", "status", "date",
			}),
		}).
		Create(&records).Error
}

func (d *RecordDAO) ListHistoryRecords(ctx context.Context, stuID string) ([]DO.HistoryRecord, error) {
	var records []DO.HistoryRecord
	err := d.db.WithContext(ctx).
		Where("stu_id = ?", stuID).
		Order("submit_time DESC").
		Find(&records).Error
	return records, err
}
