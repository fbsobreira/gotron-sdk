package account

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
)

// RemoveAccount - removes an account from the keystore
func RemoveAccount(name string) error {
	accountExists := store.DoesNamedAccountExist(name)

	if !accountExists {
		return fmt.Errorf("account %s doesn't exist", name)
	}

	accountDir := filepath.Join(store.DefaultLocation(), name)
	os.RemoveAll(accountDir)

	return nil
}
