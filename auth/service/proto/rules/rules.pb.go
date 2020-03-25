// Code generated by protoc-gen-go. DO NOT EDIT.
// source: auth/service/proto/rules/rules.proto

package go_micro_auth

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
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

type Access int32

const (
	Access_UNKNOWN Access = 0
	Access_GRANTED Access = 1
	Access_DENIED  Access = 2
)

var Access_name = map[int32]string{
	0: "UNKNOWN",
	1: "GRANTED",
	2: "DENIED",
}

var Access_value = map[string]int32{
	"UNKNOWN": 0,
	"GRANTED": 1,
	"DENIED":  2,
}

func (x Access) String() string {
	return proto.EnumName(Access_name, int32(x))
}

func (Access) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_82162781dd332cfa, []int{0}
}

type Rule struct {
	Id                   string    `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Role                 string    `protobuf:"bytes,2,opt,name=role,proto3" json:"role,omitempty"`
	Resource             *Resource `protobuf:"bytes,3,opt,name=resource,proto3" json:"resource,omitempty"`
	Access               Access    `protobuf:"varint,4,opt,name=access,proto3,enum=go.micro.auth.Access" json:"access,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *Rule) Reset()         { *m = Rule{} }
func (m *Rule) String() string { return proto.CompactTextString(m) }
func (*Rule) ProtoMessage()    {}
func (*Rule) Descriptor() ([]byte, []int) {
	return fileDescriptor_82162781dd332cfa, []int{0}
}

func (m *Rule) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Rule.Unmarshal(m, b)
}
func (m *Rule) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Rule.Marshal(b, m, deterministic)
}
func (m *Rule) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Rule.Merge(m, src)
}
func (m *Rule) XXX_Size() int {
	return xxx_messageInfo_Rule.Size(m)
}
func (m *Rule) XXX_DiscardUnknown() {
	xxx_messageInfo_Rule.DiscardUnknown(m)
}

var xxx_messageInfo_Rule proto.InternalMessageInfo

func (m *Rule) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Rule) GetRole() string {
	if m != nil {
		return m.Role
	}
	return ""
}

func (m *Rule) GetResource() *Resource {
	if m != nil {
		return m.Resource
	}
	return nil
}

func (m *Rule) GetAccess() Access {
	if m != nil {
		return m.Access
	}
	return Access_UNKNOWN
}

type Resource struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Type                 string   `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Endpoint             string   `protobuf:"bytes,3,opt,name=endpoint,proto3" json:"endpoint,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Resource) Reset()         { *m = Resource{} }
func (m *Resource) String() string { return proto.CompactTextString(m) }
func (*Resource) ProtoMessage()    {}
func (*Resource) Descriptor() ([]byte, []int) {
	return fileDescriptor_82162781dd332cfa, []int{1}
}

func (m *Resource) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Resource.Unmarshal(m, b)
}
func (m *Resource) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Resource.Marshal(b, m, deterministic)
}
func (m *Resource) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Resource.Merge(m, src)
}
func (m *Resource) XXX_Size() int {
	return xxx_messageInfo_Resource.Size(m)
}
func (m *Resource) XXX_DiscardUnknown() {
	xxx_messageInfo_Resource.DiscardUnknown(m)
}

var xxx_messageInfo_Resource proto.InternalMessageInfo

func (m *Resource) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Resource) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *Resource) GetEndpoint() string {
	if m != nil {
		return m.Endpoint
	}
	return ""
}

type CreateRequest struct {
	Role                 string    `protobuf:"bytes,1,opt,name=role,proto3" json:"role,omitempty"`
	Resource             *Resource `protobuf:"bytes,2,opt,name=resource,proto3" json:"resource,omitempty"`
	Access               Access    `protobuf:"varint,3,opt,name=access,proto3,enum=go.micro.auth.Access" json:"access,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *CreateRequest) Reset()         { *m = CreateRequest{} }
func (m *CreateRequest) String() string { return proto.CompactTextString(m) }
func (*CreateRequest) ProtoMessage()    {}
func (*CreateRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_82162781dd332cfa, []int{2}
}

func (m *CreateRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateRequest.Unmarshal(m, b)
}
func (m *CreateRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateRequest.Marshal(b, m, deterministic)
}
func (m *CreateRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateRequest.Merge(m, src)
}
func (m *CreateRequest) XXX_Size() int {
	return xxx_messageInfo_CreateRequest.Size(m)
}
func (m *CreateRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CreateRequest proto.InternalMessageInfo

func (m *CreateRequest) GetRole() string {
	if m != nil {
		return m.Role
	}
	return ""
}

func (m *CreateRequest) GetResource() *Resource {
	if m != nil {
		return m.Resource
	}
	return nil
}

func (m *CreateRequest) GetAccess() Access {
	if m != nil {
		return m.Access
	}
	return Access_UNKNOWN
}

type CreateResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CreateResponse) Reset()         { *m = CreateResponse{} }
func (m *CreateResponse) String() string { return proto.CompactTextString(m) }
func (*CreateResponse) ProtoMessage()    {}
func (*CreateResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_82162781dd332cfa, []int{3}
}

func (m *CreateResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateResponse.Unmarshal(m, b)
}
func (m *CreateResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateResponse.Marshal(b, m, deterministic)
}
func (m *CreateResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateResponse.Merge(m, src)
}
func (m *CreateResponse) XXX_Size() int {
	return xxx_messageInfo_CreateResponse.Size(m)
}
func (m *CreateResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateResponse.DiscardUnknown(m)
}

var xxx_messageInfo_CreateResponse proto.InternalMessageInfo

type DeleteRequest struct {
	Role                 string    `protobuf:"bytes,1,opt,name=role,proto3" json:"role,omitempty"`
	Resource             *Resource `protobuf:"bytes,2,opt,name=resource,proto3" json:"resource,omitempty"`
	Access               Access    `protobuf:"varint,3,opt,name=access,proto3,enum=go.micro.auth.Access" json:"access,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *DeleteRequest) Reset()         { *m = DeleteRequest{} }
func (m *DeleteRequest) String() string { return proto.CompactTextString(m) }
func (*DeleteRequest) ProtoMessage()    {}
func (*DeleteRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_82162781dd332cfa, []int{4}
}

func (m *DeleteRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteRequest.Unmarshal(m, b)
}
func (m *DeleteRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteRequest.Marshal(b, m, deterministic)
}
func (m *DeleteRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteRequest.Merge(m, src)
}
func (m *DeleteRequest) XXX_Size() int {
	return xxx_messageInfo_DeleteRequest.Size(m)
}
func (m *DeleteRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteRequest proto.InternalMessageInfo

func (m *DeleteRequest) GetRole() string {
	if m != nil {
		return m.Role
	}
	return ""
}

func (m *DeleteRequest) GetResource() *Resource {
	if m != nil {
		return m.Resource
	}
	return nil
}

func (m *DeleteRequest) GetAccess() Access {
	if m != nil {
		return m.Access
	}
	return Access_UNKNOWN
}

type DeleteResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DeleteResponse) Reset()         { *m = DeleteResponse{} }
func (m *DeleteResponse) String() string { return proto.CompactTextString(m) }
func (*DeleteResponse) ProtoMessage()    {}
func (*DeleteResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_82162781dd332cfa, []int{5}
}

func (m *DeleteResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteResponse.Unmarshal(m, b)
}
func (m *DeleteResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteResponse.Marshal(b, m, deterministic)
}
func (m *DeleteResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteResponse.Merge(m, src)
}
func (m *DeleteResponse) XXX_Size() int {
	return xxx_messageInfo_DeleteResponse.Size(m)
}
func (m *DeleteResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteResponse proto.InternalMessageInfo

type ListRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListRequest) Reset()         { *m = ListRequest{} }
func (m *ListRequest) String() string { return proto.CompactTextString(m) }
func (*ListRequest) ProtoMessage()    {}
func (*ListRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_82162781dd332cfa, []int{6}
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
	Rules                []*Rule  `protobuf:"bytes,1,rep,name=rules,proto3" json:"rules,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListResponse) Reset()         { *m = ListResponse{} }
func (m *ListResponse) String() string { return proto.CompactTextString(m) }
func (*ListResponse) ProtoMessage()    {}
func (*ListResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_82162781dd332cfa, []int{7}
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

func (m *ListResponse) GetRules() []*Rule {
	if m != nil {
		return m.Rules
	}
	return nil
}

func init() {
	proto.RegisterEnum("go.micro.auth.Access", Access_name, Access_value)
	proto.RegisterType((*Rule)(nil), "go.micro.auth.Rule")
	proto.RegisterType((*Resource)(nil), "go.micro.auth.Resource")
	proto.RegisterType((*CreateRequest)(nil), "go.micro.auth.CreateRequest")
	proto.RegisterType((*CreateResponse)(nil), "go.micro.auth.CreateResponse")
	proto.RegisterType((*DeleteRequest)(nil), "go.micro.auth.DeleteRequest")
	proto.RegisterType((*DeleteResponse)(nil), "go.micro.auth.DeleteResponse")
	proto.RegisterType((*ListRequest)(nil), "go.micro.auth.ListRequest")
	proto.RegisterType((*ListResponse)(nil), "go.micro.auth.ListResponse")
}

func init() {
	proto.RegisterFile("auth/service/proto/rules/rules.proto", fileDescriptor_82162781dd332cfa)
}

var fileDescriptor_82162781dd332cfa = []byte{
	// 391 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xcc, 0x93, 0xcb, 0x0a, 0xd3, 0x40,
	0x14, 0x86, 0x3b, 0x69, 0x1a, 0xdb, 0x13, 0x5b, 0xc2, 0x88, 0x18, 0xa2, 0x42, 0x08, 0x2e, 0xa2,
	0x60, 0x0a, 0xe9, 0xca, 0x65, 0x31, 0xa5, 0x88, 0x12, 0x61, 0x50, 0x5c, 0xc7, 0xf4, 0xa0, 0x81,
	0x34, 0x13, 0x67, 0x12, 0xc1, 0x27, 0x70, 0xe7, 0x13, 0xfa, 0x30, 0x32, 0x93, 0x0b, 0x36, 0xb5,
	0xa0, 0x3b, 0x37, 0xe1, 0xdc, 0xe6, 0xcf, 0xf7, 0x9f, 0x64, 0xe0, 0x49, 0xd6, 0x36, 0x9f, 0xb7,
	0x12, 0xc5, 0xd7, 0x22, 0xc7, 0x6d, 0x2d, 0x78, 0xc3, 0xb7, 0xa2, 0x2d, 0x51, 0x76, 0xcf, 0x48,
	0x57, 0xe8, 0xfa, 0x13, 0x8f, 0xce, 0x45, 0x2e, 0x78, 0xa4, 0xc6, 0x83, 0x1f, 0x04, 0x4c, 0xd6,
	0x96, 0x48, 0x37, 0x60, 0x14, 0x27, 0x97, 0xf8, 0x24, 0x5c, 0x31, 0xa3, 0x38, 0x51, 0x0a, 0xa6,
	0xe0, 0x25, 0xba, 0x86, 0xae, 0xe8, 0x98, 0xee, 0x60, 0x29, 0x50, 0xf2, 0x56, 0xe4, 0xe8, 0xce,
	0x7d, 0x12, 0xda, 0xf1, 0x83, 0xe8, 0x42, 0x2e, 0x62, 0x7d, 0x9b, 0x8d, 0x83, 0xf4, 0x39, 0x58,
	0x59, 0x9e, 0xa3, 0x94, 0xae, 0xe9, 0x93, 0x70, 0x13, 0xdf, 0x9f, 0x1c, 0xd9, 0xeb, 0x26, 0xeb,
	0x87, 0x82, 0x14, 0x96, 0x83, 0x88, 0x62, 0xa8, 0xb2, 0x33, 0xf6, 0x54, 0x3a, 0x56, 0xb5, 0xe6,
	0x5b, 0x3d, 0x72, 0xa9, 0x98, 0x7a, 0xb0, 0xc4, 0xea, 0x54, 0xf3, 0xa2, 0x6a, 0x34, 0xd7, 0x8a,
	0x8d, 0x79, 0xf0, 0x9d, 0xc0, 0xfa, 0xa5, 0xc0, 0xac, 0x41, 0x86, 0x5f, 0x5a, 0x94, 0xcd, 0xe8,
	0x8c, 0xdc, 0x70, 0x66, 0xfc, 0xbb, 0xb3, 0xf9, 0xdf, 0x38, 0x73, 0x60, 0x33, 0x80, 0xc8, 0x9a,
	0x57, 0x12, 0x35, 0x5b, 0x82, 0x25, 0xfe, 0x17, 0x6c, 0x03, 0x48, 0xcf, 0xb6, 0x06, 0xfb, 0x4d,
	0x21, 0x9b, 0x1e, 0x2c, 0x78, 0x01, 0x77, 0xbb, 0xb4, 0x6b, 0xd3, 0xa7, 0xb0, 0xd0, 0x7f, 0x95,
	0x4b, 0xfc, 0x79, 0x68, 0xc7, 0xf7, 0xa6, 0x44, 0x6d, 0x89, 0xac, 0x9b, 0x78, 0x16, 0x81, 0xd5,
	0xbd, 0x8d, 0xda, 0x70, 0xe7, 0x7d, 0xfa, 0x3a, 0x7d, 0xfb, 0x21, 0x75, 0x66, 0x2a, 0x39, 0xb2,
	0x7d, 0xfa, 0xee, 0x90, 0x38, 0x84, 0x02, 0x58, 0xc9, 0x21, 0x7d, 0x75, 0x48, 0x1c, 0x23, 0xfe,
	0x49, 0x60, 0xa1, 0xce, 0x4b, 0x7a, 0x04, 0xab, 0xdb, 0x18, 0x7d, 0x34, 0xd1, 0xbf, 0xf8, 0xa2,
	0xde, 0xe3, 0x1b, 0xdd, 0xde, 0xca, 0x4c, 0x09, 0x75, 0xf6, 0xae, 0x84, 0x2e, 0xd6, 0x7f, 0x25,
	0x34, 0xd9, 0xc9, 0x8c, 0xee, 0xc1, 0x54, 0x6b, 0xa0, 0xde, 0x64, 0xf0, 0xb7, 0x55, 0x79, 0x0f,
	0xff, 0xd8, 0x1b, 0x24, 0x3e, 0x5a, 0xfa, 0x1e, 0xee, 0x7e, 0x05, 0x00, 0x00, 0xff, 0xff, 0x79,
	0x64, 0x82, 0x17, 0xaf, 0x03, 0x00, 0x00,
}
