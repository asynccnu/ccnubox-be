package dao

//
//import (
//	"context"
//	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
//	"gorm.io/gorm"
//)
//
//type FeedEventIndexDAO interface {
//	GetFeedEventIndexListByStudentId(ctx context.Context, studentId string) (*[]model.FeedEventIndex, error)
//	SaveFeedEventIndex(ctx context.Context, eventIndex *model.FeedEventIndex) error
//	InsertFeedEventIndexList(ctx context.Context, eventIndexes []*model.FeedEventIndex) error
//	RemoveFeedEventIndex(ctx context.Context, studentId string, id int64, status string) error
//	GetFeedEventIndexById(ctx context.Context, id int64) (*model.FeedEventIndex, error)
//	InsertFeedEventIndexListByTx(ctx context.Context, tx *gorm.DB, eventIndexes []model.FeedEventIndex) error
//	// 用于事务
//	BeginTx(ctx context.Context) (*gorm.DB, error)
//}
//
//type feedEventIndexDAO struct {
//	gorm *gorm.DB
//}
//
//func NewFeedEventIndexDAO(db *gorm.DB) FeedEventIndexDAO {
//	return &feedEventIndexDAO{gorm: db}
//}
//
//func (dao *feedEventIndexDAO) GetFeedEventIndexListByStudentId(ctx context.Context, studentId string) (*[]model.FeedEventIndex, error) {
//	var s []model.FeedEventIndex
//	err := dao.gorm.WithContext(ctx).
//		Model(&model.FeedEventIndex{}).
//		Where("student_id = ?", studentId). // 使用参数化查询，避免 SQL 注入
//		Order("created_at DESC").           // 按照 created_at 字段降序排列，获取最新的记录
//		Limit(20).                          // 只获取最新的 20 条记录
//		Find(&s).
//		Error
//	return &s, err
//}
//
//func (dao *feedEventIndexDAO) InsertFeedEventIndexList(ctx context.Context, eventIndexes []*model.FeedEventIndex) error {
//	return dao.gorm.WithContext(ctx).Model(model.FeedEventIndex{}).Create(eventIndexes).Error
//}
//
//func (dao *feedEventIndexDAO) InsertFeedEventIndexListByTx(ctx context.Context, tx *gorm.DB, eventIndexes []model.FeedEventIndex) error {
//	return tx.WithContext(ctx).Create(eventIndexes).Error
//}
//
//func (dao *feedEventIndexDAO) SaveFeedEventIndex(ctx context.Context, eventIndex *model.FeedEventIndex) error {
//	return dao.gorm.WithContext(ctx).Save(eventIndex).Error
//}
//
//func (dao *feedEventIndexDAO) RemoveFeedEventIndex(ctx context.Context, studentId string, id int64, status string) error {
//	// studentId 必填
//	query := dao.gorm.WithContext(ctx).Where("student_id = ?", studentId)
//
//	// 如果 id 不是 0，则加入 id 过滤条件
//	if id != 0 {
//		query = query.Where("id = ?", id)
//	}
//
//	// 处理 status 参数
//	switch status {
//	case "read":
//		query = query.Where("`read` = ?", true) // 关键点：用 `read` 保护字段名,read是mysql的保留字....记得优化掉 TODO
//	case "unread":
//		query = query.Where("`read` = ?", false)
//	case "all":
//		// all 不需要添加 read 过滤条件
//	default:
//		query = query.Where("`read` = ?", true)
//	}
//
//	// 执行软删除
//	return query.Delete(&model.FeedEventIndex{}).Error
//}
//
//func (dao *feedEventIndexDAO) GetFeedEventIndexById(ctx context.Context, id int64) (*model.FeedEventIndex, error) {
//	var s model.FeedEventIndex
//	err := dao.gorm.WithContext(ctx).Where("id=?", id).Find(&s).Error
//	return &s, err
//}
//
//// BeginTx 开始事务,主要是用于批量插入feedEvent
//func (dao *feedEventIndexDAO) BeginTx(ctx context.Context) (*gorm.DB, error) {
//	tx := dao.gorm.WithContext(ctx).Begin()
//	if tx.Error != nil {
//		return nil, tx.Error
//	}
//	return tx, nil
//}
