package dao

import (
	"context"
	"errors"

	"github.com/asynccnu/ccnubox-be/be-library/internal/data/DO"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CreditPointDAO struct {
	db *gorm.DB
}

func NewCreditPointDAO(db *gorm.DB) *CreditPointDAO {
	return &CreditPointDAO{db: db}
}

func (d *CreditPointDAO) UpsertSummary(ctx context.Context, summary *DO.CreditSummary) error {
	if summary == nil {
		return nil
	}

	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "stu_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"system", "remain", "total"}),
		}).
		Create(summary).Error
}

func (d *CreditPointDAO) UpsertRecords(ctx context.Context, records []DO.CreditRecord) error {
	if len(records) == 0 {
		return nil
	}

	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "stu_id"},
				{Name: "title"},
				{Name: "subtitle"},
				{Name: "location"},
			},
			DoNothing: true,
		}).
		Create(&records).Error
}

func (d *CreditPointDAO) GetSummary(ctx context.Context, stuID string) (*DO.CreditSummary, error) {
	var summary DO.CreditSummary
	err := d.db.WithContext(ctx).Where("stu_id = ?", stuID).First(&summary).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &summary, err
}

func (d *CreditPointDAO) ListRecords(ctx context.Context, stuID string) ([]DO.CreditRecord, error) {
	var records []DO.CreditRecord
	err := d.db.WithContext(ctx).Where("stu_id = ?", stuID).Find(&records).Error
	return records, err
}

