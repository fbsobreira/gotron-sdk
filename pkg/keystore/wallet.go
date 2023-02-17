// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package keystore

import (
	"bytes"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// keystoreWallet implements the Wallet interface for the original
// keystore.
type keystoreWallet struct {
	account  Account   // Single account contained in this wallet
	keystore *KeyStore // Keystore where the account originates from
}

// URL implements Wallet, returning the URL of the account within.
func (w *keystoreWallet) URL() URL {
	return w.account.URL
}

// Status implements Wallet, returning whether the account held by the
// keystore wallet is unlocked or not.
func (w *keystoreWallet) Status() (string, error) {
	w.keystore.mu.RLock()
	defer w.keystore.mu.RUnlock()

	if _, ok := w.keystore.unlocked[w.account.Address.String()]; ok {
		return "Unlocked", nil
	}
	return "Locked", nil
}

// Open implements Wallet, but is a noop for plain wallets since there
// is no connection or decryption step necessary to access the list of account.
func (w *keystoreWallet) Open(passphrase string) error { return nil }

// Close implements Wallet, but is a noop for plain wallets since there
// is no meaningful open operation.
func (w *keystoreWallet) Close() error { return nil }

// Accounts implements Wallet, returning an account list consisting of
// a single account that the plain kestore wallet contains.
func (w *keystoreWallet) Accounts() []Account {
	return []Account{w.account}
}

// Contains implements Wallet, returning whether a particular account is
// or is not wrapped by this wallet instance.
func (w *keystoreWallet) Contains(account Account) bool {
	return bytes.Equal(account.Address, w.account.Address) && (account.URL == (URL{}) || account.URL == w.account.URL)
}

// Derive implements Wallet, but is a noop for plain wallets since there
// is no notion of hierarchical account derivation for plain keystore account.
func (w *keystoreWallet) Derive(path DerivationPath, pin bool) (Account, error) {
	return Account{}, ErrNotSupported
}

// signHash attempts to sign the given hash with
// the given account. If the wallet does not wrap this particular account, an
// error is returned to avoid account leakage (even though in theory we may be
// able to sign via our shared keystore backend).
func (w *keystoreWallet) signHash(acc Account, hash []byte) ([]byte, error) {
	// Make sure the requested account is contained within
	if !w.Contains(acc) {
		return nil, ErrUnknownAccount
	}
	// Account seems valid, request the keystore to sign
	return w.keystore.SignHash(acc, hash)
}

// SignData signs keccak256(data). The mimetype parameter describes the type of data being signed
func (w *keystoreWallet) SignData(acc Account, mimeType string, data []byte) ([]byte, error) {
	return w.signHash(acc, crypto.Keccak256(data))
}

// SignDataWithPassphrase signs keccak256(data). The mimetype parameter describes the type of data being signed
func (w *keystoreWallet) SignDataWithPassphrase(acc Account, passphrase, mimeType string, data []byte) ([]byte, error) {
	// Make sure the requested account is contained within
	if !w.Contains(acc) {
		return nil, ErrUnknownAccount
	}
	// Account seems valid, request the keystore to sign
	return w.keystore.SignHashWithPassphrase(acc, passphrase, crypto.Keccak256(data))
}

func (w *keystoreWallet) SignText(acc Account, text []byte, useFixedLength ...bool) ([]byte, error) {
	return w.signHash(acc, TextHash(text, useFixedLength...))
}

// SignTextWithPassphrase implements Wallet, attempting to sign the
// given hash with the given account using passphrase as extra authentication.
func (w *keystoreWallet) SignTextWithPassphrase(acc Account, passphrase string, text []byte) ([]byte, error) {
	// Make sure the requested account is contained within
	if !w.Contains(acc) {
		return nil, ErrUnknownAccount
	}
	// Account seems valid, request the keystore to sign
	return w.keystore.SignHashWithPassphrase(acc, passphrase, TextHash(text))
}

// SignTx implements Wallet, attempting to sign the given transaction
// with the given account. If the wallet does not wrap this particular account,
// an error is returned to avoid account leakage (even though in theory we may
// be able to sign via our shared keystore backend).
func (w *keystoreWallet) SignTx(acc Account, tx *core.Transaction) (*core.Transaction, error) {
	// Make sure the requested account is contained within
	if !w.Contains(acc) {
		return nil, ErrUnknownAccount
	}
	// Account seems valid, request the keystore to sign
	return w.keystore.SignTx(acc, tx)
}

// SignTxWithPassphrase implements Wallet, attempting to sign the given
// transaction with the given account using passphrase as extra authentication.
func (w *keystoreWallet) SignTxWithPassphrase(acc Account, passphrase string, tx *core.Transaction) (*core.Transaction, error) {
	// Make sure the requested account is contained within
	if !w.Contains(acc) {
		return nil, ErrUnknownAccount
	}
	// Account seems valid, request the keystore to sign
	return w.keystore.SignTxWithPassphrase(acc, passphrase, tx)
}
