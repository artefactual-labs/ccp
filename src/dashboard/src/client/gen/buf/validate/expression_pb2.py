# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# NO CHECKED-IN PROTOBUF GENCODE
# source: buf/validate/expression.proto
# Protobuf Python Version: 5.27.1
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import runtime_version as _runtime_version
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
_runtime_version.ValidateProtobufRuntimeVersion(
    _runtime_version.Domain.PUBLIC,
    5,
    27,
    1,
    '',
    'buf/validate/expression.proto'
)
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x1d\x62uf/validate/expression.proto\x12\x0c\x62uf.validate\"V\n\nConstraint\x12\x0e\n\x02id\x18\x01 \x01(\tR\x02id\x12\x18\n\x07message\x18\x02 \x01(\tR\x07message\x12\x1e\n\nexpression\x18\x03 \x01(\tR\nexpression\"E\n\nViolations\x12\x37\n\nviolations\x18\x01 \x03(\x0b\x32\x17.buf.validate.ViolationR\nviolations\"\x82\x01\n\tViolation\x12\x1d\n\nfield_path\x18\x01 \x01(\tR\tfieldPath\x12#\n\rconstraint_id\x18\x02 \x01(\tR\x0c\x63onstraintId\x12\x18\n\x07message\x18\x03 \x01(\tR\x07message\x12\x17\n\x07\x66or_key\x18\x04 \x01(\x08R\x06\x66orKeyB\xbd\x01\n\x10\x63om.buf.validateB\x0f\x45xpressionProtoP\x01ZGbuf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate\xa2\x02\x03\x42VX\xaa\x02\x0c\x42uf.Validate\xca\x02\x0c\x42uf\\Validate\xe2\x02\x18\x42uf\\Validate\\GPBMetadata\xea\x02\rBuf::Validateb\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'buf.validate.expression_pb2', _globals)
if not _descriptor._USE_C_DESCRIPTORS:
  _globals['DESCRIPTOR']._loaded_options = None
  _globals['DESCRIPTOR']._serialized_options = b'\n\020com.buf.validateB\017ExpressionProtoP\001ZGbuf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate\242\002\003BVX\252\002\014Buf.Validate\312\002\014Buf\\Validate\342\002\030Buf\\Validate\\GPBMetadata\352\002\rBuf::Validate'
  _globals['_CONSTRAINT']._serialized_start=47
  _globals['_CONSTRAINT']._serialized_end=133
  _globals['_VIOLATIONS']._serialized_start=135
  _globals['_VIOLATIONS']._serialized_end=204
  _globals['_VIOLATION']._serialized_start=207
  _globals['_VIOLATION']._serialized_end=337
# @@protoc_insertion_point(module_scope)
