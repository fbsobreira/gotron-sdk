package account

import (
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
)

// defaultStore wraps the global store package functions to satisfy the Store interface.
type defaultStore struct{}

// DefaultStore returns a Store backed by the global store package.
func DefaultStore() Store {
	return defaultStore{}
}

func (defaultStore) FromAddress(addr string) *keystore.KeyStore {
	return store.FromAddress(addr)
}

func (defaultStore) FromAccountName(name string) *keystore.KeyStore {
	return store.FromAccountName(name)
}

func (defaultStore) DoesNamedAccountExist(name string) bool {
	return store.DoesNamedAccountExist(name)
}

func (defaultStore) LocalAccounts() []string {
	return store.LocalAccounts()
}

func (defaultStore) DefaultLocation() string {
	return store.DefaultLocation()
}

func (defaultStore) InitConfigDir() {
	store.InitConfigDir()
}
