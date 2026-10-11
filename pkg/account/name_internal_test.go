package account

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/stretchr/testify/assert"
)

// legacyStore implements Store without the optional NameExists method.
type legacyStore struct {
	ecdsa map[string]bool
}

func (legacyStore) FromAddress(string) *keystore.KeyStore     { return nil }
func (legacyStore) FromAccountName(string) *keystore.KeyStore { return nil }
func (s legacyStore) DoesNamedAccountExist(name string) bool  { return s.ecdsa[name] }
func (legacyStore) LocalAccounts() []string                   { return nil }
func (legacyStore) DefaultLocation() string                   { return "" }
func (legacyStore) InitConfigDir()                            {}

// fullStore also implements the optional NameExists method.
type fullStore struct {
	legacyStore
	all map[string]bool
}

func (s fullStore) NameExists(name string) bool { return s.all[name] }

func TestNameTaken(t *testing.T) {
	legacy := legacyStore{ecdsa: map[string]bool{"ecdsa": true}}
	full := fullStore{legacyStore: legacy, all: map[string]bool{"ecdsa": true, "pq": true}}

	tests := []struct {
		name  string
		store Store
		acct  string
		want  bool
	}{
		{name: "legacy store falls back to ECDSA names", store: legacy, acct: "ecdsa", want: true},
		{name: "legacy store cannot see PQ names", store: legacy, acct: "pq", want: false},
		{name: "NameExists store sees PQ names", store: full, acct: "pq", want: true},
		{name: "NameExists store sees ECDSA names", store: full, acct: "ecdsa", want: true},
		{name: "free name", store: full, acct: "free", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, nameTaken(tt.store, tt.acct))
		})
	}
}

func TestDefaultStoreImplementsNameChecker(t *testing.T) {
	_, ok := DefaultStore().(nameChecker)
	assert.True(t, ok, "DefaultStore must see post-quantum names")
}
