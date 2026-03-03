package data

import (
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
)

// 将数据库中的数据结构与业务层相互转化

func metaDataDOToBO(meta ClassMetaData) biz.ClassMetaDataBO {
	return biz.ClassMetaDataBO{
		IsOfficial: !meta.IsManuallyAdded,
		Note:       meta.Note,
	}
}

func metaDataBOToDO(meta biz.ClassMetaDataBO) ClassMetaData {
	return ClassMetaData{
		IsManuallyAdded: !meta.IsOfficial,
		Note:            meta.Note,
	}
}

func classInfoDOToBO(do *ClassInfo, meta *ClassMetaData) *biz.ClassInfoBO {
	bo := &biz.ClassInfoBO{
		ID:           do.ID,
		CreatedAt:    do.CreatedAt,
		UpdatedAt:    do.UpdatedAt,
		JxbId:        do.JxbId,
		Day:          do.Day,
		Teacher:      do.Teacher,
		Where:        do.Where,
		ClassWhen:    do.ClassWhen,
		WeekDuration: do.WeekDuration,
		Classname:    do.Classname,
		Credit:       do.Credit,
		Weeks:        do.Weeks,
		Semester:     do.Semester,
		Year:         do.Year,
		Nature:       do.Nature,
	}

	if meta != nil {
		bo.MetaData = metaDataDOToBO(*meta)
	}
	return bo
}

func classInfoBOToDO(bo *biz.ClassInfoBO) *ClassInfo {
	cdo := &ClassInfo{
		ID:           bo.ID,
		CreatedAt:    bo.CreatedAt,
		UpdatedAt:    bo.UpdatedAt,
		JxbId:        bo.JxbId,
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
		Nature:       bo.Nature,
	}
	return cdo
}

func studentCourseBOToDO(bo *biz.StudentCourse) *StudentCourse {
	return &StudentCourse{
		StuID:           bo.StuID,
		ClaID:           bo.ClaID,
		Year:            bo.Year,
		Semester:        bo.Semester,
		IsManuallyAdded: bo.IsManuallyAdded,
		Note:            bo.Note,
		CreatedAt:       bo.CreatedAt,
		UpdatedAt:       bo.UpdatedAt,
	}
}

func ClassRefreshLogDOToBO(log *ClassRefreshLog) *biz.ClassRefreshLogBO {
	return &biz.ClassRefreshLogBO{
		ID:        log.ID,
		StuID:     log.StuID,
		Year:      log.Year,
		Semester:  log.Semester,
		Status:    log.Status,
		UpdatedAt: log.UpdatedAt,
	}
}
