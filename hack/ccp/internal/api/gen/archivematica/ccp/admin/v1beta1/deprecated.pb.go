// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: archivematica/ccp/admin/v1beta1/deprecated.proto

package adminv1beta1

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
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

type ApproveJobRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Identifier of the job (UUIDv4).
	JobId string `protobuf:"bytes,1,opt,name=job_id,json=jobId,proto3" json:"job_id,omitempty"`
	// Identifier of the choice (UUIDv4).
	Choice string `protobuf:"bytes,2,opt,name=choice,proto3" json:"choice,omitempty"`
}

func (x *ApproveJobRequest) Reset() {
	*x = ApproveJobRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApproveJobRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApproveJobRequest) ProtoMessage() {}

func (x *ApproveJobRequest) ProtoReflect() protoreflect.Message {
	mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApproveJobRequest.ProtoReflect.Descriptor instead.
func (*ApproveJobRequest) Descriptor() ([]byte, []int) {
	return file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescGZIP(), []int{0}
}

func (x *ApproveJobRequest) GetJobId() string {
	if x != nil {
		return x.JobId
	}
	return ""
}

func (x *ApproveJobRequest) GetChoice() string {
	if x != nil {
		return x.Choice
	}
	return ""
}

type ApproveJobResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ApproveJobResponse) Reset() {
	*x = ApproveJobResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApproveJobResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApproveJobResponse) ProtoMessage() {}

func (x *ApproveJobResponse) ProtoReflect() protoreflect.Message {
	mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApproveJobResponse.ProtoReflect.Descriptor instead.
func (*ApproveJobResponse) Descriptor() ([]byte, []int) {
	return file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescGZIP(), []int{1}
}

type ApproveTransferByPathRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Directory where the transfer is currently located.
	Directory string `protobuf:"bytes,1,opt,name=directory,proto3" json:"directory,omitempty"`
	// Type of the transfer, default to "standard".
	Type TransferType `protobuf:"varint,2,opt,name=type,proto3,enum=archivematica.ccp.admin.v1beta1.TransferType" json:"type,omitempty"`
}

func (x *ApproveTransferByPathRequest) Reset() {
	*x = ApproveTransferByPathRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApproveTransferByPathRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApproveTransferByPathRequest) ProtoMessage() {}

func (x *ApproveTransferByPathRequest) ProtoReflect() protoreflect.Message {
	mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApproveTransferByPathRequest.ProtoReflect.Descriptor instead.
func (*ApproveTransferByPathRequest) Descriptor() ([]byte, []int) {
	return file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescGZIP(), []int{2}
}

func (x *ApproveTransferByPathRequest) GetDirectory() string {
	if x != nil {
		return x.Directory
	}
	return ""
}

func (x *ApproveTransferByPathRequest) GetType() TransferType {
	if x != nil {
		return x.Type
	}
	return TransferType_TRANSFER_TYPE_UNSPECIFIED
}

type ApproveTransferByPathResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Identifier of the package (UUIDv4).
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *ApproveTransferByPathResponse) Reset() {
	*x = ApproveTransferByPathResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApproveTransferByPathResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApproveTransferByPathResponse) ProtoMessage() {}

func (x *ApproveTransferByPathResponse) ProtoReflect() protoreflect.Message {
	mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApproveTransferByPathResponse.ProtoReflect.Descriptor instead.
func (*ApproveTransferByPathResponse) Descriptor() ([]byte, []int) {
	return file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescGZIP(), []int{3}
}

func (x *ApproveTransferByPathResponse) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type ApprovePartialReingestRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Identifier of the package (UUIDv4).
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *ApprovePartialReingestRequest) Reset() {
	*x = ApprovePartialReingestRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApprovePartialReingestRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApprovePartialReingestRequest) ProtoMessage() {}

func (x *ApprovePartialReingestRequest) ProtoReflect() protoreflect.Message {
	mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApprovePartialReingestRequest.ProtoReflect.Descriptor instead.
func (*ApprovePartialReingestRequest) Descriptor() ([]byte, []int) {
	return file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescGZIP(), []int{4}
}

func (x *ApprovePartialReingestRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type ApprovePartialReingestResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ApprovePartialReingestResponse) Reset() {
	*x = ApprovePartialReingestResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ApprovePartialReingestResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ApprovePartialReingestResponse) ProtoMessage() {}

func (x *ApprovePartialReingestResponse) ProtoReflect() protoreflect.Message {
	mi := &file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ApprovePartialReingestResponse.ProtoReflect.Descriptor instead.
func (*ApprovePartialReingestResponse) Descriptor() ([]byte, []int) {
	return file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescGZIP(), []int{5}
}

var File_archivematica_ccp_admin_v1beta1_deprecated_proto protoreflect.FileDescriptor

var file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDesc = []byte{
	0x0a, 0x30, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69, 0x63, 0x61, 0x2f,
	0x63, 0x63, 0x70, 0x2f, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x2f, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61,
	0x31, 0x2f, 0x64, 0x65, 0x70, 0x72, 0x65, 0x63, 0x61, 0x74, 0x65, 0x64, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x1f, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69, 0x63,
	0x61, 0x2e, 0x63, 0x63, 0x70, 0x2e, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x62, 0x65,
	0x74, 0x61, 0x31, 0x1a, 0x2b, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69,
	0x63, 0x61, 0x2f, 0x63, 0x63, 0x70, 0x2f, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x2f, 0x76, 0x31, 0x62,
	0x65, 0x74, 0x61, 0x31, 0x2f, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1b, 0x62, 0x75, 0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x55, 0x0a,
	0x11, 0x41, 0x70, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x4a, 0x6f, 0x62, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x1f, 0x0a, 0x06, 0x6a, 0x6f, 0x62, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x42, 0x08, 0xba, 0x48, 0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0x52, 0x05, 0x6a, 0x6f,
	0x62, 0x49, 0x64, 0x12, 0x1f, 0x0a, 0x06, 0x63, 0x68, 0x6f, 0x69, 0x63, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x42, 0x07, 0xba, 0x48, 0x04, 0x72, 0x02, 0x10, 0x01, 0x52, 0x06, 0x63, 0x68,
	0x6f, 0x69, 0x63, 0x65, 0x22, 0x14, 0x0a, 0x12, 0x41, 0x70, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x4a,
	0x6f, 0x62, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x7f, 0x0a, 0x1c, 0x41, 0x70,
	0x70, 0x72, 0x6f, 0x76, 0x65, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x42, 0x79, 0x50,
	0x61, 0x74, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x64, 0x69,
	0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x64,
	0x69, 0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x41, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x2d, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65,
	0x6d, 0x61, 0x74, 0x69, 0x63, 0x61, 0x2e, 0x63, 0x63, 0x70, 0x2e, 0x61, 0x64, 0x6d, 0x69, 0x6e,
	0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65,
	0x72, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0x39, 0x0a, 0x1d, 0x41,
	0x70, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x42, 0x79,
	0x50, 0x61, 0x74, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x02,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xba, 0x48, 0x05, 0x72, 0x03, 0xb0,
	0x01, 0x01, 0x52, 0x02, 0x69, 0x64, 0x22, 0x39, 0x0a, 0x1d, 0x41, 0x70, 0x70, 0x72, 0x6f, 0x76,
	0x65, 0x50, 0x61, 0x72, 0x74, 0x69, 0x61, 0x6c, 0x52, 0x65, 0x69, 0x6e, 0x67, 0x65, 0x73, 0x74,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x42, 0x08, 0xba, 0x48, 0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0x52, 0x02, 0x69,
	0x64, 0x22, 0x20, 0x0a, 0x1e, 0x41, 0x70, 0x70, 0x72, 0x6f, 0x76, 0x65, 0x50, 0x61, 0x72, 0x74,
	0x69, 0x61, 0x6c, 0x52, 0x65, 0x69, 0x6e, 0x67, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x42, 0xc2, 0x02, 0x0a, 0x23, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x72, 0x63, 0x68,
	0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69, 0x63, 0x61, 0x2e, 0x63, 0x63, 0x70, 0x2e, 0x61, 0x64,
	0x6d, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x42, 0x0f, 0x44, 0x65, 0x70,
	0x72, 0x65, 0x63, 0x61, 0x74, 0x65, 0x64, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x6b,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x72, 0x74, 0x65, 0x66,
	0x61, 0x63, 0x74, 0x75, 0x61, 0x6c, 0x2f, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61,
	0x74, 0x69, 0x63, 0x61, 0x2f, 0x68, 0x61, 0x63, 0x6b, 0x2f, 0x63, 0x63, 0x70, 0x2f, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x61,
	0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69, 0x63, 0x61, 0x2f, 0x63, 0x63, 0x70,
	0x2f, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x2f, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x3b, 0x61,
	0x64, 0x6d, 0x69, 0x6e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0xa2, 0x02, 0x03, 0x41, 0x43,
	0x41, 0xaa, 0x02, 0x1f, 0x41, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69, 0x63,
	0x61, 0x2e, 0x43, 0x63, 0x70, 0x2e, 0x41, 0x64, 0x6d, 0x69, 0x6e, 0x2e, 0x56, 0x31, 0x62, 0x65,
	0x74, 0x61, 0x31, 0xca, 0x02, 0x1f, 0x41, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74,
	0x69, 0x63, 0x61, 0x5c, 0x43, 0x63, 0x70, 0x5c, 0x41, 0x64, 0x6d, 0x69, 0x6e, 0x5c, 0x56, 0x31,
	0x62, 0x65, 0x74, 0x61, 0x31, 0xe2, 0x02, 0x2b, 0x41, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d,
	0x61, 0x74, 0x69, 0x63, 0x61, 0x5c, 0x43, 0x63, 0x70, 0x5c, 0x41, 0x64, 0x6d, 0x69, 0x6e, 0x5c,
	0x56, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64,
	0x61, 0x74, 0x61, 0xea, 0x02, 0x22, 0x41, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74,
	0x69, 0x63, 0x61, 0x3a, 0x3a, 0x43, 0x63, 0x70, 0x3a, 0x3a, 0x41, 0x64, 0x6d, 0x69, 0x6e, 0x3a,
	0x3a, 0x56, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescOnce sync.Once
	file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescData = file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDesc
)

func file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescGZIP() []byte {
	file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescOnce.Do(func() {
		file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescData = protoimpl.X.CompressGZIP(file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescData)
	})
	return file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDescData
}

var file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_archivematica_ccp_admin_v1beta1_deprecated_proto_goTypes = []any{
	(*ApproveJobRequest)(nil),              // 0: archivematica.ccp.admin.v1beta1.ApproveJobRequest
	(*ApproveJobResponse)(nil),             // 1: archivematica.ccp.admin.v1beta1.ApproveJobResponse
	(*ApproveTransferByPathRequest)(nil),   // 2: archivematica.ccp.admin.v1beta1.ApproveTransferByPathRequest
	(*ApproveTransferByPathResponse)(nil),  // 3: archivematica.ccp.admin.v1beta1.ApproveTransferByPathResponse
	(*ApprovePartialReingestRequest)(nil),  // 4: archivematica.ccp.admin.v1beta1.ApprovePartialReingestRequest
	(*ApprovePartialReingestResponse)(nil), // 5: archivematica.ccp.admin.v1beta1.ApprovePartialReingestResponse
	(TransferType)(0),                      // 6: archivematica.ccp.admin.v1beta1.TransferType
}
var file_archivematica_ccp_admin_v1beta1_deprecated_proto_depIdxs = []int32{
	6, // 0: archivematica.ccp.admin.v1beta1.ApproveTransferByPathRequest.type:type_name -> archivematica.ccp.admin.v1beta1.TransferType
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_archivematica_ccp_admin_v1beta1_deprecated_proto_init() }
func file_archivematica_ccp_admin_v1beta1_deprecated_proto_init() {
	if File_archivematica_ccp_admin_v1beta1_deprecated_proto != nil {
		return
	}
	file_archivematica_ccp_admin_v1beta1_admin_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*ApproveJobRequest); i {
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
		file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*ApproveJobResponse); i {
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
		file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*ApproveTransferByPathRequest); i {
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
		file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*ApproveTransferByPathResponse); i {
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
		file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[4].Exporter = func(v any, i int) any {
			switch v := v.(*ApprovePartialReingestRequest); i {
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
		file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes[5].Exporter = func(v any, i int) any {
			switch v := v.(*ApprovePartialReingestResponse); i {
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
			RawDescriptor: file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_archivematica_ccp_admin_v1beta1_deprecated_proto_goTypes,
		DependencyIndexes: file_archivematica_ccp_admin_v1beta1_deprecated_proto_depIdxs,
		MessageInfos:      file_archivematica_ccp_admin_v1beta1_deprecated_proto_msgTypes,
	}.Build()
	File_archivematica_ccp_admin_v1beta1_deprecated_proto = out.File
	file_archivematica_ccp_admin_v1beta1_deprecated_proto_rawDesc = nil
	file_archivematica_ccp_admin_v1beta1_deprecated_proto_goTypes = nil
	file_archivematica_ccp_admin_v1beta1_deprecated_proto_depIdxs = nil
}
