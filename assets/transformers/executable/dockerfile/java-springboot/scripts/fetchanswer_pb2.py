# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: fetchanswer.proto
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor.FileDescriptor(
  name='fetchanswer.proto',
  package='qagrpc',
  syntax='proto3',
  serialized_options=b'Z3github.com/konveyor/move2kube/types/qaengine/qagrpc',
  create_key=_descriptor._internal_create_key,
  serialized_pb=b'\n\x11\x66\x65tchanswer.proto\x12\x06qagrpc\"i\n\x07Problem\x12\n\n\x02id\x18\x01 \x01(\t\x12\x0c\n\x04type\x18\x02 \x01(\t\x12\x13\n\x0b\x64\x65scription\x18\x03 \x01(\t\x12\r\n\x05hints\x18\x04 \x03(\t\x12\x0f\n\x07options\x18\x05 \x03(\t\x12\x0f\n\x07\x64\x65\x66\x61ult\x18\x06 \x03(\t\"\x18\n\x06\x41nswer\x12\x0e\n\x06\x61nswer\x18\x01 \x03(\t2<\n\x08QAEngine\x12\x30\n\x0b\x46\x65tchAnswer\x12\x0f.qagrpc.Problem\x1a\x0e.qagrpc.Answer\"\x00\x42\x35Z3github.com/konveyor/move2kube/types/qaengine/qagrpcb\x06proto3'
)




_PROBLEM = _descriptor.Descriptor(
  name='Problem',
  full_name='qagrpc.Problem',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  create_key=_descriptor._internal_create_key,
  fields=[
    _descriptor.FieldDescriptor(
      name='id', full_name='qagrpc.Problem.id', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
    _descriptor.FieldDescriptor(
      name='type', full_name='qagrpc.Problem.type', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
    _descriptor.FieldDescriptor(
      name='description', full_name='qagrpc.Problem.description', index=2,
      number=3, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=b"".decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
    _descriptor.FieldDescriptor(
      name='hints', full_name='qagrpc.Problem.hints', index=3,
      number=4, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
    _descriptor.FieldDescriptor(
      name='options', full_name='qagrpc.Problem.options', index=4,
      number=5, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
    _descriptor.FieldDescriptor(
      name='default', full_name='qagrpc.Problem.default', index=5,
      number=6, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=29,
  serialized_end=134,
)


_ANSWER = _descriptor.Descriptor(
  name='Answer',
  full_name='qagrpc.Answer',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  create_key=_descriptor._internal_create_key,
  fields=[
    _descriptor.FieldDescriptor(
      name='answer', full_name='qagrpc.Answer.answer', index=0,
      number=1, type=9, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR,  create_key=_descriptor._internal_create_key),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=136,
  serialized_end=160,
)

DESCRIPTOR.message_types_by_name['Problem'] = _PROBLEM
DESCRIPTOR.message_types_by_name['Answer'] = _ANSWER
_sym_db.RegisterFileDescriptor(DESCRIPTOR)

Problem = _reflection.GeneratedProtocolMessageType('Problem', (_message.Message,), {
  'DESCRIPTOR' : _PROBLEM,
  '__module__' : 'fetchanswer_pb2'
  # @@protoc_insertion_point(class_scope:qagrpc.Problem)
  })
_sym_db.RegisterMessage(Problem)

Answer = _reflection.GeneratedProtocolMessageType('Answer', (_message.Message,), {
  'DESCRIPTOR' : _ANSWER,
  '__module__' : 'fetchanswer_pb2'
  # @@protoc_insertion_point(class_scope:qagrpc.Answer)
  })
_sym_db.RegisterMessage(Answer)


DESCRIPTOR._options = None

_QAENGINE = _descriptor.ServiceDescriptor(
  name='QAEngine',
  full_name='qagrpc.QAEngine',
  file=DESCRIPTOR,
  index=0,
  serialized_options=None,
  create_key=_descriptor._internal_create_key,
  serialized_start=162,
  serialized_end=222,
  methods=[
  _descriptor.MethodDescriptor(
    name='FetchAnswer',
    full_name='qagrpc.QAEngine.FetchAnswer',
    index=0,
    containing_service=None,
    input_type=_PROBLEM,
    output_type=_ANSWER,
    serialized_options=None,
    create_key=_descriptor._internal_create_key,
  ),
])
_sym_db.RegisterServiceDescriptor(_QAENGINE)

DESCRIPTOR.services_by_name['QAEngine'] = _QAENGINE

# @@protoc_insertion_point(module_scope)
