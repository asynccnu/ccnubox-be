package service

import (
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

var (
	ClassNotFoundError         = errorx.FormatErrorFunc(classlistv1.ErrorDbNotfound("课程信息未找到"))
	ClassFindError             = errorx.FormatErrorFunc(classlistv1.ErrorDbFinderr("数据库查找课程失败"))
	ClassUpdateError           = errorx.FormatErrorFunc(classlistv1.ErrorDbUpdateerr("课程更新失败"))
	ParamError                 = errorx.FormatErrorFunc(classlistv1.ErrorParamErr("入参错误"))
	ClassDeleteError           = errorx.FormatErrorFunc(classlistv1.ErrorDbDeleteerror("课程删除失败"))
	StudentCourseNotFoundError = errorx.FormatErrorFunc(classlistv1.ErrorScidnotexistErroe("学号与课程ID的对应关系未找到"))
	GetStuIDByJxbIDError       = errorx.FormatErrorFunc(classlistv1.ErrorGetstuidbyjxbid("通过jxb_id获取stu_ids获取失败"))
	ClassAlreadyExistsError    = errorx.FormatErrorFunc(classlistv1.ErrorClassisexist("已有该课程"))
	ConfigError                = errorx.FormatErrorFunc(classlistv1.ErrorConfigError("配置错误"))
	ClassScheduleConflictError = errorx.FormatErrorFunc(classlistv1.ErrorErrClassScheduleConflict("添加课程时间冲突"))
)
