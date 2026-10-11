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

// NameExists reports whether name is used by any local account, ECDSA
// (account-keys) or post-quantum (pq-keys). It is false for invalid names.
func (defaultStore) NameExists(name string) bool {
	return store.NameExists(name)
}

// nameChecker is an optional Store extension for stores that can tell
// whether a name is used by any kind of account, not only ECDSA ones. It is
// kept out of Store so existing implementations keep compiling.
type nameChecker interface {
	NameExists(name string) bool
}

// nameTaken reports whether name is already used in s. Stores without
// NameExists fall back to DoesNamedAccountExist.
func nameTaken(s Store, name string) bool {
	if nc, ok := s.(nameChecker); ok {
		return nc.NameExists(name)
	}
	return s.DoesNamedAccountExist(name)
}
