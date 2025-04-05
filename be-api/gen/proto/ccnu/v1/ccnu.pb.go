// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.12.4
// source: ccnu/v1/ccnu.proto

package ccnuv1

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

type LoginRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StudentId string `protobuf:"bytes,1,opt,name=student_id,json=studentId,proto3" json:"student_id,omitempty"`
	Password  string `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
}

func (x *LoginRequest) Reset() {
	*x = LoginRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ccnu_v1_ccnu_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LoginRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoginRequest) ProtoMessage() {}

func (x *LoginRequest) ProtoReflect() protoreflect.Message {
	mi := &file_ccnu_v1_ccnu_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoginRequest.ProtoReflect.Descriptor instead.
func (*LoginRequest) Descriptor() ([]byte, []int) {
	return file_ccnu_v1_ccnu_proto_rawDescGZIP(), []int{0}
}

func (x *LoginRequest) GetStudentId() string {
	if x != nil {
		return x.StudentId
	}
	return ""
}

func (x *LoginRequest) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

type LoginResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success bool `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
}

func (x *LoginResponse) Reset() {
	*x = LoginResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ccnu_v1_ccnu_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LoginResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoginResponse) ProtoMessage() {}

func (x *LoginResponse) ProtoReflect() protoreflect.Message {
	mi := &file_ccnu_v1_ccnu_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoginResponse.ProtoReflect.Descriptor instead.
func (*LoginResponse) Descriptor() ([]byte, []int) {
	return file_ccnu_v1_ccnu_proto_rawDescGZIP(), []int{1}
}

func (x *LoginResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

type GetCCNUCookieRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StudentId string `protobuf:"bytes,1,opt,name=student_id,json=studentId,proto3" json:"student_id,omitempty"`
	Password  string `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
}

func (x *GetCCNUCookieRequest) Reset() {
	*x = GetCCNUCookieRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ccnu_v1_ccnu_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetCCNUCookieRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCCNUCookieRequest) ProtoMessage() {}

func (x *GetCCNUCookieRequest) ProtoReflect() protoreflect.Message {
	mi := &file_ccnu_v1_ccnu_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCCNUCookieRequest.ProtoReflect.Descriptor instead.
func (*GetCCNUCookieRequest) Descriptor() ([]byte, []int) {
	return file_ccnu_v1_ccnu_proto_rawDescGZIP(), []int{2}
}

func (x *GetCCNUCookieRequest) GetStudentId() string {
	if x != nil {
		return x.StudentId
	}
	return ""
}

func (x *GetCCNUCookieRequest) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

type GetCCNUCookieResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Cookie string `protobuf:"bytes,1,opt,name=cookie,proto3" json:"cookie,omitempty"`
}

func (x *GetCCNUCookieResponse) Reset() {
	*x = GetCCNUCookieResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ccnu_v1_ccnu_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetCCNUCookieResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCCNUCookieResponse) ProtoMessage() {}

func (x *GetCCNUCookieResponse) ProtoReflect() protoreflect.Message {
	mi := &file_ccnu_v1_ccnu_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCCNUCookieResponse.ProtoReflect.Descriptor instead.
func (*GetCCNUCookieResponse) Descriptor() ([]byte, []int) {
	return file_ccnu_v1_ccnu_proto_rawDescGZIP(), []int{3}
}

func (x *GetCCNUCookieResponse) GetCookie() string {
	if x != nil {
		return x.Cookie
	}
	return ""
}

var File_ccnu_v1_ccnu_proto protoreflect.FileDescriptor

var file_ccnu_v1_ccnu_proto_rawDesc = []byte{
	0x0a, 0x12, 0x63, 0x63, 0x6e, 0x75, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x63, 0x6e, 0x75, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x63, 0x63, 0x6e, 0x75, 0x2e, 0x76, 0x31, 0x22, 0x49, 0x0a,
	0x0c, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a,
	0x0a, 0x73, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x09, 0x73, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x1a, 0x0a, 0x08,
	0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x22, 0x29, 0x0a, 0x0d, 0x4c, 0x6f, 0x67, 0x69,
	0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63,
	0x63, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63,
	0x65, 0x73, 0x73, 0x22, 0x51, 0x0a, 0x14, 0x47, 0x65, 0x74, 0x43, 0x43, 0x4e, 0x55, 0x43, 0x6f,
	0x6f, 0x6b, 0x69, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x73,
	0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x09, 0x73, 0x74, 0x75, 0x64, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61,
	0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x61,
	0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x22, 0x2f, 0x0a, 0x15, 0x47, 0x65, 0x74, 0x43, 0x43, 0x4e,
	0x55, 0x43, 0x6f, 0x6f, 0x6b, 0x69, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6f, 0x6b, 0x69, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x63, 0x6f, 0x6f, 0x6b, 0x69, 0x65, 0x32, 0x95, 0x01, 0x0a, 0x0b, 0x43, 0x43, 0x4e, 0x55,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x36, 0x0a, 0x05, 0x4c, 0x6f, 0x67, 0x69, 0x6e,
	0x12, 0x15, 0x2e, 0x63, 0x63, 0x6e, 0x75, 0x2e, 0x76, 0x31, 0x2e, 0x4c, 0x6f, 0x67, 0x69, 0x6e,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x63, 0x63, 0x6e, 0x75, 0x2e, 0x76,
	0x31, 0x2e, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x4e, 0x0a, 0x0d, 0x47, 0x65, 0x74, 0x43, 0x43, 0x4e, 0x55, 0x43, 0x6f, 0x6f, 0x6b, 0x69, 0x65,
	0x12, 0x1d, 0x2e, 0x63, 0x63, 0x6e, 0x75, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x43,
	0x4e, 0x55, 0x43, 0x6f, 0x6f, 0x6b, 0x69, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x1e, 0x2e, 0x63, 0x63, 0x6e, 0x75, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x43, 0x4e,
	0x55, 0x43, 0x6f, 0x6f, 0x6b, 0x69, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42,
	0x40, 0x5a, 0x3e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x73,
	0x79, 0x6e, 0x63, 0x63, 0x6e, 0x75, 0x2f, 0x63, 0x63, 0x6e, 0x75, 0x62, 0x6f, 0x78, 0x2d, 0x62,
	0x65, 0x2f, 0x62, 0x65, 0x2d, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x63, 0x63, 0x6e, 0x75, 0x2f, 0x76, 0x31, 0x3b, 0x63, 0x63, 0x6e, 0x75, 0x76,
	0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_ccnu_v1_ccnu_proto_rawDescOnce sync.Once
	file_ccnu_v1_ccnu_proto_rawDescData = file_ccnu_v1_ccnu_proto_rawDesc
)

func file_ccnu_v1_ccnu_proto_rawDescGZIP() []byte {
	file_ccnu_v1_ccnu_proto_rawDescOnce.Do(func() {
		file_ccnu_v1_ccnu_proto_rawDescData = protoimpl.X.CompressGZIP(file_ccnu_v1_ccnu_proto_rawDescData)
	})
	return file_ccnu_v1_ccnu_proto_rawDescData
}

var file_ccnu_v1_ccnu_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_ccnu_v1_ccnu_proto_goTypes = []interface{}{
	(*LoginRequest)(nil),          // 0: ccnu.v1.LoginRequest
	(*LoginResponse)(nil),         // 1: ccnu.v1.LoginResponse
	(*GetCCNUCookieRequest)(nil),  // 2: ccnu.v1.GetCCNUCookieRequest
	(*GetCCNUCookieResponse)(nil), // 3: ccnu.v1.GetCCNUCookieResponse
}
var file_ccnu_v1_ccnu_proto_depIdxs = []int32{
	0, // 0: ccnu.v1.CCNUService.Login:input_type -> ccnu.v1.LoginRequest
	2, // 1: ccnu.v1.CCNUService.GetCCNUCookie:input_type -> ccnu.v1.GetCCNUCookieRequest
	1, // 2: ccnu.v1.CCNUService.Login:output_type -> ccnu.v1.LoginResponse
	3, // 3: ccnu.v1.CCNUService.GetCCNUCookie:output_type -> ccnu.v1.GetCCNUCookieResponse
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_ccnu_v1_ccnu_proto_init() }
func file_ccnu_v1_ccnu_proto_init() {
	if File_ccnu_v1_ccnu_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_ccnu_v1_ccnu_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LoginRequest); i {
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
		file_ccnu_v1_ccnu_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LoginResponse); i {
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
		file_ccnu_v1_ccnu_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetCCNUCookieRequest); i {
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
		file_ccnu_v1_ccnu_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetCCNUCookieResponse); i {
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
			RawDescriptor: file_ccnu_v1_ccnu_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_ccnu_v1_ccnu_proto_goTypes,
		DependencyIndexes: file_ccnu_v1_ccnu_proto_depIdxs,
		MessageInfos:      file_ccnu_v1_ccnu_proto_msgTypes,
	}.Build()
	File_ccnu_v1_ccnu_proto = out.File
	file_ccnu_v1_ccnu_proto_rawDesc = nil
	file_ccnu_v1_ccnu_proto_goTypes = nil
	file_ccnu_v1_ccnu_proto_depIdxs = nil
}
