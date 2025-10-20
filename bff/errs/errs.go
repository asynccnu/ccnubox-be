package errs

import (
	"net/http"

	"github.com/asynccnu/ccnubox-be/bff/pkg/errorx"
)

// TODO 细化错误码,根据错误类型区分不同的错误码
// 现在这个错误码基本上是随便写的,因为微服务的错误码非常之多暂时没时间细化,所有的都先默认系统错误

// 400
const (
	UNAUTHORIED_ERROR_CODE = iota + 40001
	BAD_ENTITY_ERROR_CODE
	ROLE_ERROR_CODE
	INVALID_PARAM_VALUE_ERROR_CODE
	USER_SID_OR_PASSPORD_ERROR_CODE
)

// 500
const (
	INTERNAL_SERVER_ERROR_CODE = iota + 50001 // INTERNAL_SERVER_ERROR_CODE 一个非常含糊的错误码。代表系统内部错误
	ERROR_TYPE_ERROR_CODE
	TYPE_CHANGE_ERROR_CODE
	LOGIN_BY_CCNU_ERROR_CODE
)

// Banner
var (
	GET_BANNER_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取用banner失败!", "Banner", err)
	}

	Save_BANNER_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "保存banner失败!", "Banner", err)
	}

	Del_BANNER_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "删除banner失败!", "Banner", err)
	}
)

// Calendar
var (
	GET_CALENDAR_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取日历失败!", "Calendar", err)
	}

	Save_CALENDAR_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "保存日历失败!", "Calendar", err)
	}

	Del_CALENDAR_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "删除日历失败!", "Calendar", err)
	}
)

// InfoSum
var (
	GET_INFOSUM_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取信息汇总失败!", "InfoSum", err)
	}

	Save_INFOSUM_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "保存信息汇总失败!", "InfoSum", err)
	}

	Del_INFOSUM_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "删除信息汇总失败!", "InfoSum", err)
	}
)

// department
var (
	GET_DEPARTMENT_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取部门信息失败!", "Department", err)
	}

	SAVE_DEPARTMENT_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "保存部门信息失败!", "Department", err)
	}

	DEL_DEPARTMENT_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "删除部门信息失败!", "Department", err)
	}
)

// Card
var (
	NOTE_USER_KEY_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "保存用户key失败!", "InfoSum", err)
	}

	UPDATE_USER_KEY_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "更新用户key失败!", "InfoSum", err)
	}

	GET_RECORDS_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取校园卡信息失败!", "InfoSum", err)
	}
)

// Class
var (
	GET_CLASS_LIST_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取课程列表失败!", "Class", err)
	}

	ADD_CLASS_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "添加课程失败!", "Class", err)
	}

	DELETE_CLASS_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "删除课程失败!", "class", err)
	}

	UPDATE_CLASS_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "更新课程失败!", "Class", err)
	}

	GET_RECYCLE_CLASS_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取回收站中的课程信息失败!", "Class", err)
	}

	RECOVER_CLASS_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "恢复课程失败!", "Class", err)
	}

	SEARCH_CLASS_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "搜索课程失败!", "Class", err)
	}
)

var (
	ELECPRICE_CHECK_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "检查电费失败!", "elecprice", err)
	}

	ELECPRICE_SET_STANDARD_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "设置电费提醒标准失败!", "elecprice", err)
	}

	ELECPRICE_GET_STANDARD_LIST_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取电费提醒标准失败!", "elecprice", err)
	}

	ELECPRICE_CANCEL_STANDARD_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "取消电费提醒标准失败!", "elecprice", err)
	}
)

// Feed
var (
	GET_FEED_EVENTS_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取订阅事件失败!", "feed", err)
	}

	CLEAR_FEED_EVENT_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "清空订阅事件失败!", "feed", err)
	}

	CHANGE_FEED_ALLOW_LIST_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "修改订阅白名单失败!", "feed", err)
	}

	GET_FEED_ALLOW_LIST_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取订阅白名单失败!", "feed", err)
	}

	READ_FEED_EVENT_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "标记订阅事件为已读失败!", "feed", err)
	}

	SAVE_FEED_TOKEN_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "保存订阅令牌失败!", "feed", err)
	}

	REMOVE_FEED_TOKEN_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "删除订阅令牌失败!", "feed", err)
	}

	PUBLIC_MUXI_OFFICIAL_MSG_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "发布木犀官方消息失败!", "feed", err)
	}

	STOP_MUXI_OFFICIAL_MSG_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "停止木犀官方消息失败!", "feed", err)
	}

	GET_TO_BE_PUBLIC_OFFICIAL_MSG_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取待发布的官方消息失败!", "feed", err)
	}

	GET_FAIL_MSG_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取失败的消息失败!", "feed", err)
	}
)

// question
var (
	GET_QUESTION_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取问题失败!", "question", err)
	}

	CREATE_QUESTION_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "创建问题失败!", "question", err)
	}

	CHANGE_QUESTION_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "修改问题失败!", "question", err)
	}

	DELETE_QUESTION_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "删除问题失败!", "question", err)
	}

	FIND_QUESTIONS_BY_NAME_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "按名称查找问题失败!", "question", err)
	}

	NOTE_QUESTION_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "标记问题状态失败!", "question", err)
	}
)

// grade
var (
	GET_GRADE_BY_TERM_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "按学期获取成绩失败!", "grade", err)
	}

	GET_GRADE_SCORE_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取成绩分数失败!", "grade", err)
	}
)

// static
var (
	GET_STATIC_BY_LABELS_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "按标签匹配静态数据失败!", "static", err)
	}

	SAVE_STATIC_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "保存静态数据失败!", "static", err)
	}

	SAVE_STATIC_BY_FILE_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "通过文件保存静态数据失败!", "static", err)
	}
)

// login
var (
	LOGIN_BY_CCNU_ERROR = func(err error) error {
		return errorx.New(http.StatusUnauthorized, LOGIN_BY_CCNU_ERROR_CODE, "华中师范大学账号登录失败!", "ccnu", err)
	}

	LOGOUT_ERROR = func(err error) error {
		return errorx.New(http.StatusUnauthorized, INTERNAL_SERVER_ERROR_CODE, "登出失败!", "user", err)
	}

	REFRESH_TOKEN_ERROR = func(err error) error {
		return errorx.New(http.StatusUnauthorized, INTERNAL_SERVER_ERROR_CODE, "刷新 Token 失败!", "user", err)
	}

	USER_SID_Or_PASSPORD_ERROR = func(err error) error {
		return errorx.New(http.StatusUnauthorized, USER_SID_OR_PASSPORD_ERROR_CODE, "账号或者密码错误!", "user", err)
	}
)

// Common
var (
	BAD_ENTITY_ERROR = func(err error) error {
		return errorx.New(http.StatusUnprocessableEntity, BAD_ENTITY_ERROR_CODE, "请求参数错误", "Common", err)
	}

	ROLE_ERROR = func(err error) error {
		return errorx.New(http.StatusForbidden, ROLE_ERROR_CODE, "访问权限不足", "Common", err)
	}

	TYPE_CHANGE_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, TYPE_CHANGE_ERROR_CODE, "类型转换错误", "Common", err)
	}

	INVALID_PARAM_VALUE_ERROR = func(err error) error {
		return errorx.New(http.StatusBadRequest, INVALID_PARAM_VALUE_ERROR_CODE, "非法的参数值", "Common", err)
	}
)

// website
var (
	GET_WEBSITES_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "获取网站列表失败!", "GetWebsites", err)
	}

	SAVE_WEBSITE_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "保存网站信息失败!", "SaveWebsite", err)
	}

	DEL_WEBSITE_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "删除网站失败!", "DelWebsite", err)
	}
)

// JWT
var (
	UNAUTHORIED_ERROR = func(err error) error {
		return errorx.New(http.StatusUnauthorized, UNAUTHORIED_ERROR_CODE, "Authorization错误", "authorization", err)
	}

	AUTH_PASSED_ERROR = func(err error) error {
		return errorx.New(http.StatusUnauthorized, UNAUTHORIED_ERROR_CODE, "Authorization过期", "authorization", err)
	}

	JWT_SYSTEM_ERROR = func(err error) error {
		return errorx.New(http.StatusInternalServerError, INTERNAL_SERVER_ERROR_CODE, "验证系统发生内部错误", "authorization", err)
	}
)
