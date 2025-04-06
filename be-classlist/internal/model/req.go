package model

import "time"

//const (
//	COMMONINFO = "commoninfo"
//)
//
//type CommonInfo struct {
//	StuId    string
//	Year     string
//	Semester string
//}

type GetClassInfosForUndergraduateReq struct {
	StuID    string
	Year     string
	Semester string
	Cookie   string
}
type GetClassInfoForGraduateStudentReq struct {
	StuID    string
	Year     string
	Semester string
	Cookie   string
}
type SaveClassReq struct {
	StuID      string
	Year       string
	Semester   string
	ClassInfos []*ClassInfo
	Scs        []*StudentCourse
}
type GetClassesFromLocalReq struct {
	StuID    string
	Year     string
	Semester string
}

type GetSpecificClassInfoReq struct {
	StuID    string
	Year     string
	Semester string
	ClassId  string
}
type AddClassReq struct {
	StuID     string
	Year      string
	Semester  string
	ClassInfo *ClassInfo
	Sc        *StudentCourse
}
type DeleteClassReq struct {
	StuID    string
	Year     string
	Semester string
	ClassId  []string
	//Sc []*StudentCourse
}
type RecoverClassFromRecycleBinReq struct {
	StuID    string
	Year     string
	Semester string
	ClassId  string
}
type UpdateClassReq struct {
	StuID        string
	Year         string
	Semester     string
	NewClassInfo *ClassInfo
	NewSc        *StudentCourse
	OldClassId   string
}
type CheckSCIdsExistReq struct {
	StuID    string
	Year     string
	Semester string
	ClassId  string
}
type CheckClassIdIsInRecycledBinReq struct {
	StuID    string
	Year     string
	Semester string
	ClassId  string
}
type GetRecycledIdsReq struct {
	StuID    string
	Year     string
	Semester string
}

type GetAllSchoolClassInfosReq struct {
	Year     string
	Semester string
	Cursor   time.Time
}

type GetAddedClassesReq struct {
	StudID   string
	Year     string
	Semester string
}

//func StoreCommonInfoInCtx(ctx context.Context, info CommonInfo) context.Context {
//	return context.WithValue(ctx, COMMONINFO, info)
//}
//func GetCommonInfoFromCtx(ctx context.Context) CommonInfo {
//	return ctx.Value(COMMONINFO).(CommonInfo)
//}
