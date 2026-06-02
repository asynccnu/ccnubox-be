package biz

import "errors"

var (
	ErrClassNotFound         = errors.New("class not found")
	ErrClassAlreadyExists    = errors.New("class already exists")
	ErrClassScheduleConflict = errors.New("class schedule conflict")
	ErrStudentCourseNotFound = errors.New("student course relation not found")
	ErrInvalidParam          = errors.New("invalid param")
	ErrClassDeleteRejected   = errors.New("class delete rejected")
	ErrClassUpdateRejected   = errors.New("class update rejected")
	ErrGetStuIDsByJxbID      = errors.New("get student ids by jxb id failed")
)
