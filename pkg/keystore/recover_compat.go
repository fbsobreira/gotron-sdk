//go:build windows || tronsdk_compat
// +build windows tronsdk_compat

package keystore

import (
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
)

func RecoverPubkey(hash []byte, signature []byte) (address.Address, error) {
	return nil, fmt.Errorf("not implemented")
}
