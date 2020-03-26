// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/micro/go-micro/auth/service/proto/accounts/accounts.proto

package go_micro_auth

import (
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
	auth "github.com/micro/go-micro/v2/auth/service/proto/auth"
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

type ListRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListRequest) Reset()         { *m = ListRequest{} }
func (m *ListRequest) String() string { return proto.CompactTextString(m) }
func (*ListRequest) ProtoMessage()    {}
func (*ListRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_25929ace37374fcc, []int{0}
}

func (m *ListRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListRequest.Unmarshal(m, b)
}
func (m *ListRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListRequest.Marshal(b, m, deterministic)
}
func (m *ListRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListRequest.Merge(m, src)
}
func (m *ListRequest) XXX_Size() int {
	return xxx_messageInfo_ListRequest.Size(m)
}
func (m *ListRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListRequest proto.InternalMessageInfo

type ListResponse struct {
	Accounts             []*auth.Account `protobuf:"bytes,1,rep,name=accounts,proto3" json:"accounts,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *ListResponse) Reset()         { *m = ListResponse{} }
func (m *ListResponse) String() string { return proto.CompactTextString(m) }
func (*ListResponse) ProtoMessage()    {}
func (*ListResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_25929ace37374fcc, []int{1}
}

func (m *ListResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListResponse.Unmarshal(m, b)
}
func (m *ListResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListResponse.Marshal(b, m, deterministic)
}
func (m *ListResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListResponse.Merge(m, src)
}
func (m *ListResponse) XXX_Size() int {
	return xxx_messageInfo_ListResponse.Size(m)
}
func (m *ListResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListResponse proto.InternalMessageInfo

func (m *ListResponse) GetAccounts() []*auth.Account {
	if m != nil {
		return m.Accounts
	}
	return nil
}

func init() {
	proto.RegisterType((*ListRequest)(nil), "go.micro.auth.ListRequest")
	proto.RegisterType((*ListResponse)(nil), "go.micro.auth.ListResponse")
}

func init() {
	proto.RegisterFile("github.com/micro/go-micro/auth/service/proto/accounts/accounts.proto", fileDescriptor_25929ace37374fcc)
}

var fileDescriptor_25929ace37374fcc = []byte{
	// 184 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x72, 0x49, 0xcf, 0x2c, 0xc9,
	0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0xcf, 0xcd, 0x4c, 0x2e, 0xca, 0xd7, 0x4f, 0xcf, 0xd7,
	0x85, 0x30, 0x12, 0x4b, 0x4b, 0x32, 0xf4, 0x8b, 0x53, 0x8b, 0xca, 0x32, 0x93, 0x53, 0xf5, 0x0b,
	0x8a, 0xf2, 0x4b, 0xf2, 0xf5, 0x13, 0x93, 0x93, 0xf3, 0x4b, 0xf3, 0x4a, 0x8a, 0xe1, 0x0c, 0x3d,
	0xb0, 0xb8, 0x10, 0x6f, 0x7a, 0xbe, 0x1e, 0x58, 0x93, 0x1e, 0x48, 0x93, 0x94, 0x0d, 0x69, 0x86,
	0x82, 0x84, 0x40, 0x04, 0xc4, 0x30, 0x25, 0x5e, 0x2e, 0x6e, 0x9f, 0xcc, 0xe2, 0x92, 0xa0, 0xd4,
	0xc2, 0xd2, 0xd4, 0xe2, 0x12, 0x25, 0x27, 0x2e, 0x1e, 0x08, 0xb7, 0xb8, 0x20, 0x3f, 0xaf, 0x38,
	0x55, 0xc8, 0x88, 0x8b, 0x03, 0x66, 0xbb, 0x04, 0xa3, 0x02, 0xb3, 0x06, 0xb7, 0x91, 0x98, 0x1e,
	0x8a, 0xf5, 0x7a, 0x8e, 0x10, 0xe9, 0x20, 0xb8, 0x3a, 0x23, 0x5f, 0x2e, 0x0e, 0xa8, 0x60, 0xb1,
	0x90, 0x23, 0x17, 0x0b, 0xc8, 0x3c, 0x21, 0x29, 0x34, 0x5d, 0x48, 0x76, 0x4a, 0x49, 0x63, 0x95,
	0x83, 0x38, 0x40, 0x89, 0x21, 0x89, 0x0d, 0xec, 0x50, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff,
	0x83, 0x8d, 0x7c, 0xb2, 0x3d, 0x01, 0x00, 0x00,
}
