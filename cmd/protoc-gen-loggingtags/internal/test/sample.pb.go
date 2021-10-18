// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.11.2
// source: cmd/protoc-gen-loggingtags/internal/test/sample.proto

package test

import (
	duration "github.com/golang/protobuf/ptypes/duration"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	_ "github.com/heroku/x/loggingtags"
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

type Sample struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Safe      string               `protobuf:"bytes,1,opt,name=safe,proto3" json:"safe,omitempty"`
	Unsafe    string               `protobuf:"bytes,2,opt,name=unsafe,proto3" json:"unsafe,omitempty"`
	Timestamp *timestamp.Timestamp `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Duration  *duration.Duration   `protobuf:"bytes,4,opt,name=duration,proto3" json:"duration,omitempty"`
}

func (x *Sample) Reset() {
	*x = Sample{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Sample) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Sample) ProtoMessage() {}

func (x *Sample) ProtoReflect() protoreflect.Message {
	mi := &file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Sample.ProtoReflect.Descriptor instead.
func (*Sample) Descriptor() ([]byte, []int) {
	return file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDescGZIP(), []int{0}
}

func (x *Sample) GetSafe() string {
	if x != nil {
		return x.Safe
	}
	return ""
}

func (x *Sample) GetUnsafe() string {
	if x != nil {
		return x.Unsafe
	}
	return ""
}

func (x *Sample) GetTimestamp() *timestamp.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *Sample) GetDuration() *duration.Duration {
	if x != nil {
		return x.Duration
	}
	return nil
}

type NestedSample struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data *Sample `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *NestedSample) Reset() {
	*x = NestedSample{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NestedSample) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NestedSample) ProtoMessage() {}

func (x *NestedSample) ProtoReflect() protoreflect.Message {
	mi := &file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NestedSample.ProtoReflect.Descriptor instead.
func (*NestedSample) Descriptor() ([]byte, []int) {
	return file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDescGZIP(), []int{1}
}

func (x *NestedSample) GetData() *Sample {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_cmd_protoc_gen_loggingtags_internal_test_sample_proto protoreflect.FileDescriptor

var file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDesc = []byte{
	0x0a, 0x35, 0x63, 0x6d, 0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67, 0x65, 0x6e,
	0x2d, 0x6c, 0x6f, 0x67, 0x67, 0x69, 0x6e, 0x67, 0x74, 0x61, 0x67, 0x73, 0x2f, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x73, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x04, 0x74, 0x65, 0x73, 0x74, 0x1a, 0x1e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64,
	0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x16,
	0x6c, 0x6f, 0x67, 0x67, 0x69, 0x6e, 0x67, 0x74, 0x61, 0x67, 0x73, 0x2f, 0x73, 0x61, 0x66, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xb7, 0x01, 0x0a, 0x06, 0x53, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x12, 0x18, 0x0a, 0x04, 0x73, 0x61, 0x66, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42,
	0x04, 0x80, 0xb5, 0x18, 0x01, 0x52, 0x04, 0x73, 0x61, 0x66, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x75,
	0x6e, 0x73, 0x61, 0x66, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x75, 0x6e, 0x73,
	0x61, 0x66, 0x65, 0x12, 0x3e, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x42, 0x04, 0x80, 0xb5, 0x18, 0x01, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x12, 0x3b, 0x0a, 0x08, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x42, 0x04, 0x80, 0xb5, 0x18, 0x01, 0x52, 0x08, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x22, 0x36, 0x0a, 0x0c, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65,
	0x12, 0x26, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0c,
	0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x42, 0x04, 0x80, 0xb5,
	0x18, 0x01, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x42, 0x3e, 0x5a, 0x3c, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x68, 0x65, 0x72, 0x6f, 0x6b, 0x75, 0x2f, 0x78, 0x2f,
	0x63, 0x6d, 0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67, 0x65, 0x6e, 0x2d, 0x6c,
	0x6f, 0x67, 0x67, 0x69, 0x6e, 0x67, 0x74, 0x61, 0x67, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDescOnce sync.Once
	file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDescData = file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDesc
)

func file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDescGZIP() []byte {
	file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDescOnce.Do(func() {
		file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDescData = protoimpl.X.CompressGZIP(file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDescData)
	})
	return file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDescData
}

var file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_goTypes = []interface{}{
	(*Sample)(nil),              // 0: test.Sample
	(*NestedSample)(nil),        // 1: test.NestedSample
	(*timestamp.Timestamp)(nil), // 2: google.protobuf.Timestamp
	(*duration.Duration)(nil),   // 3: google.protobuf.Duration
}
var file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_depIdxs = []int32{
	2, // 0: test.Sample.timestamp:type_name -> google.protobuf.Timestamp
	3, // 1: test.Sample.duration:type_name -> google.protobuf.Duration
	0, // 2: test.NestedSample.data:type_name -> test.Sample
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_init() }
func file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_init() {
	if File_cmd_protoc_gen_loggingtags_internal_test_sample_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Sample); i {
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
		file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NestedSample); i {
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
			RawDescriptor: file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_goTypes,
		DependencyIndexes: file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_depIdxs,
		MessageInfos:      file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_msgTypes,
	}.Build()
	File_cmd_protoc_gen_loggingtags_internal_test_sample_proto = out.File
	file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_rawDesc = nil
	file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_goTypes = nil
	file_cmd_protoc_gen_loggingtags_internal_test_sample_proto_depIdxs = nil
}
