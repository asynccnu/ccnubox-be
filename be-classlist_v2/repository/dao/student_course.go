package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/gorm"
)

type StudentCourseDAO struct {
	Mysql *gorm.DB
}

func (s StudentCourseDAO) GetClassMetaData(ctx context.Context, stuID, year, semester string, claIds []string) map[string]model.ClassMetaData {
	logh := logger.From(ctx)

	// 初始化返回的 map
	res := make(map[string]model.ClassMetaData)
	if len(claIds) == 0 {
		return res
	}

	// 定义一个内部结构体用于接收扫描结果，因为需要 cla_id 作为 map 的 key
	type queryResult struct {
		ClaID           string `gorm:"column:cla_id"`
		IsManuallyAdded bool   `gorm:"column:is_manually_added"`
		Note            string `gorm:"column:note"`
	}

	var results []queryResult

	// 执行数据库查询
	db := s.Mysql(ctx).Table(model.StudentCourseTableName).WithContext(ctx)
	err := db.Select("cla_id", "is_manually_added", "note").
		Where("stu_id = ? AND year = ? AND semester = ? AND cla_id IN (?)", stuID, year, semester, claIds).
		Find(&results).Error
	if err != nil {
		logh.Errorf("GetClassMetaData 数据库查询失败: %v, stuID: %s", err, stuID)
		return res
	}

	// 将切片结果转换为 map
	for _, row := range results {
		res[row.ClaID] = model.ClassMetaData{
			Note:            row.Note,
			IsManuallyAdded: row.IsManuallyAdded,
		}
	}

	return res
}
