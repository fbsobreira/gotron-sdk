package cmd

import (
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/pkg/errors"
)

type tronAddress struct {
	address string
}

func (tronAddress tronAddress) String() string {
	return tronAddress.address
}

func (tronAddress *tronAddress) Set(s string) error {
	_, err := address.Base58ToAddress(s)
	if err != nil {
		return errors.Wrap(err, "not a valid one address")
	}
	tronAddress.address = s
	return nil
}

func (tronAddress *tronAddress) GetAddress() address.Address {
	addr, err := address.Base58ToAddress(tronAddress.address)
	if err != nil {
		return nil
	}
	return addr
}

func (tronAddress tronAddress) Type() string {
	return "tron-address"
}
