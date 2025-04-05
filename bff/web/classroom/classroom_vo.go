package classroom

import cs "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classService/v1"

type GetFreeClassRoomReq struct {
	Year        string  `form:"year"`        // 学年
	Semester    string  `form:"semester"`    // 学期
	Week        int32   `form:"week"`        // 哪一周
	Day         int32   `form:"day"`         // 哪一天
	Sections    []int32 `form:"sections"`    // 哪几节课（多个字段：sections=1&sections=2）
	WherePrefix string  `form:"wherePrefix"` // 地点前缀
	StuID       string  `form:"stuID"`       // 学号
}

type ClassroomAvailableStat struct {
	Classroom     string `json:"classroom"`     // 教室名
	AvailableStat []bool `json:"availableStat"` // 空闲情况（与sections一一对应）
}

type GetFreeClassRoomResp struct {
	Stat []ClassroomAvailableStat `json:"stat"` // 各教室的空闲情况
}

func convertToGetFreeClassRoomResp(protoResp *cs.QueryFreeClassroomResp) *GetFreeClassRoomResp {
	if protoResp == nil {
		return &GetFreeClassRoomResp{}
	}

	var result GetFreeClassRoomResp
	for _, stat := range protoResp.Stat {
		if stat == nil {
			continue
		}
		result.Stat = append(result.Stat, ClassroomAvailableStat{
			Classroom:     stat.Classroom,
			AvailableStat: stat.AvailableStat,
		})
	}
	return &result
}
