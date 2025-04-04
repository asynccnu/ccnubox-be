// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.2
// 	protoc        v5.26.1
// source: grade/v1/grade.proto

package gradev1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// 请求体
type GetGradeByTermReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StudentId string `protobuf:"bytes,1,opt,name=studentId,proto3" json:"studentId,omitempty"` //学号
	Xnm       int64  `protobuf:"varint,2,opt,name=xnm,proto3" json:"xnm,omitempty"`            //学年名:例如2024表示2024-2025学年
	Xqm       int64  `protobuf:"varint,3,opt,name=xqm,proto3" json:"xqm,omitempty"`            //学期名
}

func (x *GetGradeByTermReq) Reset() {
	*x = GetGradeByTermReq{}
	mi := &file_grade_v1_grade_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetGradeByTermReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGradeByTermReq) ProtoMessage() {}

func (x *GetGradeByTermReq) ProtoReflect() protoreflect.Message {
	mi := &file_grade_v1_grade_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGradeByTermReq.ProtoReflect.Descriptor instead.
func (*GetGradeByTermReq) Descriptor() ([]byte, []int) {
	return file_grade_v1_grade_proto_rawDescGZIP(), []int{0}
}

func (x *GetGradeByTermReq) GetStudentId() string {
	if x != nil {
		return x.StudentId
	}
	return ""
}

func (x *GetGradeByTermReq) GetXnm() int64 {
	if x != nil {
		return x.Xnm
	}
	return 0
}

func (x *GetGradeByTermReq) GetXqm() int64 {
	if x != nil {
		return x.Xqm
	}
	return 0
}

// 响应体
type GetGradeByTermResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Grades []*Grade `protobuf:"bytes,1,rep,name=grades,proto3" json:"grades,omitempty"` // 课程详细信息
}

func (x *GetGradeByTermResp) Reset() {
	*x = GetGradeByTermResp{}
	mi := &file_grade_v1_grade_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetGradeByTermResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGradeByTermResp) ProtoMessage() {}

func (x *GetGradeByTermResp) ProtoReflect() protoreflect.Message {
	mi := &file_grade_v1_grade_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGradeByTermResp.ProtoReflect.Descriptor instead.
func (*GetGradeByTermResp) Descriptor() ([]byte, []int) {
	return file_grade_v1_grade_proto_rawDescGZIP(), []int{1}
}

func (x *GetGradeByTermResp) GetGrades() []*Grade {
	if x != nil {
		return x.Grades
	}
	return nil
}

// 成绩结构体
type Grade struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Kcmc                string  `protobuf:"bytes,1,opt,name=Kcmc,proto3" json:"Kcmc,omitempty"`                               //课程名
	Xf                  float32 `protobuf:"fixed32,2,opt,name=Xf,proto3" json:"Xf,omitempty"`                                 //学分
	Cj                  float32 `protobuf:"fixed32,3,opt,name=Cj,proto3" json:"Cj,omitempty"`                                 //总成绩
	Kcxzmc              string  `protobuf:"bytes,4,opt,name=kcxzmc,proto3" json:"kcxzmc,omitempty"`                           //课程性质名称 比如专业主干课程/通识必修课
	Kclbmc              string  `protobuf:"bytes,5,opt,name=Kclbmc,proto3" json:"Kclbmc,omitempty"`                           //课程类别名称，比如专业课/公共课
	Kcbj                string  `protobuf:"bytes,6,opt,name=kcbj,proto3" json:"kcbj,omitempty"`                               //课程标记，比如主修/辅修
	Jd                  float32 `protobuf:"fixed32,7,opt,name=jd,proto3" json:"jd,omitempty"`                                 // 绩点
	RegularGradePercent string  `protobuf:"bytes,8,opt,name=regularGradePercent,proto3" json:"regularGradePercent,omitempty"` //平时成绩占比
	RegularGrade        float32 `protobuf:"fixed32,9,opt,name=regularGrade,proto3" json:"regularGrade,omitempty"`             //平时成绩
	FinalGradePercent   string  `protobuf:"bytes,10,opt,name=finalGradePercent,proto3" json:"finalGradePercent,omitempty"`    //期末成绩占比
	FinalGrade          float32 `protobuf:"fixed32,11,opt,name=finalGrade,proto3" json:"finalGrade,omitempty"`                //期末成绩
}

func (x *Grade) Reset() {
	*x = Grade{}
	mi := &file_grade_v1_grade_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Grade) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Grade) ProtoMessage() {}

func (x *Grade) ProtoReflect() protoreflect.Message {
	mi := &file_grade_v1_grade_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Grade.ProtoReflect.Descriptor instead.
func (*Grade) Descriptor() ([]byte, []int) {
	return file_grade_v1_grade_proto_rawDescGZIP(), []int{2}
}

func (x *Grade) GetKcmc() string {
	if x != nil {
		return x.Kcmc
	}
	return ""
}

func (x *Grade) GetXf() float32 {
	if x != nil {
		return x.Xf
	}
	return 0
}

func (x *Grade) GetCj() float32 {
	if x != nil {
		return x.Cj
	}
	return 0
}

func (x *Grade) GetKcxzmc() string {
	if x != nil {
		return x.Kcxzmc
	}
	return ""
}

func (x *Grade) GetKclbmc() string {
	if x != nil {
		return x.Kclbmc
	}
	return ""
}

func (x *Grade) GetKcbj() string {
	if x != nil {
		return x.Kcbj
	}
	return ""
}

func (x *Grade) GetJd() float32 {
	if x != nil {
		return x.Jd
	}
	return 0
}

func (x *Grade) GetRegularGradePercent() string {
	if x != nil {
		return x.RegularGradePercent
	}
	return ""
}

func (x *Grade) GetRegularGrade() float32 {
	if x != nil {
		return x.RegularGrade
	}
	return 0
}

func (x *Grade) GetFinalGradePercent() string {
	if x != nil {
		return x.FinalGradePercent
	}
	return ""
}

func (x *Grade) GetFinalGrade() float32 {
	if x != nil {
		return x.FinalGrade
	}
	return 0
}

type GetGradeScoreReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StudentId string `protobuf:"bytes,1,opt,name=studentId,proto3" json:"studentId,omitempty"`
}

func (x *GetGradeScoreReq) Reset() {
	*x = GetGradeScoreReq{}
	mi := &file_grade_v1_grade_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetGradeScoreReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGradeScoreReq) ProtoMessage() {}

func (x *GetGradeScoreReq) ProtoReflect() protoreflect.Message {
	mi := &file_grade_v1_grade_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGradeScoreReq.ProtoReflect.Descriptor instead.
func (*GetGradeScoreReq) Descriptor() ([]byte, []int) {
	return file_grade_v1_grade_proto_rawDescGZIP(), []int{3}
}

func (x *GetGradeScoreReq) GetStudentId() string {
	if x != nil {
		return x.StudentId
	}
	return ""
}

type GetGradeScoreResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TypeOfGradeScore []*TypeOfGradeScore `protobuf:"bytes,1,rep,name=typeOfGradeScore,proto3" json:"typeOfGradeScore,omitempty"`
}

func (x *GetGradeScoreResp) Reset() {
	*x = GetGradeScoreResp{}
	mi := &file_grade_v1_grade_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetGradeScoreResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGradeScoreResp) ProtoMessage() {}

func (x *GetGradeScoreResp) ProtoReflect() protoreflect.Message {
	mi := &file_grade_v1_grade_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGradeScoreResp.ProtoReflect.Descriptor instead.
func (*GetGradeScoreResp) Descriptor() ([]byte, []int) {
	return file_grade_v1_grade_proto_rawDescGZIP(), []int{4}
}

func (x *GetGradeScoreResp) GetTypeOfGradeScore() []*TypeOfGradeScore {
	if x != nil {
		return x.TypeOfGradeScore
	}
	return nil
}

type TypeOfGradeScore struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Kcxzmc         string        `protobuf:"bytes,1,opt,name=kcxzmc,proto3" json:"kcxzmc,omitempty"`
	GradeScoreList []*GradeScore `protobuf:"bytes,2,rep,name=gradeScoreList,proto3" json:"gradeScoreList,omitempty"`
}

func (x *TypeOfGradeScore) Reset() {
	*x = TypeOfGradeScore{}
	mi := &file_grade_v1_grade_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TypeOfGradeScore) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TypeOfGradeScore) ProtoMessage() {}

func (x *TypeOfGradeScore) ProtoReflect() protoreflect.Message {
	mi := &file_grade_v1_grade_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TypeOfGradeScore.ProtoReflect.Descriptor instead.
func (*TypeOfGradeScore) Descriptor() ([]byte, []int) {
	return file_grade_v1_grade_proto_rawDescGZIP(), []int{5}
}

func (x *TypeOfGradeScore) GetKcxzmc() string {
	if x != nil {
		return x.Kcxzmc
	}
	return ""
}

func (x *TypeOfGradeScore) GetGradeScoreList() []*GradeScore {
	if x != nil {
		return x.GradeScoreList
	}
	return nil
}

type GradeScore struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Kcmc string  `protobuf:"bytes,1,opt,name=Kcmc,proto3" json:"Kcmc,omitempty"` //课程名
	Xf   float32 `protobuf:"fixed32,2,opt,name=Xf,proto3" json:"Xf,omitempty"`   //学分
}

func (x *GradeScore) Reset() {
	*x = GradeScore{}
	mi := &file_grade_v1_grade_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GradeScore) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GradeScore) ProtoMessage() {}

func (x *GradeScore) ProtoReflect() protoreflect.Message {
	mi := &file_grade_v1_grade_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GradeScore.ProtoReflect.Descriptor instead.
func (*GradeScore) Descriptor() ([]byte, []int) {
	return file_grade_v1_grade_proto_rawDescGZIP(), []int{6}
}

func (x *GradeScore) GetKcmc() string {
	if x != nil {
		return x.Kcmc
	}
	return ""
}

func (x *GradeScore) GetXf() float32 {
	if x != nil {
		return x.Xf
	}
	return 0
}

var File_grade_v1_grade_proto protoreflect.FileDescriptor

var file_grade_v1_grade_proto_rawDesc = []byte{
	0x0a, 0x14, 0x67, 0x72, 0x61, 0x64, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x67, 0x72, 0x61, 0x64, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x67, 0x72, 0x61, 0x64, 0x65, 0x2e, 0x76, 0x31,
	0x22, 0x55, 0x0a, 0x11, 0x47, 0x65, 0x74, 0x47, 0x72, 0x61, 0x64, 0x65, 0x42, 0x79, 0x54, 0x65,
	0x72, 0x6d, 0x52, 0x65, 0x71, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74,
	0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x74, 0x75, 0x64, 0x65, 0x6e,
	0x74, 0x49, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x78, 0x6e, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x03, 0x78, 0x6e, 0x6d, 0x12, 0x10, 0x0a, 0x03, 0x78, 0x71, 0x6d, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x03, 0x78, 0x71, 0x6d, 0x22, 0x3d, 0x0a, 0x12, 0x47, 0x65, 0x74, 0x47, 0x72,
	0x61, 0x64, 0x65, 0x42, 0x79, 0x54, 0x65, 0x72, 0x6d, 0x52, 0x65, 0x73, 0x70, 0x12, 0x27, 0x0a,
	0x06, 0x67, 0x72, 0x61, 0x64, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0f, 0x2e,
	0x67, 0x72, 0x61, 0x64, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x72, 0x61, 0x64, 0x65, 0x52, 0x06,
	0x67, 0x72, 0x61, 0x64, 0x65, 0x73, 0x22, 0xb3, 0x02, 0x0a, 0x05, 0x47, 0x72, 0x61, 0x64, 0x65,
	0x12, 0x12, 0x0a, 0x04, 0x4b, 0x63, 0x6d, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x4b, 0x63, 0x6d, 0x63, 0x12, 0x0e, 0x0a, 0x02, 0x58, 0x66, 0x18, 0x02, 0x20, 0x01, 0x28, 0x02,
	0x52, 0x02, 0x58, 0x66, 0x12, 0x0e, 0x0a, 0x02, 0x43, 0x6a, 0x18, 0x03, 0x20, 0x01, 0x28, 0x02,
	0x52, 0x02, 0x43, 0x6a, 0x12, 0x16, 0x0a, 0x06, 0x6b, 0x63, 0x78, 0x7a, 0x6d, 0x63, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6b, 0x63, 0x78, 0x7a, 0x6d, 0x63, 0x12, 0x16, 0x0a, 0x06,
	0x4b, 0x63, 0x6c, 0x62, 0x6d, 0x63, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x4b, 0x63,
	0x6c, 0x62, 0x6d, 0x63, 0x12, 0x12, 0x0a, 0x04, 0x6b, 0x63, 0x62, 0x6a, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x6b, 0x63, 0x62, 0x6a, 0x12, 0x0e, 0x0a, 0x02, 0x6a, 0x64, 0x18, 0x07,
	0x20, 0x01, 0x28, 0x02, 0x52, 0x02, 0x6a, 0x64, 0x12, 0x30, 0x0a, 0x13, 0x72, 0x65, 0x67, 0x75,
	0x6c, 0x61, 0x72, 0x47, 0x72, 0x61, 0x64, 0x65, 0x50, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74, 0x18,
	0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x13, 0x72, 0x65, 0x67, 0x75, 0x6c, 0x61, 0x72, 0x47, 0x72,
	0x61, 0x64, 0x65, 0x50, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74, 0x12, 0x22, 0x0a, 0x0c, 0x72, 0x65,
	0x67, 0x75, 0x6c, 0x61, 0x72, 0x47, 0x72, 0x61, 0x64, 0x65, 0x18, 0x09, 0x20, 0x01, 0x28, 0x02,
	0x52, 0x0c, 0x72, 0x65, 0x67, 0x75, 0x6c, 0x61, 0x72, 0x47, 0x72, 0x61, 0x64, 0x65, 0x12, 0x2c,
	0x0a, 0x11, 0x66, 0x69, 0x6e, 0x61, 0x6c, 0x47, 0x72, 0x61, 0x64, 0x65, 0x50, 0x65, 0x72, 0x63,
	0x65, 0x6e, 0x74, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x09, 0x52, 0x11, 0x66, 0x69, 0x6e, 0x61, 0x6c,
	0x47, 0x72, 0x61, 0x64, 0x65, 0x50, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74, 0x12, 0x1e, 0x0a, 0x0a,
	0x66, 0x69, 0x6e, 0x61, 0x6c, 0x47, 0x72, 0x61, 0x64, 0x65, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x02,
	0x52, 0x0a, 0x66, 0x69, 0x6e, 0x61, 0x6c, 0x47, 0x72, 0x61, 0x64, 0x65, 0x22, 0x30, 0x0a, 0x10,
	0x47, 0x65, 0x74, 0x47, 0x72, 0x61, 0x64, 0x65, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x52, 0x65, 0x71,
	0x12, 0x1c, 0x0a, 0x09, 0x73, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x22, 0x5b,
	0x0a, 0x11, 0x47, 0x65, 0x74, 0x47, 0x72, 0x61, 0x64, 0x65, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x52,
	0x65, 0x73, 0x70, 0x12, 0x46, 0x0a, 0x10, 0x74, 0x79, 0x70, 0x65, 0x4f, 0x66, 0x47, 0x72, 0x61,
	0x64, 0x65, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1a, 0x2e,
	0x67, 0x72, 0x61, 0x64, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x4f, 0x66, 0x47,
	0x72, 0x61, 0x64, 0x65, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x52, 0x10, 0x74, 0x79, 0x70, 0x65, 0x4f,
	0x66, 0x47, 0x72, 0x61, 0x64, 0x65, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x22, 0x68, 0x0a, 0x10, 0x54,
	0x79, 0x70, 0x65, 0x4f, 0x66, 0x47, 0x72, 0x61, 0x64, 0x65, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x6b, 0x63, 0x78, 0x7a, 0x6d, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x6b, 0x63, 0x78, 0x7a, 0x6d, 0x63, 0x12, 0x3c, 0x0a, 0x0e, 0x67, 0x72, 0x61, 0x64, 0x65,
	0x53, 0x63, 0x6f, 0x72, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x14, 0x2e, 0x67, 0x72, 0x61, 0x64, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x72, 0x61, 0x64, 0x65,
	0x53, 0x63, 0x6f, 0x72, 0x65, 0x52, 0x0e, 0x67, 0x72, 0x61, 0x64, 0x65, 0x53, 0x63, 0x6f, 0x72,
	0x65, 0x4c, 0x69, 0x73, 0x74, 0x22, 0x30, 0x0a, 0x0a, 0x47, 0x72, 0x61, 0x64, 0x65, 0x53, 0x63,
	0x6f, 0x72, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x4b, 0x63, 0x6d, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x4b, 0x63, 0x6d, 0x63, 0x12, 0x0e, 0x0a, 0x02, 0x58, 0x66, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x02, 0x52, 0x02, 0x58, 0x66, 0x32, 0xa5, 0x01, 0x0a, 0x0c, 0x47, 0x72, 0x61, 0x64,
	0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x4b, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x47,
	0x72, 0x61, 0x64, 0x65, 0x42, 0x79, 0x54, 0x65, 0x72, 0x6d, 0x12, 0x1b, 0x2e, 0x67, 0x72, 0x61,
	0x64, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x47, 0x72, 0x61, 0x64, 0x65, 0x42, 0x79,
	0x54, 0x65, 0x72, 0x6d, 0x52, 0x65, 0x71, 0x1a, 0x1c, 0x2e, 0x67, 0x72, 0x61, 0x64, 0x65, 0x2e,
	0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x47, 0x72, 0x61, 0x64, 0x65, 0x42, 0x79, 0x54, 0x65, 0x72,
	0x6d, 0x52, 0x65, 0x73, 0x70, 0x12, 0x48, 0x0a, 0x0d, 0x47, 0x65, 0x74, 0x47, 0x72, 0x61, 0x64,
	0x65, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x12, 0x1a, 0x2e, 0x67, 0x72, 0x61, 0x64, 0x65, 0x2e, 0x76,
	0x31, 0x2e, 0x47, 0x65, 0x74, 0x47, 0x72, 0x61, 0x64, 0x65, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x52,
	0x65, 0x71, 0x1a, 0x1b, 0x2e, 0x67, 0x72, 0x61, 0x64, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65,
	0x74, 0x47, 0x72, 0x61, 0x64, 0x65, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x52, 0x65, 0x73, 0x70, 0x42,
	0x42, 0x5a, 0x40, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x73,
	0x79, 0x6e, 0x63, 0x63, 0x6e, 0x75, 0x2f, 0x63, 0x63, 0x6e, 0x75, 0x62, 0x6f, 0x78, 0x2d, 0x62,
	0x65, 0x2f, 0x62, 0x65, 0x2d, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x67, 0x72, 0x61, 0x64, 0x65, 0x2f, 0x76, 0x31, 0x3b, 0x67, 0x72, 0x61, 0x64,
	0x65, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_grade_v1_grade_proto_rawDescOnce sync.Once
	file_grade_v1_grade_proto_rawDescData = file_grade_v1_grade_proto_rawDesc
)

func file_grade_v1_grade_proto_rawDescGZIP() []byte {
	file_grade_v1_grade_proto_rawDescOnce.Do(func() {
		file_grade_v1_grade_proto_rawDescData = protoimpl.X.CompressGZIP(file_grade_v1_grade_proto_rawDescData)
	})
	return file_grade_v1_grade_proto_rawDescData
}

var file_grade_v1_grade_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_grade_v1_grade_proto_goTypes = []any{
	(*GetGradeByTermReq)(nil),  // 0: grade.v1.GetGradeByTermReq
	(*GetGradeByTermResp)(nil), // 1: grade.v1.GetGradeByTermResp
	(*Grade)(nil),              // 2: grade.v1.Grade
	(*GetGradeScoreReq)(nil),   // 3: grade.v1.GetGradeScoreReq
	(*GetGradeScoreResp)(nil),  // 4: grade.v1.GetGradeScoreResp
	(*TypeOfGradeScore)(nil),   // 5: grade.v1.TypeOfGradeScore
	(*GradeScore)(nil),         // 6: grade.v1.GradeScore
}
var file_grade_v1_grade_proto_depIdxs = []int32{
	2, // 0: grade.v1.GetGradeByTermResp.grades:type_name -> grade.v1.Grade
	5, // 1: grade.v1.GetGradeScoreResp.typeOfGradeScore:type_name -> grade.v1.TypeOfGradeScore
	6, // 2: grade.v1.TypeOfGradeScore.gradeScoreList:type_name -> grade.v1.GradeScore
	0, // 3: grade.v1.GradeService.GetGradeByTerm:input_type -> grade.v1.GetGradeByTermReq
	3, // 4: grade.v1.GradeService.GetGradeScore:input_type -> grade.v1.GetGradeScoreReq
	1, // 5: grade.v1.GradeService.GetGradeByTerm:output_type -> grade.v1.GetGradeByTermResp
	4, // 6: grade.v1.GradeService.GetGradeScore:output_type -> grade.v1.GetGradeScoreResp
	5, // [5:7] is the sub-list for method output_type
	3, // [3:5] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_grade_v1_grade_proto_init() }
func file_grade_v1_grade_proto_init() {
	if File_grade_v1_grade_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_grade_v1_grade_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_grade_v1_grade_proto_goTypes,
		DependencyIndexes: file_grade_v1_grade_proto_depIdxs,
		MessageInfos:      file_grade_v1_grade_proto_msgTypes,
	}.Build()
	File_grade_v1_grade_proto = out.File
	file_grade_v1_grade_proto_rawDesc = nil
	file_grade_v1_grade_proto_goTypes = nil
	file_grade_v1_grade_proto_depIdxs = nil
}
