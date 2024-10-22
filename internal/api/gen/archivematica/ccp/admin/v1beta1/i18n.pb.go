// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.1
// 	protoc        (unknown)
// source: archivematica/ccp/admin/v1beta1/i18n.proto

package adminv1beta1

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

// I18n represents a collection of translations for different languages. It is
// used for internationalization (i18n) purposes.
type I18N struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// A map that stores translations for different languages. Keys are language
	// codes (e.g., "en", "es", "fr"). Values are the corresponding translations
	// in those languages.
	Tx map[string]string `protobuf:"bytes,1,rep,name=tx,proto3" json:"tx,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *I18N) Reset() {
	*x = I18N{}
	mi := &file_archivematica_ccp_admin_v1beta1_i18n_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *I18N) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*I18N) ProtoMessage() {}

func (x *I18N) ProtoReflect() protoreflect.Message {
	mi := &file_archivematica_ccp_admin_v1beta1_i18n_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use I18N.ProtoReflect.Descriptor instead.
func (*I18N) Descriptor() ([]byte, []int) {
	return file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDescGZIP(), []int{0}
}

func (x *I18N) GetTx() map[string]string {
	if x != nil {
		return x.Tx
	}
	return nil
}

var File_archivematica_ccp_admin_v1beta1_i18n_proto protoreflect.FileDescriptor

var file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDesc = []byte{
	0x0a, 0x2a, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69, 0x63, 0x61, 0x2f,
	0x63, 0x63, 0x70, 0x2f, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x2f, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61,
	0x31, 0x2f, 0x69, 0x31, 0x38, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1f, 0x61, 0x72,
	0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69, 0x63, 0x61, 0x2e, 0x63, 0x63, 0x70, 0x2e,
	0x61, 0x64, 0x6d, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x22, 0x7c, 0x0a,
	0x04, 0x49, 0x31, 0x38, 0x6e, 0x12, 0x3d, 0x0a, 0x02, 0x74, 0x78, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x2d, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69, 0x63,
	0x61, 0x2e, 0x63, 0x63, 0x70, 0x2e, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x62, 0x65,
	0x74, 0x61, 0x31, 0x2e, 0x49, 0x31, 0x38, 0x6e, 0x2e, 0x54, 0x78, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x52, 0x02, 0x74, 0x78, 0x1a, 0x35, 0x0a, 0x07, 0x54, 0x78, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0xae, 0x02, 0x0a, 0x23,
	0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69, 0x63,
	0x61, 0x2e, 0x63, 0x63, 0x70, 0x2e, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x62, 0x65,
	0x74, 0x61, 0x31, 0x42, 0x09, 0x49, 0x31, 0x38, 0x6e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01,
	0x5a, 0x5d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x72, 0x74,
	0x65, 0x66, 0x61, 0x63, 0x74, 0x75, 0x61, 0x6c, 0x2d, 0x6c, 0x61, 0x62, 0x73, 0x2f, 0x63, 0x63,
	0x70, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x67,
	0x65, 0x6e, 0x2f, 0x61, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69, 0x63, 0x61,
	0x2f, 0x63, 0x63, 0x70, 0x2f, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x2f, 0x76, 0x31, 0x62, 0x65, 0x74,
	0x61, 0x31, 0x3b, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0xa2,
	0x02, 0x03, 0x41, 0x43, 0x41, 0xaa, 0x02, 0x1f, 0x41, 0x72, 0x63, 0x68, 0x69, 0x76, 0x65, 0x6d,
	0x61, 0x74, 0x69, 0x63, 0x61, 0x2e, 0x43, 0x63, 0x70, 0x2e, 0x41, 0x64, 0x6d, 0x69, 0x6e, 0x2e,
	0x56, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0xca, 0x02, 0x1f, 0x41, 0x72, 0x63, 0x68, 0x69, 0x76,
	0x65, 0x6d, 0x61, 0x74, 0x69, 0x63, 0x61, 0x5c, 0x43, 0x63, 0x70, 0x5c, 0x41, 0x64, 0x6d, 0x69,
	0x6e, 0x5c, 0x56, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0xe2, 0x02, 0x2b, 0x41, 0x72, 0x63, 0x68,
	0x69, 0x76, 0x65, 0x6d, 0x61, 0x74, 0x69, 0x63, 0x61, 0x5c, 0x43, 0x63, 0x70, 0x5c, 0x41, 0x64,
	0x6d, 0x69, 0x6e, 0x5c, 0x56, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x22, 0x41, 0x72, 0x63, 0x68, 0x69, 0x76,
	0x65, 0x6d, 0x61, 0x74, 0x69, 0x63, 0x61, 0x3a, 0x3a, 0x43, 0x63, 0x70, 0x3a, 0x3a, 0x41, 0x64,
	0x6d, 0x69, 0x6e, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDescOnce sync.Once
	file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDescData = file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDesc
)

func file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDescGZIP() []byte {
	file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDescOnce.Do(func() {
		file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDescData = protoimpl.X.CompressGZIP(file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDescData)
	})
	return file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDescData
}

var file_archivematica_ccp_admin_v1beta1_i18n_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_archivematica_ccp_admin_v1beta1_i18n_proto_goTypes = []any{
	(*I18N)(nil), // 0: archivematica.ccp.admin.v1beta1.I18n
	nil,          // 1: archivematica.ccp.admin.v1beta1.I18n.TxEntry
}
var file_archivematica_ccp_admin_v1beta1_i18n_proto_depIdxs = []int32{
	1, // 0: archivematica.ccp.admin.v1beta1.I18n.tx:type_name -> archivematica.ccp.admin.v1beta1.I18n.TxEntry
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_archivematica_ccp_admin_v1beta1_i18n_proto_init() }
func file_archivematica_ccp_admin_v1beta1_i18n_proto_init() {
	if File_archivematica_ccp_admin_v1beta1_i18n_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_archivematica_ccp_admin_v1beta1_i18n_proto_goTypes,
		DependencyIndexes: file_archivematica_ccp_admin_v1beta1_i18n_proto_depIdxs,
		MessageInfos:      file_archivematica_ccp_admin_v1beta1_i18n_proto_msgTypes,
	}.Build()
	File_archivematica_ccp_admin_v1beta1_i18n_proto = out.File
	file_archivematica_ccp_admin_v1beta1_i18n_proto_rawDesc = nil
	file_archivematica_ccp_admin_v1beta1_i18n_proto_goTypes = nil
	file_archivematica_ccp_admin_v1beta1_i18n_proto_depIdxs = nil
}
