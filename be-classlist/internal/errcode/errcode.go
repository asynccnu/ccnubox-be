package errcode

import (
	v1 "github.com/asynccnu/be-api/gen/proto/classlist/v1" //此处改成了be-api的部分
	"github.com/go-kratos/kratos/v2/errors"
)

var (
	ErrClassNotFound         = errors.New(450, v1.ErrorReason_DB_NOTFOUND.String(), "课程信息未找到")
	ErrClassFound            = errors.New(451, v1.ErrorReason_DB_FINDERR.String(), "数据库查找课程失败")
	ErrClassUpdate           = errors.New(452, v1.ErrorReason_DB_UPDATEERR.String(), "课程更新失败")
	ErrParam                 = errors.New(453, v1.ErrorReason_DB_UPDATEERR.String(), "入参错误")
	ErrCourseSave            = errors.New(454, v1.ErrorReason_DB_SAVEERROR.String(), "课程保存失败")
	ErrClassDelete           = errors.New(455, v1.ErrorReason_DB_DELETEERROR.String(), "课程删除失败")
	ErrCrawler               = errors.New(456, v1.ErrorReason_Crawler_Error.String(), "爬取课表失败")
	ErrCCNULogin             = errors.New(457, v1.ErrorReason_CCNULogin_Error.String(), "请求ccnu一站式登录服务错误")
	ErrSCIDNOTEXIST          = errors.New(458, v1.ErrorReason_SCIDNOTEXIST_Erroe.String(), "学号与课程ID的对应关系未找到")
	ErrRecycleBinDoNotHaveIt = errors.New(459, v1.ErrorReason_RECYCLEBINDONOTHAVETHECLASS.String(), "回收站中不存在该课程")
	ErrRecover               = errors.New(460, v1.ErrorReason_RECOVERFAILED.String(), "恢复课程失败")
	ErrGetStuIdByJxbId       = errors.New(461, v1.ErrorReason_GETSTUIDBYJXBID.String(), "通过jxb_id获取stu_ids获取失败")
	ErrClassIsExist          = errors.New(462, v1.ErrorReason_CLASSISEXIST.String(), "已有该课程")
)
