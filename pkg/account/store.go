package account

import "github.com/fbsobreira/gotron-sdk/pkg/keystore"

// Store abstracts the account storage backend so that functions in this
// package are not coupled to the global store package.
type Store interface {
	FromAddress(addr string) *keystore.KeyStore
	FromAccountName(name string) *keystore.KeyStore
	DoesNamedAccountExist(name string) bool
	LocalAccounts() []string
	DefaultLocation() string
	InitConfigDir()
}
