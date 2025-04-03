// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: core/contract/market_contract.proto

package core

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type MarketSellAssetContract struct {
	state             protoimpl.MessageState `protogen:"open.v1"`
	OwnerAddress      []byte                 `protobuf:"bytes,1,opt,name=owner_address,json=ownerAddress,proto3" json:"owner_address,omitempty"`
	SellTokenId       []byte                 `protobuf:"bytes,2,opt,name=sell_token_id,json=sellTokenId,proto3" json:"sell_token_id,omitempty"`
	SellTokenQuantity int64                  `protobuf:"varint,3,opt,name=sell_token_quantity,json=sellTokenQuantity,proto3" json:"sell_token_quantity,omitempty"`
	BuyTokenId        []byte                 `protobuf:"bytes,4,opt,name=buy_token_id,json=buyTokenId,proto3" json:"buy_token_id,omitempty"`
	BuyTokenQuantity  int64                  `protobuf:"varint,5,opt,name=buy_token_quantity,json=buyTokenQuantity,proto3" json:"buy_token_quantity,omitempty"` // min to receive
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *MarketSellAssetContract) Reset() {
	*x = MarketSellAssetContract{}
	mi := &file_core_contract_market_contract_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *MarketSellAssetContract) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MarketSellAssetContract) ProtoMessage() {}

func (x *MarketSellAssetContract) ProtoReflect() protoreflect.Message {
	mi := &file_core_contract_market_contract_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MarketSellAssetContract.ProtoReflect.Descriptor instead.
func (*MarketSellAssetContract) Descriptor() ([]byte, []int) {
	return file_core_contract_market_contract_proto_rawDescGZIP(), []int{0}
}

func (x *MarketSellAssetContract) GetOwnerAddress() []byte {
	if x != nil {
		return x.OwnerAddress
	}
	return nil
}

func (x *MarketSellAssetContract) GetSellTokenId() []byte {
	if x != nil {
		return x.SellTokenId
	}
	return nil
}

func (x *MarketSellAssetContract) GetSellTokenQuantity() int64 {
	if x != nil {
		return x.SellTokenQuantity
	}
	return 0
}

func (x *MarketSellAssetContract) GetBuyTokenId() []byte {
	if x != nil {
		return x.BuyTokenId
	}
	return nil
}

func (x *MarketSellAssetContract) GetBuyTokenQuantity() int64 {
	if x != nil {
		return x.BuyTokenQuantity
	}
	return 0
}

type MarketCancelOrderContract struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	OwnerAddress  []byte                 `protobuf:"bytes,1,opt,name=owner_address,json=ownerAddress,proto3" json:"owner_address,omitempty"`
	OrderId       []byte                 `protobuf:"bytes,2,opt,name=order_id,json=orderId,proto3" json:"order_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *MarketCancelOrderContract) Reset() {
	*x = MarketCancelOrderContract{}
	mi := &file_core_contract_market_contract_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *MarketCancelOrderContract) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MarketCancelOrderContract) ProtoMessage() {}

func (x *MarketCancelOrderContract) ProtoReflect() protoreflect.Message {
	mi := &file_core_contract_market_contract_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MarketCancelOrderContract.ProtoReflect.Descriptor instead.
func (*MarketCancelOrderContract) Descriptor() ([]byte, []int) {
	return file_core_contract_market_contract_proto_rawDescGZIP(), []int{1}
}

func (x *MarketCancelOrderContract) GetOwnerAddress() []byte {
	if x != nil {
		return x.OwnerAddress
	}
	return nil
}

func (x *MarketCancelOrderContract) GetOrderId() []byte {
	if x != nil {
		return x.OrderId
	}
	return nil
}

var File_core_contract_market_contract_proto protoreflect.FileDescriptor

const file_core_contract_market_contract_proto_rawDesc = "" +
	"\n" +
	"#core/contract/market_contract.proto\x12\bprotocol\"\xe2\x01\n" +
	"\x17MarketSellAssetContract\x12#\n" +
	"\rowner_address\x18\x01 \x01(\fR\fownerAddress\x12\"\n" +
	"\rsell_token_id\x18\x02 \x01(\fR\vsellTokenId\x12.\n" +
	"\x13sell_token_quantity\x18\x03 \x01(\x03R\x11sellTokenQuantity\x12 \n" +
	"\fbuy_token_id\x18\x04 \x01(\fR\n" +
	"buyTokenId\x12,\n" +
	"\x12buy_token_quantity\x18\x05 \x01(\x03R\x10buyTokenQuantity\"[\n" +
	"\x19MarketCancelOrderContract\x12#\n" +
	"\rowner_address\x18\x01 \x01(\fR\fownerAddress\x12\x19\n" +
	"\border_id\x18\x02 \x01(\fR\aorderIdBK\n" +
	"\x18org.tron.protos.contractZ/github.com/fbsobreira/gotron-sdk/pkg/proto/coreb\x06proto3"

var (
	file_core_contract_market_contract_proto_rawDescOnce sync.Once
	file_core_contract_market_contract_proto_rawDescData []byte
)

func file_core_contract_market_contract_proto_rawDescGZIP() []byte {
	file_core_contract_market_contract_proto_rawDescOnce.Do(func() {
		file_core_contract_market_contract_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_core_contract_market_contract_proto_rawDesc), len(file_core_contract_market_contract_proto_rawDesc)))
	})
	return file_core_contract_market_contract_proto_rawDescData
}

var file_core_contract_market_contract_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_core_contract_market_contract_proto_goTypes = []any{
	(*MarketSellAssetContract)(nil),   // 0: protocol.MarketSellAssetContract
	(*MarketCancelOrderContract)(nil), // 1: protocol.MarketCancelOrderContract
}
var file_core_contract_market_contract_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_core_contract_market_contract_proto_init() }
func file_core_contract_market_contract_proto_init() {
	if File_core_contract_market_contract_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_core_contract_market_contract_proto_rawDesc), len(file_core_contract_market_contract_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_core_contract_market_contract_proto_goTypes,
		DependencyIndexes: file_core_contract_market_contract_proto_depIdxs,
		MessageInfos:      file_core_contract_market_contract_proto_msgTypes,
	}.Build()
	File_core_contract_market_contract_proto = out.File
	file_core_contract_market_contract_proto_goTypes = nil
	file_core_contract_market_contract_proto_depIdxs = nil
}
