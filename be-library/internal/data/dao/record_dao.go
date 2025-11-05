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

func (d *RecordDAO) UpsertFutureRecords(ctx context.Context, records []DO.FutureRecord) error {
	if len(records) == 0 {
		return nil
	}

	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "stu_id"},
				{Name: "start"},
				{Name: "end"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"remote_id", "owner", "time_desc", "states", "dev_name", "room_id", "room_name", "lab_name",
			}),
		}).
		Create(&records).Error
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
