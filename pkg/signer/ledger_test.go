package signer

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLedgerSignerWithAddress(t *testing.T) {
	addr, err := address.Base58ToAddress("TPL66VK2gCXNCD7EJg9pgJRfqcRazjhUZY")
	require.NoError(t, err)

	// Construction must not touch a device: no Ledger is attached in tests,
	// and the constructor has no error path that could report a failed open.
	s := NewLedgerSignerWithAddress(addr)
	require.NotNil(t, s)
	assert.Equal(t, addr, s.Address())

	// The signer keeps its own copy of the address.
	addr[1] ^= 0xff
	assert.NotEqual(t, addr, s.Address(), "caller mutation must not leak into the signer")
}

func TestNewLedgerSignerWithAddress_NilAddress(t *testing.T) {
	s := NewLedgerSignerWithAddress(nil)
	require.NotNil(t, s)
	assert.Empty(t, s.Address())
}
