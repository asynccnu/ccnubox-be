package errs

import (
	"net/http"

	b_errorx "github.com/asynccnu/ccnubox-be/bff/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

// --- Banner ---
var (
	GET_BANNER_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_BANNER_ERROR_CODE, "获取用banner失败!"))
	SAVE_BANNER_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SAVE_BANNER_ERROR_CODE, "保存banner失败!"))
	DEL_BANNER_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, DEL_BANNER_ERROR_CODE, "删除banner失败!"))
)

// --- Calendar ---
var (
	GET_CALENDAR_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_CALENDAR_ERROR_CODE, "获取日历失败!"))
	SAVE_CALENDAR_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SAVE_CALENDAR_ERROR_CODE, "保存日历失败!"))
	DEL_CALENDAR_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, DEL_CALENDAR_ERROR_CODE, "删除日历失败!"))
)

// --- InfoSum ---
var (
	GET_INFOSUM_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_INFOSUM_ERROR_CODE, "获取信息汇总失败!"))
	SAVE_INFOSUM_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SAVE_INFOSUM_ERROR_CODE, "保存信息汇总失败!"))
	DEL_INFOSUM_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, DEL_INFOSUM_ERROR_CODE, "删除信息汇总失败!"))
)

// --- Department ---
var (
	GET_DEPARTMENT_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_DEPARTMENT_ERROR_CODE, "获取部门信息失败!"))
	SAVE_DEPARTMENT_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SAVE_DEPARTMENT_ERROR_CODE, "保存部门信息失败!"))
	DEL_DEPARTMENT_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, DEL_DEPARTMENT_ERROR_CODE, "删除部门信息失败!"))
)

// --- Card ---
var (
	NOTE_USER_KEY_ERROR   = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, NOTE_USER_KEY_ERROR_CODE, "保存用户key失败!"))
	UPDATE_USER_KEY_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, UPDATE_USER_KEY_ERROR_CODE, "更新用户key失败!"))
	GET_RECORDS_ERROR     = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_RECORDS_ERROR_CODE, "获取校园卡信息失败!"))
)

// --- Class ---
var (
	GET_CLASS_LIST_ERROR          = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_CLASS_LIST_ERROR_CODE, "获取课程列表失败!"))
	ADD_CLASS_ERROR               = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, ADD_CLASS_ERROR_CODE, "添加课程失败!"))
	ADD_CLASS_CONFLICT_ERROR      = errorx.FormatErrorFunc(b_errorx.New(http.StatusConflict, ADD_CLASS_CONFLICT_ERROR_CODE, "课程时间冲突"))
	DELETE_CLASS_ERROR            = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, DELETE_CLASS_ERROR_CODE, "删除课程失败!"))
	UPDATE_CLASS_ERROR            = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, UPDATE_CLASS_ERROR_CODE, "更新课程失败!"))
	GET_RECYCLE_CLASS_ERROR       = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_RECYCLE_CLASS_ERROR_CODE, "获取回收站中的课程信息失败!"))
	RECOVER_CLASS_ERROR           = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, RECOVER_CLASS_ERROR_CODE, "恢复课程失败!"))
	SEARCH_CLASS_ERROR            = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SEARCH_CLASS_ERROR_CODE, "搜索课程失败!"))
	GET_TO_BE_STUDIED_CLASS_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_TO_BE_STUDIED_CLASS_ERROR_CODE, "获取待修读课程失败!"))
)

// --- ElecPrice ---
var (
	ELECPRICE_CHECK_ERROR             = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, ELECPRICE_CHECK_ERROR_CODE, "检查电费失败!"))
	ELECPRICE_SET_STANDARD_ERROR      = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, ELECPRICE_SET_STANDARD_ERROR_CODE, "设置电费提醒标准失败!"))
	ELECPRICE_GET_STANDARD_LIST_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, ELECPRICE_GET_STANDARD_LIST_ERROR_CODE, "获取电费提醒标准失败!"))
	ELECPRICE_CANCEL_STANDARD_ERROR   = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, ELECPRICE_CANCEL_STANDARD_ERROR_CODE, "取消电费提醒标准失败!"))
)

// --- Feed ---
var (
	GET_FEED_EVENTS_ERROR               = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_FEED_EVENTS_ERROR_CODE, "获取订阅事件失败!"))
	CLEAR_FEED_EVENT_ERROR              = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, CLEAR_FEED_EVENT_ERROR_CODE, "清空订阅事件失败!"))
	CHANGE_FEED_ALLOW_LIST_ERROR        = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, CHANGE_FEED_ALLOW_LIST_ERROR_CODE, "修改订阅白名单失败!"))
	GET_FEED_ALLOW_LIST_ERROR           = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_FEED_ALLOW_LIST_ERROR_CODE, "获取订阅白名单失败!"))
	READ_FEED_EVENT_ERROR               = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, READ_FEED_EVENT_ERROR_CODE, "标记订阅事件为已读失败!"))
	SAVE_FEED_TOKEN_ERROR               = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SAVE_FEED_TOKEN_ERROR_CODE, "保存订阅令牌失败!"))
	REMOVE_FEED_TOKEN_ERROR             = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, REMOVE_FEED_TOKEN_ERROR_CODE, "删除订阅令牌失败!"))
	PUBLIC_MUXI_OFFICIAL_MSG_ERROR      = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, PUBLIC_MUXI_OFFICIAL_MSG_ERROR_CODE, "发布木犀官方消息失败!"))
	STOP_MUXI_OFFICIAL_MSG_ERROR        = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, STOP_MUXI_OFFICIAL_MSG_ERROR_CODE, "停止木犀官方消息失败!"))
	GET_TO_BE_PUBLIC_OFFICIAL_MSG_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_TO_BE_PUBLIC_OFFICIAL_MSG_ERROR_CODE, "获取待发布的官方消息失败!"))
	GET_FAIL_MSG_ERROR                  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_FAIL_MSG_ERROR_CODE, "获取失败的消息失败!"))
	PUBLIC_FEED_EVENT_ERROR             = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, PUBLIC_FEED_EVENT_ERROR_CODE, "发布消息失败"))
)

// --- Question ---
var (
	GET_QUESTION_ERROR           = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_QUESTION_ERROR_CODE, "获取问题失败!"))
	CREATE_QUESTION_ERROR        = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, CREATE_QUESTION_ERROR_CODE, "创建问题失败!"))
	CHANGE_QUESTION_ERROR        = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, CHANGE_QUESTION_ERROR_CODE, "修改问题失败!"))
	DELETE_QUESTION_ERROR        = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, DELETE_QUESTION_ERROR_CODE, "删除问题失败!"))
	FIND_QUESTIONS_BY_NAME_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, FIND_QUESTIONS_BY_NAME_ERROR_CODE, "按名称查找问题失败!"))
	NOTE_QUESTION_ERROR          = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, NOTE_QUESTION_ERROR_CODE, "标记问题状态失败!"))
)

// --- Grade ---
var (
	GET_GRADE_BY_TERM_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_GRADE_BY_TERM_ERROR_CODE, "按学期获取成绩失败!"))
	GET_GRADE_SCORE_ERROR   = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_GRADE_SCORE_ERROR_CODE, "获取成绩分数失败!"))
	GET_RANK_BY_TERM_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_RANK_BY_TERM_ERROR_CODE, "获取学分绩排名失败!"))
	GET_GRADE_TYPE_ERROR    = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_GRADE_TYPE_ERROR_CODE, "获取课程性质列表失败！"))
)

// --- Static ---
var (
	GET_STATIC_BY_LABELS_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_STATIC_BY_LABELS_ERROR_CODE, "按标签匹配静态数据失败!"))
	SAVE_STATIC_ERROR          = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SAVE_STATIC_ERROR_CODE, "保存静态数据失败!"))
	SAVE_STATIC_BY_FILE_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SAVE_STATIC_BY_FILE_ERROR_CODE, "通过文件保存静态数据失败!"))
)

// --- Login & User ---
var (
	LOGIN_BY_CCNU_ERROR        = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, LOGIN_BY_CCNU_ERROR_CODE, "华中师范大学账号登录失败!"))
	LOGOUT_ERROR               = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, LOGOUT_ERROR_CODE, "登出失败!"))
	REFRESH_TOKEN_ERROR        = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, REFRESH_TOKEN_ERROR_CODE, "刷新 Token 失败!"))
	USER_SID_OR_PASSWORD_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusUnauthorized, USER_SID_OR_PASSWORD_ERROR_CODE, "账号或者密码错误!"))
	USER_SID_Or_PASSPORD_ERROR = USER_SID_OR_PASSWORD_ERROR // Deprecated: use USER_SID_OR_PASSWORD_ERROR.
)

// --- Common ---
var (
	BAD_ENTITY_ERROR          = errorx.FormatErrorFunc(b_errorx.New(http.StatusUnprocessableEntity, BAD_ENTITY_ERROR_CODE, "请求参数错误"))
	ROLE_ERROR                = errorx.FormatErrorFunc(b_errorx.New(http.StatusForbidden, ROLE_ERROR_CODE, "访问权限不足"))
	TYPE_CHANGE_ERROR         = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, TYPE_CHANGE_ERROR_CODE, "类型转换错误"))
	INVALID_PARAM_VALUE_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusBadRequest, INVALID_PARAM_VALUE_ERROR_CODE, "非法的参数值"))
)

// --- Website ---
var (
	GET_WEBSITES_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_WEBSITES_ERROR_CODE, "获取网站列表失败!"))
	SAVE_WEBSITE_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SAVE_WEBSITE_ERROR_CODE, "保存网站信息失败!"))
	DEL_WEBSITE_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, DEL_WEBSITE_ERROR_CODE, "删除网站失败!"))
)

// --- JWT / Auth ---
var (
	UNAUTHORIZED_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusUnauthorized, UNAUTHORIZED_ERROR_CODE, "Authorization错误"))
	UNAUTHORIED_ERROR  = UNAUTHORIZED_ERROR // Deprecated: use UNAUTHORIZED_ERROR.
	AUTH_EXPIRED_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusUnauthorized, AUTH_EXPIRED_ERROR_CODE, "Authorization过期"))
	AUTH_PASSED_ERROR  = AUTH_EXPIRED_ERROR // Deprecated: use AUTH_EXPIRED_ERROR.
	JWT_SYSTEM_ERROR   = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, JWT_SYSTEM_ERROR_CODE, "验证系统发生内部错误"))
)

// --- Library ---
var (
	GET_SEAT_ERROR           = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_SEAT_ERROR_CODE, "获取座位信息失败!"))
	RESERVE_SEAT_ERROR       = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, RESERVE_SEAT_ERROR_CODE, "预约座位失败!"))
	GET_SEAT_RECORD_ERROR    = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_SEAT_RECORD_ERROR_CODE, "获取未来预约失败!"))
	GET_HISTORY_ERROR        = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_HISTORY_ERROR_CODE, "获取历史记录失败!"))
	CANCEL_SEAT_ERROR        = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, CANCEL_SEAT_ERROR_CODE, "取消座位失败!"))
	GET_CREDIT_POINTS_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_CREDIT_POINTS_ERROR_CODE, "获取信誉分失败!"))
	GET_DISCUSSION_ERROR     = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_DISCUSSION_ERROR_CODE, "获取研讨间信息失败!"))
	SEARCH_USER_ERROR        = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SEARCH_USER_ERROR_CODE, "搜索用户失败!"))
	RESERVE_DISCUSSION_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, RESERVE_DISCUSSION_ERROR_CODE, "预约研讨间失败!"))
	CANCEL_DISCUSSION_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, CANCEL_DISCUSSION_ERROR_CODE, "取消研讨间失败!"))
	CREATE_COMMENT_ERROR     = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, CREATE_COMMENT_ERROR_CODE, "创建评论失败!"))
	GET_COMMENT_ERROR        = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_COMMENT_ERROR_CODE, "获取评论失败!"))
	DELETE_COMMENT_ERROR     = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, DELETE_COMMENT_ERROR_CODE, "删除评论失败!"))
)

// --- Swag ---
var (
	OPEN_SWAG_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, OPEN_SWAG_ERROR_CODE, "打开swagger失败"))
	MAKE_SWAG_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, MAKE_SWAG_ERROR_CODE, "生成swagger失败"))
)

// version
var (
	GET_UPDATE_VERSION_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_UPDATE_VERSION_ERROR_CODE, "获取热更新版本失败"))
	SAVE_UPDATE_VERSION_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SAVE_UPDATE_VERSION_ERROR_CODE, "保存热更新版本失败"))
)

var (
	GET_SEMESTER_ERROR  = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, GET_SEMESTER_ERROR_CODE, "获取学期列表失败"))
	SAVE_SEMESTER_ERROR = errorx.FormatErrorFunc(b_errorx.New(http.StatusInternalServerError, SAVE_SEMESTER_ERROR_CODE, "保存学期失败"))
)
