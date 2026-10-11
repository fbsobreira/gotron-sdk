package account

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fbsobreira/gotron-sdk/pkg/store"
)

// ErrAccountNotFound matches, under errors.Is, the error RemoveAccount
// returns when the named account does not exist in the local store.
var ErrAccountNotFound = errors.New("account not found")

// RemoveAccount deletes the named account and its keystore files from the local store.
func RemoveAccount(name string) error {
	accountExists := store.DoesNamedAccountExist(name)

	if !accountExists {
		return &accountNotFoundError{name: name}
	}

	accountDir := filepath.Join(store.DefaultLocation(), name)
	if err := os.RemoveAll(accountDir); err != nil {
		return fmt.Errorf("failed to remove account %s: %w", name, err)
	}

	return nil
}

// accountNotFoundError keeps RemoveAccount's original message while matching
// ErrAccountNotFound under errors.Is.
type accountNotFoundError struct {
	name string
}

func (e *accountNotFoundError) Error() string {
	return fmt.Sprintf("account %s doesn't exist", e.name)
}

func (e *accountNotFoundError) Is(target error) bool {
	return target == ErrAccountNotFound
}
