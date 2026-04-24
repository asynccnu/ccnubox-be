package grpc

import (
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/pkg/tool"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
)

// classInfoBOToPb 将业务 BO 转换为 proto 消息
func classInfoBOToPb(bo *model.ClassInfoBO) *classlistv1.ClassInfo {
	return &classlistv1.ClassInfo{
		Day:          bo.Day,
		Teacher:      bo.Teacher,
		Where:        bo.Where,
		ClassWhen:    bo.ClassWhen,
		WeekDuration: bo.WeekDuration,
		Classname:    bo.Classname,
		Credit:       bo.Credit,
		Weeks:        bo.Weeks,
		Semester:     bo.Semester,
		Year:         bo.Year,
		Id:           bo.ID,
		Note:         bo.MetaData.Note,
		IsOfficial:   bo.MetaData.IsOfficial,
		Nature:       bo.Nature,
	}
}

// toShanghaiTimeStamp 将 time.Time 转换为上海时区下的 Unix 秒
func toShanghaiTimeStamp(t time.Time) int64 {
	return tool.ToShanghaiTime(t).Unix()
}
