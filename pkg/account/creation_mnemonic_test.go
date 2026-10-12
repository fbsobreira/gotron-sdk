package account_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/account"
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateNewLocalAccount_RejectsInvalidMnemonic(t *testing.T) {
	tests := []struct {
		name     string
		mnemonic string
	}{
		{"bad checksum", strings.TrimSpace(strings.Repeat("abandon ", 12))},
		{"wrong word count", "abandon abandon abandon"},
		{"unknown word", strings.Repeat("abandon ", 11) + "notaword"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestStore(t)
			const name = "bad-mnemonic-account"

			err := account.CreateNewLocalAccount(&account.Creation{
				Name:       name,
				Passphrase: "pass",
				Mnemonic:   tt.mnemonic,
			})
			require.Error(t, err)
			assert.True(t, errors.Is(err, mnemonic.ErrInvalidMnemonic),
				"error must match ErrInvalidMnemonic, got %v", err)

			assert.False(t, store.DoesNamedAccountExist(name), "no account may be created")
			_, statErr := os.Stat(filepath.Join(store.DefaultLocation(), name))
			assert.True(t, os.IsNotExist(statErr), "no account folder may be left behind, stat: %v", statErr)
		})
	}
}
