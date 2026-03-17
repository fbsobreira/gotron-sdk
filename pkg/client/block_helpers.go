package client

import (
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// blockHeaderWitnessAddress extracts the witness address from a BlockHeader.
// Returns nil if the header or raw data is nil, if the witness address is empty,
// or if the bytes do not form a valid TRON address.
// The returned Address is a copy and does not alias the protobuf backing array.
func blockHeaderWitnessAddress(header *core.BlockHeader) address.Address {
	if header == nil || header.RawData == nil || len(header.RawData.WitnessAddress) == 0 {
		return nil
	}
	buf := make([]byte, len(header.RawData.WitnessAddress))
	copy(buf, header.RawData.WitnessAddress)
	addr := address.Address(buf)
	if !addr.IsValid() {
		return nil
	}
	return addr
}

// BlockExtentionWitnessAddress returns the witness address from a BlockExtention
// as an address.Address. Returns nil if the block or header is nil.
func BlockExtentionWitnessAddress(block *api.BlockExtention) address.Address {
	if block == nil {
		return nil
	}
	return blockHeaderWitnessAddress(block.BlockHeader)
}

// BlockExtentionWitnessBase58 returns the witness address from a BlockExtention
// as a base58-encoded string. Returns an empty string if the block or header is nil.
func BlockExtentionWitnessBase58(block *api.BlockExtention) string {
	addr := BlockExtentionWitnessAddress(block)
	if addr == nil {
		return ""
	}
	return addr.String()
}

// BlockWitnessAddress returns the witness address from a Block
// as an address.Address. Returns nil if the block or header is nil.
func BlockWitnessAddress(block *core.Block) address.Address {
	if block == nil {
		return nil
	}
	return blockHeaderWitnessAddress(block.BlockHeader)
}

// BlockWitnessBase58 returns the witness address from a Block
// as a base58-encoded string. Returns an empty string if the block or header is nil.
func BlockWitnessBase58(block *core.Block) string {
	addr := BlockWitnessAddress(block)
	if addr == nil {
		return ""
	}
	return addr.String()
}
