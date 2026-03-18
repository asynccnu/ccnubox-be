package cache

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewBaseCache,
	NewClassInfoCache,
	NewStudentCourseCache,
)
