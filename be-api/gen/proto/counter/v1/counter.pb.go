// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.12.4
// source: counter/v1/counter.proto

package counterv1

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

type AddCounterReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StudentId string `protobuf:"bytes,1,opt,name=studentId,proto3" json:"studentId,omitempty"` //发送一个studentId增加一次,根据具体的次数划分等级
}

func (x *AddCounterReq) Reset() {
	*x = AddCounterReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_counter_v1_counter_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddCounterReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddCounterReq) ProtoMessage() {}

func (x *AddCounterReq) ProtoReflect() protoreflect.Message {
	mi := &file_counter_v1_counter_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddCounterReq.ProtoReflect.Descriptor instead.
func (*AddCounterReq) Descriptor() ([]byte, []int) {
	return file_counter_v1_counter_proto_rawDescGZIP(), []int{0}
}

func (x *AddCounterReq) GetStudentId() string {
	if x != nil {
		return x.StudentId
	}
	return ""
}

type AddCounterResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *AddCounterResp) Reset() {
	*x = AddCounterResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_counter_v1_counter_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddCounterResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddCounterResp) ProtoMessage() {}

func (x *AddCounterResp) ProtoReflect() protoreflect.Message {
	mi := &file_counter_v1_counter_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddCounterResp.ProtoReflect.Descriptor instead.
func (*AddCounterResp) Descriptor() ([]byte, []int) {
	return file_counter_v1_counter_proto_rawDescGZIP(), []int{1}
}

type GetCounterLevelsReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Label string `protobuf:"bytes,1,opt,name=label,proto3" json:"label,omitempty"`
}

func (x *GetCounterLevelsReq) Reset() {
	*x = GetCounterLevelsReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_counter_v1_counter_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetCounterLevelsReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCounterLevelsReq) ProtoMessage() {}

func (x *GetCounterLevelsReq) ProtoReflect() protoreflect.Message {
	mi := &file_counter_v1_counter_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCounterLevelsReq.ProtoReflect.Descriptor instead.
func (*GetCounterLevelsReq) Descriptor() ([]byte, []int) {
	return file_counter_v1_counter_proto_rawDescGZIP(), []int{2}
}

func (x *GetCounterLevelsReq) GetLabel() string {
	if x != nil {
		return x.Label
	}
	return ""
}

type GetCounterLevelsResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StudentIds []string `protobuf:"bytes,1,rep,name=studentIds,proto3" json:"studentIds,omitempty"`
}

func (x *GetCounterLevelsResp) Reset() {
	*x = GetCounterLevelsResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_counter_v1_counter_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetCounterLevelsResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCounterLevelsResp) ProtoMessage() {}

func (x *GetCounterLevelsResp) ProtoReflect() protoreflect.Message {
	mi := &file_counter_v1_counter_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCounterLevelsResp.ProtoReflect.Descriptor instead.
func (*GetCounterLevelsResp) Descriptor() ([]byte, []int) {
	return file_counter_v1_counter_proto_rawDescGZIP(), []int{3}
}

func (x *GetCounterLevelsResp) GetStudentIds() []string {
	if x != nil {
		return x.StudentIds
	}
	return nil
}

type ChangeCounterLevelsReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StudentIds []string `protobuf:"bytes,1,rep,name=studentIds,proto3" json:"studentIds,omitempty"`
	IsReduce   bool     `protobuf:"varint,2,opt,name=isReduce,proto3" json:"isReduce,omitempty"` //如果是ture表示降低等级，1,3,7表示其等级
	Step       int64    `protobuf:"varint,3,opt,name=step,proto3" json:"step,omitempty"`         //表示降低几个等级,
}

func (x *ChangeCounterLevelsReq) Reset() {
	*x = ChangeCounterLevelsReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_counter_v1_counter_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ChangeCounterLevelsReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChangeCounterLevelsReq) ProtoMessage() {}

func (x *ChangeCounterLevelsReq) ProtoReflect() protoreflect.Message {
	mi := &file_counter_v1_counter_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChangeCounterLevelsReq.ProtoReflect.Descriptor instead.
func (*ChangeCounterLevelsReq) Descriptor() ([]byte, []int) {
	return file_counter_v1_counter_proto_rawDescGZIP(), []int{4}
}

func (x *ChangeCounterLevelsReq) GetStudentIds() []string {
	if x != nil {
		return x.StudentIds
	}
	return nil
}

func (x *ChangeCounterLevelsReq) GetIsReduce() bool {
	if x != nil {
		return x.IsReduce
	}
	return false
}

func (x *ChangeCounterLevelsReq) GetStep() int64 {
	if x != nil {
		return x.Step
	}
	return 0
}

type ChangeCounterLevelsResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ChangeCounterLevelsResp) Reset() {
	*x = ChangeCounterLevelsResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_counter_v1_counter_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ChangeCounterLevelsResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChangeCounterLevelsResp) ProtoMessage() {}

func (x *ChangeCounterLevelsResp) ProtoReflect() protoreflect.Message {
	mi := &file_counter_v1_counter_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChangeCounterLevelsResp.ProtoReflect.Descriptor instead.
func (*ChangeCounterLevelsResp) Descriptor() ([]byte, []int) {
	return file_counter_v1_counter_proto_rawDescGZIP(), []int{5}
}

type ClearCounterLevelsReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ClearCounterLevelsReq) Reset() {
	*x = ClearCounterLevelsReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_counter_v1_counter_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ClearCounterLevelsReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClearCounterLevelsReq) ProtoMessage() {}

func (x *ClearCounterLevelsReq) ProtoReflect() protoreflect.Message {
	mi := &file_counter_v1_counter_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClearCounterLevelsReq.ProtoReflect.Descriptor instead.
func (*ClearCounterLevelsReq) Descriptor() ([]byte, []int) {
	return file_counter_v1_counter_proto_rawDescGZIP(), []int{6}
}

type ClearCounterLevelsResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ClearCounterLevelsResp) Reset() {
	*x = ClearCounterLevelsResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_counter_v1_counter_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ClearCounterLevelsResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClearCounterLevelsResp) ProtoMessage() {}

func (x *ClearCounterLevelsResp) ProtoReflect() protoreflect.Message {
	mi := &file_counter_v1_counter_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClearCounterLevelsResp.ProtoReflect.Descriptor instead.
func (*ClearCounterLevelsResp) Descriptor() ([]byte, []int) {
	return file_counter_v1_counter_proto_rawDescGZIP(), []int{7}
}

var File_counter_v1_counter_proto protoreflect.FileDescriptor

var file_counter_v1_counter_proto_rawDesc = []byte{
	0x0a, 0x18, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x22, 0x2d, 0x0a, 0x0d, 0x41, 0x64, 0x64, 0x43, 0x6f, 0x75,
	0x6e, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x74, 0x75, 0x64, 0x65,
	0x6e, 0x74, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x74, 0x75, 0x64,
	0x65, 0x6e, 0x74, 0x49, 0x64, 0x22, 0x10, 0x0a, 0x0e, 0x41, 0x64, 0x64, 0x43, 0x6f, 0x75, 0x6e,
	0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x22, 0x2b, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x43, 0x6f,
	0x75, 0x6e, 0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x52, 0x65, 0x71, 0x12, 0x14,
	0x0a, 0x05, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6c,
	0x61, 0x62, 0x65, 0x6c, 0x22, 0x36, 0x0a, 0x14, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74,
	0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x52, 0x65, 0x73, 0x70, 0x12, 0x1e, 0x0a, 0x0a,
	0x73, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x0a, 0x73, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x73, 0x22, 0x68, 0x0a, 0x16,
	0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x4c, 0x65, 0x76,
	0x65, 0x6c, 0x73, 0x52, 0x65, 0x71, 0x12, 0x1e, 0x0a, 0x0a, 0x73, 0x74, 0x75, 0x64, 0x65, 0x6e,
	0x74, 0x49, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0a, 0x73, 0x74, 0x75, 0x64,
	0x65, 0x6e, 0x74, 0x49, 0x64, 0x73, 0x12, 0x1a, 0x0a, 0x08, 0x69, 0x73, 0x52, 0x65, 0x64, 0x75,
	0x63, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x69, 0x73, 0x52, 0x65, 0x64, 0x75,
	0x63, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x74, 0x65, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x04, 0x73, 0x74, 0x65, 0x70, 0x22, 0x19, 0x0a, 0x17, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65,
	0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x22, 0x17, 0x0a, 0x15, 0x43, 0x6c, 0x65, 0x61, 0x72, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65,
	0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x52, 0x65, 0x71, 0x22, 0x18, 0x0a, 0x16, 0x43, 0x6c,
	0x65, 0x61, 0x72, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73,
	0x52, 0x65, 0x73, 0x70, 0x32, 0xe9, 0x02, 0x0a, 0x0e, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x43, 0x0a, 0x0a, 0x41, 0x64, 0x64, 0x43, 0x6f,
	0x75, 0x6e, 0x74, 0x65, 0x72, 0x12, 0x19, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x2e,
	0x76, 0x31, 0x2e, 0x41, 0x64, 0x64, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71,
	0x1a, 0x1a, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x64,
	0x64, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x12, 0x55, 0x0a, 0x10,
	0x47, 0x65, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73,
	0x12, 0x1f, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65,
	0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x52, 0x65,
	0x71, 0x1a, 0x20, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x47,
	0x65, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x52,
	0x65, 0x73, 0x70, 0x12, 0x5e, 0x0a, 0x13, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x43, 0x6f, 0x75,
	0x6e, 0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x12, 0x22, 0x2e, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x43, 0x6f,
	0x75, 0x6e, 0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x52, 0x65, 0x71, 0x1a, 0x23,
	0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x68, 0x61, 0x6e,
	0x67, 0x65, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x52,
	0x65, 0x73, 0x70, 0x12, 0x5b, 0x0a, 0x12, 0x43, 0x6c, 0x65, 0x61, 0x72, 0x43, 0x6f, 0x75, 0x6e,
	0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x12, 0x21, 0x2e, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6c, 0x65, 0x61, 0x72, 0x43, 0x6f, 0x75, 0x6e,
	0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x52, 0x65, 0x71, 0x1a, 0x22, 0x2e, 0x63,
	0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6c, 0x65, 0x61, 0x72, 0x43,
	0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x73, 0x52, 0x65, 0x73, 0x70,
	0x42, 0x46, 0x5a, 0x44, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61,
	0x73, 0x79, 0x6e, 0x63, 0x63, 0x6e, 0x75, 0x2f, 0x63, 0x63, 0x6e, 0x75, 0x62, 0x6f, 0x78, 0x2d,
	0x62, 0x65, 0x2f, 0x62, 0x65, 0x2d, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x3b, 0x63,
	0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_counter_v1_counter_proto_rawDescOnce sync.Once
	file_counter_v1_counter_proto_rawDescData = file_counter_v1_counter_proto_rawDesc
)

func file_counter_v1_counter_proto_rawDescGZIP() []byte {
	file_counter_v1_counter_proto_rawDescOnce.Do(func() {
		file_counter_v1_counter_proto_rawDescData = protoimpl.X.CompressGZIP(file_counter_v1_counter_proto_rawDescData)
	})
	return file_counter_v1_counter_proto_rawDescData
}

var file_counter_v1_counter_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_counter_v1_counter_proto_goTypes = []interface{}{
	(*AddCounterReq)(nil),           // 0: counter.v1.AddCounterReq
	(*AddCounterResp)(nil),          // 1: counter.v1.AddCounterResp
	(*GetCounterLevelsReq)(nil),     // 2: counter.v1.GetCounterLevelsReq
	(*GetCounterLevelsResp)(nil),    // 3: counter.v1.GetCounterLevelsResp
	(*ChangeCounterLevelsReq)(nil),  // 4: counter.v1.ChangeCounterLevelsReq
	(*ChangeCounterLevelsResp)(nil), // 5: counter.v1.ChangeCounterLevelsResp
	(*ClearCounterLevelsReq)(nil),   // 6: counter.v1.ClearCounterLevelsReq
	(*ClearCounterLevelsResp)(nil),  // 7: counter.v1.ClearCounterLevelsResp
}
var file_counter_v1_counter_proto_depIdxs = []int32{
	0, // 0: counter.v1.CounterService.AddCounter:input_type -> counter.v1.AddCounterReq
	2, // 1: counter.v1.CounterService.GetCounterLevels:input_type -> counter.v1.GetCounterLevelsReq
	4, // 2: counter.v1.CounterService.ChangeCounterLevels:input_type -> counter.v1.ChangeCounterLevelsReq
	6, // 3: counter.v1.CounterService.ClearCounterLevels:input_type -> counter.v1.ClearCounterLevelsReq
	1, // 4: counter.v1.CounterService.AddCounter:output_type -> counter.v1.AddCounterResp
	3, // 5: counter.v1.CounterService.GetCounterLevels:output_type -> counter.v1.GetCounterLevelsResp
	5, // 6: counter.v1.CounterService.ChangeCounterLevels:output_type -> counter.v1.ChangeCounterLevelsResp
	7, // 7: counter.v1.CounterService.ClearCounterLevels:output_type -> counter.v1.ClearCounterLevelsResp
	4, // [4:8] is the sub-list for method output_type
	0, // [0:4] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_counter_v1_counter_proto_init() }
func file_counter_v1_counter_proto_init() {
	if File_counter_v1_counter_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_counter_v1_counter_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddCounterReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_counter_v1_counter_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddCounterResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_counter_v1_counter_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetCounterLevelsReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_counter_v1_counter_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetCounterLevelsResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_counter_v1_counter_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ChangeCounterLevelsReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_counter_v1_counter_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ChangeCounterLevelsResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_counter_v1_counter_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ClearCounterLevelsReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_counter_v1_counter_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ClearCounterLevelsResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_counter_v1_counter_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_counter_v1_counter_proto_goTypes,
		DependencyIndexes: file_counter_v1_counter_proto_depIdxs,
		MessageInfos:      file_counter_v1_counter_proto_msgTypes,
	}.Build()
	File_counter_v1_counter_proto = out.File
	file_counter_v1_counter_proto_rawDesc = nil
	file_counter_v1_counter_proto_goTypes = nil
	file_counter_v1_counter_proto_depIdxs = nil
}
