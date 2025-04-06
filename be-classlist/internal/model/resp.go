package model

type GetClassInfosForUndergraduateResp struct {
	ClassInfos     []*ClassInfo
	StudentCourses []*StudentCourse
}
type GetClassInfoForGraduateStudentResp struct {
	ClassInfos     []*ClassInfo
	StudentCourses []*StudentCourse
}
type GetClassesFromLocalResp struct {
	ClassInfos []*ClassInfo
}
type GetSpecificClassInfoResp struct {
	ClassInfo *ClassInfo
}
type GetRecycledIdsResp struct {
	Ids []string
}
type GetAllSchoolClassInfosResp struct {
	ClassInfos []*ClassInfo
}
type GetAddedClassesResp struct {
	ClassInfos []*ClassInfo
}
