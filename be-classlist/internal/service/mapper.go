package service

import (
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	pb "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
)

// 将业务层的数据结构与服务层相互转化

func classInfoBOToPb(bo *biz.ClassInfoBO) *pb.ClassInfo {
	return &pb.ClassInfo{
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
