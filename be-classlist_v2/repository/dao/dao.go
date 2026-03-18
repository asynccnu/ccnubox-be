package dao

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewBaseDAO,
	NewClassInfoDAO,
	NewJxbDAO,
	NewRefreshLogDAO,
	NewStudentCourseDAO,
)
