//go:build windows
// +build windows

package keystore

import (
	"fmt"

	"github.com/bizvip/gotron/pkg/address"
)

func RecoverPubkey(hash []byte, signature []byte) (address.Address, error) {
	return nil, fmt.Errorf("not implemented")
}
