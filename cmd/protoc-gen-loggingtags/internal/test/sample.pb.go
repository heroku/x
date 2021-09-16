// Code generated by protoc-gen-go. DO NOT EDIT.
// source: cmd/protoc-gen-loggingtags/internal/test/sample.proto

package test

import (
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
	duration "github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
	_ "github.com/heroku/x/loggingtags"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Sample struct {
	Safe                 string               `protobuf:"bytes,1,opt,name=safe,proto3" json:"safe,omitempty"`
	Unsafe               string               `protobuf:"bytes,2,opt,name=unsafe,proto3" json:"unsafe,omitempty"`
	Timestamp            *timestamp.Timestamp `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Duration             *duration.Duration   `protobuf:"bytes,4,opt,name=duration,proto3" json:"duration,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *Sample) Reset()         { *m = Sample{} }
func (m *Sample) String() string { return proto.CompactTextString(m) }
func (*Sample) ProtoMessage()    {}
func (*Sample) Descriptor() ([]byte, []int) {
	return fileDescriptor_5293983fff447228, []int{0}
}

func (m *Sample) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Sample.Unmarshal(m, b)
}
func (m *Sample) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Sample.Marshal(b, m, deterministic)
}
func (m *Sample) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Sample.Merge(m, src)
}
func (m *Sample) XXX_Size() int {
	return xxx_messageInfo_Sample.Size(m)
}
func (m *Sample) XXX_DiscardUnknown() {
	xxx_messageInfo_Sample.DiscardUnknown(m)
}

var xxx_messageInfo_Sample proto.InternalMessageInfo

func (m *Sample) GetSafe() string {
	if m != nil {
		return m.Safe
	}
	return ""
}

func (m *Sample) GetUnsafe() string {
	if m != nil {
		return m.Unsafe
	}
	return ""
}

func (m *Sample) GetTimestamp() *timestamp.Timestamp {
	if m != nil {
		return m.Timestamp
	}
	return nil
}

func (m *Sample) GetDuration() *duration.Duration {
	if m != nil {
		return m.Duration
	}
	return nil
}

type NestedSample struct {
	Data                 *Sample  `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NestedSample) Reset()         { *m = NestedSample{} }
func (m *NestedSample) String() string { return proto.CompactTextString(m) }
func (*NestedSample) ProtoMessage()    {}
func (*NestedSample) Descriptor() ([]byte, []int) {
	return fileDescriptor_5293983fff447228, []int{1}
}

func (m *NestedSample) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NestedSample.Unmarshal(m, b)
}
func (m *NestedSample) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NestedSample.Marshal(b, m, deterministic)
}
func (m *NestedSample) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NestedSample.Merge(m, src)
}
func (m *NestedSample) XXX_Size() int {
	return xxx_messageInfo_NestedSample.Size(m)
}
func (m *NestedSample) XXX_DiscardUnknown() {
	xxx_messageInfo_NestedSample.DiscardUnknown(m)
}

var xxx_messageInfo_NestedSample proto.InternalMessageInfo

func (m *NestedSample) GetData() *Sample {
	if m != nil {
		return m.Data
	}
	return nil
}

func init() {
	proto.RegisterType((*Sample)(nil), "test.Sample")
	proto.RegisterType((*NestedSample)(nil), "test.NestedSample")
}

func init() {
	proto.RegisterFile("cmd/protoc-gen-loggingtags/internal/test/sample.proto", fileDescriptor_5293983fff447228)
}

var fileDescriptor_5293983fff447228 = []byte{
	// 254 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x90, 0x31, 0x4f, 0xc3, 0x30,
	0x10, 0x85, 0x65, 0xb0, 0x22, 0xea, 0x76, 0xf2, 0x50, 0x99, 0x0c, 0x50, 0x75, 0x40, 0x5d, 0x6a,
	0x4b, 0x45, 0xb0, 0x20, 0x31, 0x20, 0x66, 0x86, 0xc2, 0x1f, 0x70, 0x9b, 0xab, 0x15, 0x29, 0xb1,
	0xa3, 0xf8, 0xb2, 0xf3, 0xa7, 0x10, 0x7f, 0x0f, 0xd9, 0xb1, 0x03, 0xa2, 0xa3, 0xef, 0xbd, 0xef,
	0xde, 0x3b, 0xb3, 0x87, 0x63, 0x5b, 0xa9, 0xae, 0x77, 0xe8, 0x8e, 0x5b, 0x03, 0x76, 0xdb, 0x38,
	0x63, 0x6a, 0x6b, 0x50, 0x1b, 0xaf, 0x6a, 0x8b, 0xd0, 0x5b, 0xdd, 0x28, 0x04, 0x8f, 0xca, 0xeb,
	0xb6, 0x6b, 0x40, 0x46, 0x2f, 0xa7, 0x61, 0x54, 0xde, 0x18, 0xe7, 0x4c, 0x03, 0x23, 0x7f, 0x18,
	0x4e, 0xaa, 0x1a, 0x7a, 0x8d, 0xb5, 0xb3, 0xa3, 0xab, 0xbc, 0xfd, 0xaf, 0x63, 0xdd, 0x82, 0x47,
	0xdd, 0x76, 0xc9, 0xb0, 0xfc, 0x1b, 0xe7, 0xf5, 0x29, 0xad, 0x5f, 0x7f, 0x13, 0x56, 0xbc, 0xc7,
	0x3c, 0x2e, 0x18, 0x0d, 0x82, 0x20, 0x2b, 0xb2, 0x99, 0xbd, 0xd0, 0xcf, 0x2f, 0x41, 0xf6, 0x71,
	0xc2, 0x97, 0xac, 0x18, 0x6c, 0xd4, 0x2e, 0x82, 0xb6, 0x4f, 0x2f, 0xfe, 0xcc, 0x66, 0x53, 0x8e,
	0xb8, 0x5c, 0x91, 0xcd, 0x7c, 0x57, 0xca, 0xb1, 0x89, 0xcc, 0x4d, 0xe4, 0x47, 0x76, 0xa4, 0x95,
	0xbf, 0x08, 0x7f, 0x62, 0x57, 0xf9, 0x0e, 0x41, 0x23, 0x7e, 0x7d, 0x86, 0xbf, 0x26, 0x43, 0xa2,
	0x27, 0x60, 0xfd, 0xc8, 0x16, 0x6f, 0xe0, 0x11, 0xaa, 0x54, 0xff, 0x8e, 0xd1, 0x4a, 0xa3, 0x8e,
	0xf5, 0xe7, 0xbb, 0x85, 0x0c, 0xff, 0x26, 0x47, 0x2d, 0x1f, 0x13, 0xf4, 0x43, 0x11, 0x57, 0xdf,
	0xff, 0x04, 0x00, 0x00, 0xff, 0xff, 0xc8, 0x70, 0xb3, 0xe9, 0x90, 0x01, 0x00, 0x00,
}
