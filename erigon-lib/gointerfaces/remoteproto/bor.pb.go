// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v5.27.1
// source: remote/bor.proto

package remoteproto

import (
	typesproto "github.com/erigontech/erigon-lib/gointerfaces/typesproto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type BorTxnLookupRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BorTxHash *typesproto.H256 `protobuf:"bytes,1,opt,name=bor_tx_hash,json=borTxHash,proto3" json:"bor_tx_hash,omitempty"`
}

func (x *BorTxnLookupRequest) Reset() {
	*x = BorTxnLookupRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_remote_bor_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BorTxnLookupRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BorTxnLookupRequest) ProtoMessage() {}

func (x *BorTxnLookupRequest) ProtoReflect() protoreflect.Message {
	mi := &file_remote_bor_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BorTxnLookupRequest.ProtoReflect.Descriptor instead.
func (*BorTxnLookupRequest) Descriptor() ([]byte, []int) {
	return file_remote_bor_proto_rawDescGZIP(), []int{0}
}

func (x *BorTxnLookupRequest) GetBorTxHash() *typesproto.H256 {
	if x != nil {
		return x.BorTxHash
	}
	return nil
}

type BorTxnLookupReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Present     bool   `protobuf:"varint,1,opt,name=present,proto3" json:"present,omitempty"`
	BlockNumber uint64 `protobuf:"varint,2,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
}

func (x *BorTxnLookupReply) Reset() {
	*x = BorTxnLookupReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_remote_bor_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BorTxnLookupReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BorTxnLookupReply) ProtoMessage() {}

func (x *BorTxnLookupReply) ProtoReflect() protoreflect.Message {
	mi := &file_remote_bor_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BorTxnLookupReply.ProtoReflect.Descriptor instead.
func (*BorTxnLookupReply) Descriptor() ([]byte, []int) {
	return file_remote_bor_proto_rawDescGZIP(), []int{1}
}

func (x *BorTxnLookupReply) GetPresent() bool {
	if x != nil {
		return x.Present
	}
	return false
}

func (x *BorTxnLookupReply) GetBlockNumber() uint64 {
	if x != nil {
		return x.BlockNumber
	}
	return 0
}

type BorEventsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BlockNum  uint64           `protobuf:"varint,1,opt,name=block_num,json=blockNum,proto3" json:"block_num,omitempty"`
	BlockHash *typesproto.H256 `protobuf:"bytes,2,opt,name=block_hash,json=blockHash,proto3" json:"block_hash,omitempty"`
}

func (x *BorEventsRequest) Reset() {
	*x = BorEventsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_remote_bor_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BorEventsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BorEventsRequest) ProtoMessage() {}

func (x *BorEventsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_remote_bor_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BorEventsRequest.ProtoReflect.Descriptor instead.
func (*BorEventsRequest) Descriptor() ([]byte, []int) {
	return file_remote_bor_proto_rawDescGZIP(), []int{2}
}

func (x *BorEventsRequest) GetBlockNum() uint64 {
	if x != nil {
		return x.BlockNum
	}
	return 0
}

func (x *BorEventsRequest) GetBlockHash() *typesproto.H256 {
	if x != nil {
		return x.BlockHash
	}
	return nil
}

type BorEventsReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StateReceiverContractAddress string   `protobuf:"bytes,1,opt,name=state_receiver_contract_address,json=stateReceiverContractAddress,proto3" json:"state_receiver_contract_address,omitempty"`
	EventRlps                    [][]byte `protobuf:"bytes,2,rep,name=event_rlps,json=eventRlps,proto3" json:"event_rlps,omitempty"`
}

func (x *BorEventsReply) Reset() {
	*x = BorEventsReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_remote_bor_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BorEventsReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BorEventsReply) ProtoMessage() {}

func (x *BorEventsReply) ProtoReflect() protoreflect.Message {
	mi := &file_remote_bor_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BorEventsReply.ProtoReflect.Descriptor instead.
func (*BorEventsReply) Descriptor() ([]byte, []int) {
	return file_remote_bor_proto_rawDescGZIP(), []int{3}
}

func (x *BorEventsReply) GetStateReceiverContractAddress() string {
	if x != nil {
		return x.StateReceiverContractAddress
	}
	return ""
}

func (x *BorEventsReply) GetEventRlps() [][]byte {
	if x != nil {
		return x.EventRlps
	}
	return nil
}

var File_remote_bor_proto protoreflect.FileDescriptor

var file_remote_bor_proto_rawDesc = []byte{
	0x0a, 0x10, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x2f, 0x62, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x06, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74,
	0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x11, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2f, 0x74,
	0x79, 0x70, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x42, 0x0a, 0x13, 0x42, 0x6f,
	0x72, 0x54, 0x78, 0x6e, 0x4c, 0x6f, 0x6f, 0x6b, 0x75, 0x70, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x2b, 0x0a, 0x0b, 0x62, 0x6f, 0x72, 0x5f, 0x74, 0x78, 0x5f, 0x68, 0x61, 0x73, 0x68,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x48,
	0x32, 0x35, 0x36, 0x52, 0x09, 0x62, 0x6f, 0x72, 0x54, 0x78, 0x48, 0x61, 0x73, 0x68, 0x22, 0x50,
	0x0a, 0x11, 0x42, 0x6f, 0x72, 0x54, 0x78, 0x6e, 0x4c, 0x6f, 0x6f, 0x6b, 0x75, 0x70, 0x52, 0x65,
	0x70, 0x6c, 0x79, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x72, 0x65, 0x73, 0x65, 0x6e, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x70, 0x72, 0x65, 0x73, 0x65, 0x6e, 0x74, 0x12, 0x21, 0x0a,
	0x0c, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72,
	0x22, 0x5b, 0x0a, 0x10, 0x42, 0x6f, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x6e, 0x75,
	0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x08, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x4e, 0x75,
	0x6d, 0x12, 0x2a, 0x0a, 0x0a, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x48, 0x32,
	0x35, 0x36, 0x52, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x22, 0x76, 0x0a,
	0x0e, 0x42, 0x6f, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12,
	0x45, 0x0a, 0x1f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x5f, 0x72, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65,
	0x72, 0x5f, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65,
	0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x1c, 0x73, 0x74, 0x61, 0x74, 0x65, 0x52,
	0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x41,
	0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x1d, 0x0a, 0x0a, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f,
	0x72, 0x6c, 0x70, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x09, 0x65, 0x76, 0x65, 0x6e,
	0x74, 0x52, 0x6c, 0x70, 0x73, 0x32, 0xce, 0x01, 0x0a, 0x0d, 0x42, 0x72, 0x69, 0x64, 0x67, 0x65,
	0x42, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x12, 0x36, 0x0a, 0x07, 0x56, 0x65, 0x72, 0x73, 0x69,
	0x6f, 0x6e, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x13, 0x2e, 0x74, 0x79, 0x70,
	0x65, 0x73, 0x2e, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12,
	0x46, 0x0a, 0x0c, 0x42, 0x6f, 0x72, 0x54, 0x78, 0x6e, 0x4c, 0x6f, 0x6f, 0x6b, 0x75, 0x70, 0x12,
	0x1b, 0x2e, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x2e, 0x42, 0x6f, 0x72, 0x54, 0x78, 0x6e, 0x4c,
	0x6f, 0x6f, 0x6b, 0x75, 0x70, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x72,
	0x65, 0x6d, 0x6f, 0x74, 0x65, 0x2e, 0x42, 0x6f, 0x72, 0x54, 0x78, 0x6e, 0x4c, 0x6f, 0x6f, 0x6b,
	0x75, 0x70, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x3d, 0x0a, 0x09, 0x42, 0x6f, 0x72, 0x45, 0x76,
	0x65, 0x6e, 0x74, 0x73, 0x12, 0x18, 0x2e, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x2e, 0x42, 0x6f,
	0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16,
	0x2e, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x2e, 0x42, 0x6f, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74,
	0x73, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x42, 0x16, 0x5a, 0x14, 0x2e, 0x2f, 0x72, 0x65, 0x6d, 0x6f,
	0x74, 0x65, 0x3b, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_remote_bor_proto_rawDescOnce sync.Once
	file_remote_bor_proto_rawDescData = file_remote_bor_proto_rawDesc
)

func file_remote_bor_proto_rawDescGZIP() []byte {
	file_remote_bor_proto_rawDescOnce.Do(func() {
		file_remote_bor_proto_rawDescData = protoimpl.X.CompressGZIP(file_remote_bor_proto_rawDescData)
	})
	return file_remote_bor_proto_rawDescData
}

var file_remote_bor_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_remote_bor_proto_goTypes = []any{
	(*BorTxnLookupRequest)(nil),     // 0: remote.BorTxnLookupRequest
	(*BorTxnLookupReply)(nil),       // 1: remote.BorTxnLookupReply
	(*BorEventsRequest)(nil),        // 2: remote.BorEventsRequest
	(*BorEventsReply)(nil),          // 3: remote.BorEventsReply
	(*typesproto.H256)(nil),         // 4: types.H256
	(*emptypb.Empty)(nil),           // 5: google.protobuf.Empty
	(*typesproto.VersionReply)(nil), // 6: types.VersionReply
}
var file_remote_bor_proto_depIdxs = []int32{
	4, // 0: remote.BorTxnLookupRequest.bor_tx_hash:type_name -> types.H256
	4, // 1: remote.BorEventsRequest.block_hash:type_name -> types.H256
	5, // 2: remote.BridgeBackend.Version:input_type -> google.protobuf.Empty
	0, // 3: remote.BridgeBackend.BorTxnLookup:input_type -> remote.BorTxnLookupRequest
	2, // 4: remote.BridgeBackend.BorEvents:input_type -> remote.BorEventsRequest
	6, // 5: remote.BridgeBackend.Version:output_type -> types.VersionReply
	1, // 6: remote.BridgeBackend.BorTxnLookup:output_type -> remote.BorTxnLookupReply
	3, // 7: remote.BridgeBackend.BorEvents:output_type -> remote.BorEventsReply
	5, // [5:8] is the sub-list for method output_type
	2, // [2:5] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_remote_bor_proto_init() }
func file_remote_bor_proto_init() {
	if File_remote_bor_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_remote_bor_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*BorTxnLookupRequest); i {
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
		file_remote_bor_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*BorTxnLookupReply); i {
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
		file_remote_bor_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*BorEventsRequest); i {
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
		file_remote_bor_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*BorEventsReply); i {
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
			RawDescriptor: file_remote_bor_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_remote_bor_proto_goTypes,
		DependencyIndexes: file_remote_bor_proto_depIdxs,
		MessageInfos:      file_remote_bor_proto_msgTypes,
	}.Build()
	File_remote_bor_proto = out.File
	file_remote_bor_proto_rawDesc = nil
	file_remote_bor_proto_goTypes = nil
	file_remote_bor_proto_depIdxs = nil
}
