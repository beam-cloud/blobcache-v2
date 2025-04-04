// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.25.1
// source: blobcache.proto

package proto

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

type GetContentRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Hash   string `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`      // Hash of the content to retrieve
	Offset int64  `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"` // Offset into the content
	Length int64  `protobuf:"varint,3,opt,name=length,proto3" json:"length,omitempty"` // Length of the content to retrieve
}

func (x *GetContentRequest) Reset() {
	*x = GetContentRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_blobcache_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetContentRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetContentRequest) ProtoMessage() {}

func (x *GetContentRequest) ProtoReflect() protoreflect.Message {
	mi := &file_blobcache_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetContentRequest.ProtoReflect.Descriptor instead.
func (*GetContentRequest) Descriptor() ([]byte, []int) {
	return file_blobcache_proto_rawDescGZIP(), []int{0}
}

func (x *GetContentRequest) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

func (x *GetContentRequest) GetOffset() int64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

func (x *GetContentRequest) GetLength() int64 {
	if x != nil {
		return x.Length
	}
	return 0
}

type GetContentResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ok      bool   `protobuf:"varint,1,opt,name=ok,proto3" json:"ok,omitempty"`
	Content []byte `protobuf:"bytes,2,opt,name=content,proto3" json:"content,omitempty"` // Content data
}

func (x *GetContentResponse) Reset() {
	*x = GetContentResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_blobcache_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetContentResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetContentResponse) ProtoMessage() {}

func (x *GetContentResponse) ProtoReflect() protoreflect.Message {
	mi := &file_blobcache_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetContentResponse.ProtoReflect.Descriptor instead.
func (*GetContentResponse) Descriptor() ([]byte, []int) {
	return file_blobcache_proto_rawDescGZIP(), []int{1}
}

func (x *GetContentResponse) GetOk() bool {
	if x != nil {
		return x.Ok
	}
	return false
}

func (x *GetContentResponse) GetContent() []byte {
	if x != nil {
		return x.Content
	}
	return nil
}

type StoreContentRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Content []byte `protobuf:"bytes,1,opt,name=content,proto3" json:"content,omitempty"`
}

func (x *StoreContentRequest) Reset() {
	*x = StoreContentRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_blobcache_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StoreContentRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StoreContentRequest) ProtoMessage() {}

func (x *StoreContentRequest) ProtoReflect() protoreflect.Message {
	mi := &file_blobcache_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StoreContentRequest.ProtoReflect.Descriptor instead.
func (*StoreContentRequest) Descriptor() ([]byte, []int) {
	return file_blobcache_proto_rawDescGZIP(), []int{2}
}

func (x *StoreContentRequest) GetContent() []byte {
	if x != nil {
		return x.Content
	}
	return nil
}

type StoreContentResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Hash string `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"` // Hash of the stored content
}

func (x *StoreContentResponse) Reset() {
	*x = StoreContentResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_blobcache_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StoreContentResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StoreContentResponse) ProtoMessage() {}

func (x *StoreContentResponse) ProtoReflect() protoreflect.Message {
	mi := &file_blobcache_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StoreContentResponse.ProtoReflect.Descriptor instead.
func (*StoreContentResponse) Descriptor() ([]byte, []int) {
	return file_blobcache_proto_rawDescGZIP(), []int{3}
}

func (x *StoreContentResponse) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

type GetStateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetStateRequest) Reset() {
	*x = GetStateRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_blobcache_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetStateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetStateRequest) ProtoMessage() {}

func (x *GetStateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_blobcache_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetStateRequest.ProtoReflect.Descriptor instead.
func (*GetStateRequest) Descriptor() ([]byte, []int) {
	return file_blobcache_proto_rawDescGZIP(), []int{4}
}

type GetStateResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Version          string  `protobuf:"bytes,1,opt,name=version,proto3" json:"version,omitempty"`
	CapacityUsagePct float32 `protobuf:"fixed32,2,opt,name=capacity_usage_pct,json=capacityUsagePct,proto3" json:"capacity_usage_pct,omitempty"`
	PrivateIpAddr    string  `protobuf:"bytes,3,opt,name=private_ip_addr,json=privateIpAddr,proto3" json:"private_ip_addr,omitempty"`
}

func (x *GetStateResponse) Reset() {
	*x = GetStateResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_blobcache_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetStateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetStateResponse) ProtoMessage() {}

func (x *GetStateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_blobcache_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetStateResponse.ProtoReflect.Descriptor instead.
func (*GetStateResponse) Descriptor() ([]byte, []int) {
	return file_blobcache_proto_rawDescGZIP(), []int{5}
}

func (x *GetStateResponse) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *GetStateResponse) GetCapacityUsagePct() float32 {
	if x != nil {
		return x.CapacityUsagePct
	}
	return 0
}

func (x *GetStateResponse) GetPrivateIpAddr() string {
	if x != nil {
		return x.PrivateIpAddr
	}
	return ""
}

type StoreContentFromSourceRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SourcePath   string `protobuf:"bytes,1,opt,name=source_path,json=sourcePath,proto3" json:"source_path,omitempty"`
	SourceOffset int64  `protobuf:"varint,2,opt,name=source_offset,json=sourceOffset,proto3" json:"source_offset,omitempty"`
}

func (x *StoreContentFromSourceRequest) Reset() {
	*x = StoreContentFromSourceRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_blobcache_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StoreContentFromSourceRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StoreContentFromSourceRequest) ProtoMessage() {}

func (x *StoreContentFromSourceRequest) ProtoReflect() protoreflect.Message {
	mi := &file_blobcache_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StoreContentFromSourceRequest.ProtoReflect.Descriptor instead.
func (*StoreContentFromSourceRequest) Descriptor() ([]byte, []int) {
	return file_blobcache_proto_rawDescGZIP(), []int{6}
}

func (x *StoreContentFromSourceRequest) GetSourcePath() string {
	if x != nil {
		return x.SourcePath
	}
	return ""
}

func (x *StoreContentFromSourceRequest) GetSourceOffset() int64 {
	if x != nil {
		return x.SourceOffset
	}
	return 0
}

type StoreContentFromSourceResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ok   bool   `protobuf:"varint,1,opt,name=ok,proto3" json:"ok,omitempty"`
	Hash string `protobuf:"bytes,2,opt,name=hash,proto3" json:"hash,omitempty"`
}

func (x *StoreContentFromSourceResponse) Reset() {
	*x = StoreContentFromSourceResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_blobcache_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StoreContentFromSourceResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StoreContentFromSourceResponse) ProtoMessage() {}

func (x *StoreContentFromSourceResponse) ProtoReflect() protoreflect.Message {
	mi := &file_blobcache_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StoreContentFromSourceResponse.ProtoReflect.Descriptor instead.
func (*StoreContentFromSourceResponse) Descriptor() ([]byte, []int) {
	return file_blobcache_proto_rawDescGZIP(), []int{7}
}

func (x *StoreContentFromSourceResponse) GetOk() bool {
	if x != nil {
		return x.Ok
	}
	return false
}

func (x *StoreContentFromSourceResponse) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

var File_blobcache_proto protoreflect.FileDescriptor

var file_blobcache_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x62, 0x6c, 0x6f, 0x62, 0x63, 0x61, 0x63, 0x68, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x09, 0x62, 0x6c, 0x6f, 0x62, 0x63, 0x61, 0x63, 0x68, 0x65, 0x22, 0x57, 0x0a, 0x11,
	0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x68, 0x61, 0x73, 0x68, 0x12, 0x16, 0x0a, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x12, 0x16, 0x0a,
	0x06, 0x6c, 0x65, 0x6e, 0x67, 0x74, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x6c,
	0x65, 0x6e, 0x67, 0x74, 0x68, 0x22, 0x3e, 0x0a, 0x12, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x74,
	0x65, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x6f,
	0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x02, 0x6f, 0x6b, 0x12, 0x18, 0x0a, 0x07, 0x63,
	0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x63, 0x6f,
	0x6e, 0x74, 0x65, 0x6e, 0x74, 0x22, 0x2f, 0x0a, 0x13, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x43, 0x6f,
	0x6e, 0x74, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x07,
	0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x63,
	0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x22, 0x2a, 0x0a, 0x14, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x43,
	0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x68, 0x61, 0x73, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x61,
	0x73, 0x68, 0x22, 0x11, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x82, 0x01, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61,
	0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65,
	0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72,
	0x73, 0x69, 0x6f, 0x6e, 0x12, 0x2c, 0x0a, 0x12, 0x63, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79,
	0x5f, 0x75, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x70, 0x63, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x02,
	0x52, 0x10, 0x63, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x55, 0x73, 0x61, 0x67, 0x65, 0x50,
	0x63, 0x74, 0x12, 0x26, 0x0a, 0x0f, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x5f, 0x69, 0x70,
	0x5f, 0x61, 0x64, 0x64, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x70, 0x72, 0x69,
	0x76, 0x61, 0x74, 0x65, 0x49, 0x70, 0x41, 0x64, 0x64, 0x72, 0x22, 0x65, 0x0a, 0x1d, 0x53, 0x74,
	0x6f, 0x72, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x46, 0x72, 0x6f, 0x6d, 0x53, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0a, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x50, 0x61, 0x74, 0x68, 0x12, 0x23, 0x0a, 0x0d,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x0c, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x4f, 0x66, 0x66, 0x73, 0x65,
	0x74, 0x22, 0x44, 0x0a, 0x1e, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e,
	0x74, 0x46, 0x72, 0x6f, 0x6d, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x6f, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x02, 0x6f, 0x6b, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x32, 0xba, 0x03, 0x0a, 0x09, 0x42, 0x6c, 0x6f, 0x62,
	0x43, 0x61, 0x63, 0x68, 0x65, 0x12, 0x4b, 0x0a, 0x0a, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x74,
	0x65, 0x6e, 0x74, 0x12, 0x1c, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x63, 0x61, 0x63, 0x68, 0x65, 0x2e,
	0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x1d, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x63, 0x61, 0x63, 0x68, 0x65, 0x2e, 0x47, 0x65,
	0x74, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x22, 0x00, 0x12, 0x53, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74,
	0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x1c, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x63, 0x61, 0x63,
	0x68, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x1d, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x63, 0x61, 0x63, 0x68, 0x65,
	0x2e, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x30, 0x01, 0x12, 0x53, 0x0a, 0x0c, 0x53, 0x74, 0x6f, 0x72, 0x65,
	0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x12, 0x1e, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x63, 0x61,
	0x63, 0x68, 0x65, 0x2e, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1f, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x63, 0x61,
	0x63, 0x68, 0x65, 0x2e, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x28, 0x01, 0x12, 0x6f, 0x0a, 0x16,
	0x53, 0x74, 0x6f, 0x72, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x46, 0x72, 0x6f, 0x6d,
	0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x28, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x63, 0x61, 0x63,
	0x68, 0x65, 0x2e, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x46,
	0x72, 0x6f, 0x6d, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x29, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x63, 0x61, 0x63, 0x68, 0x65, 0x2e, 0x53, 0x74, 0x6f,
	0x72, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x46, 0x72, 0x6f, 0x6d, 0x53, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x45, 0x0a,
	0x08, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x1a, 0x2e, 0x62, 0x6c, 0x6f, 0x62,
	0x63, 0x61, 0x63, 0x68, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1b, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x63, 0x61, 0x63, 0x68,
	0x65, 0x2e, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x00, 0x42, 0x27, 0x5a, 0x25, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x62, 0x65, 0x61, 0x6d, 0x2d, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x62, 0x6c,
	0x6f, 0x62, 0x63, 0x61, 0x63, 0x68, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_blobcache_proto_rawDescOnce sync.Once
	file_blobcache_proto_rawDescData = file_blobcache_proto_rawDesc
)

func file_blobcache_proto_rawDescGZIP() []byte {
	file_blobcache_proto_rawDescOnce.Do(func() {
		file_blobcache_proto_rawDescData = protoimpl.X.CompressGZIP(file_blobcache_proto_rawDescData)
	})
	return file_blobcache_proto_rawDescData
}

var file_blobcache_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_blobcache_proto_goTypes = []interface{}{
	(*GetContentRequest)(nil),              // 0: blobcache.GetContentRequest
	(*GetContentResponse)(nil),             // 1: blobcache.GetContentResponse
	(*StoreContentRequest)(nil),            // 2: blobcache.StoreContentRequest
	(*StoreContentResponse)(nil),           // 3: blobcache.StoreContentResponse
	(*GetStateRequest)(nil),                // 4: blobcache.GetStateRequest
	(*GetStateResponse)(nil),               // 5: blobcache.GetStateResponse
	(*StoreContentFromSourceRequest)(nil),  // 6: blobcache.StoreContentFromSourceRequest
	(*StoreContentFromSourceResponse)(nil), // 7: blobcache.StoreContentFromSourceResponse
}
var file_blobcache_proto_depIdxs = []int32{
	0, // 0: blobcache.BlobCache.GetContent:input_type -> blobcache.GetContentRequest
	0, // 1: blobcache.BlobCache.GetContentStream:input_type -> blobcache.GetContentRequest
	2, // 2: blobcache.BlobCache.StoreContent:input_type -> blobcache.StoreContentRequest
	6, // 3: blobcache.BlobCache.StoreContentFromSource:input_type -> blobcache.StoreContentFromSourceRequest
	4, // 4: blobcache.BlobCache.GetState:input_type -> blobcache.GetStateRequest
	1, // 5: blobcache.BlobCache.GetContent:output_type -> blobcache.GetContentResponse
	1, // 6: blobcache.BlobCache.GetContentStream:output_type -> blobcache.GetContentResponse
	3, // 7: blobcache.BlobCache.StoreContent:output_type -> blobcache.StoreContentResponse
	7, // 8: blobcache.BlobCache.StoreContentFromSource:output_type -> blobcache.StoreContentFromSourceResponse
	5, // 9: blobcache.BlobCache.GetState:output_type -> blobcache.GetStateResponse
	5, // [5:10] is the sub-list for method output_type
	0, // [0:5] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_blobcache_proto_init() }
func file_blobcache_proto_init() {
	if File_blobcache_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_blobcache_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetContentRequest); i {
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
		file_blobcache_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetContentResponse); i {
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
		file_blobcache_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StoreContentRequest); i {
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
		file_blobcache_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StoreContentResponse); i {
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
		file_blobcache_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetStateRequest); i {
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
		file_blobcache_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetStateResponse); i {
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
		file_blobcache_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StoreContentFromSourceRequest); i {
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
		file_blobcache_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StoreContentFromSourceResponse); i {
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
			RawDescriptor: file_blobcache_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_blobcache_proto_goTypes,
		DependencyIndexes: file_blobcache_proto_depIdxs,
		MessageInfos:      file_blobcache_proto_msgTypes,
	}.Build()
	File_blobcache_proto = out.File
	file_blobcache_proto_rawDesc = nil
	file_blobcache_proto_goTypes = nil
	file_blobcache_proto_depIdxs = nil
}
