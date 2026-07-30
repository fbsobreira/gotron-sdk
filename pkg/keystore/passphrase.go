// Copyright 2014 The go-ethereum Authors
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

/*

This key store behaves as KeyStorePlain with the difference that
the private key is encrypted and on disk uses another JSON encoding.

The crypto is documented at https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition

*/

package keystore

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

const (
	keyHeaderKDF = "scrypt"

	// StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptN = 1 << 18

	// StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptP = 1

	// LightScryptN is the N parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptN = 1 << 12

	// LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptP = 6

	scryptR     = 8
	scryptDKLen = 32
)

type keyStorePassphrase struct {
	keysDirPath string
	scryptN     int
	scryptP     int
	// skipKeyFileVerification disables the security-feature which does
	// reads and decrypts any newly created keyfiles. This should be 'false' in all
	// cases except tests -- setting this to 'true' is not recommended.
	skipKeyFileVerification bool
}

func (ks keyStorePassphrase) GetKey(addr address.Address, filename, auth string) (*Key, error) {
	// Load the key from the keystore and decrypt its contents
	keyjson, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	key, err := DecryptKey(keyjson, auth)
	if err != nil {
		return nil, err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if !bytes.Equal(key.Address, addr) {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
	}
	return key, nil
}

// StoreKey generates a key, encrypts with 'auth' and stores in the given directory
func StoreKey(dir, auth string, scryptN, scryptP int) (Account, error) {
	_, a, err := storeNewKey(&keyStorePassphrase{dir, scryptN, scryptP, false}, rand.Reader, auth)
	return a, err
}

func (ks keyStorePassphrase) StoreKey(filename string, key *Key, auth string) error {
	keyjson, err := EncryptKey(key, auth, ks.scryptN, ks.scryptP)
	if err != nil {
		return err
	}
	// Write into temporary file
	tmpName, err := writeTemporaryKeyFile(filename, keyjson)
	if err != nil {
		return err
	}
	if !ks.skipKeyFileVerification {
		// Verify that we can decrypt the file with the given password.
		_, err = ks.GetKey(key.Address, tmpName, auth)
		if err != nil {
			msg := "An error was encountered when saving and verifying the keystore file. \n" +
				"This indicates that the keystore is corrupted. \n" +
				"The corrupted file is stored at \n%v\n" +
				"Please file a ticket at:\n\n" +
				"https://github.com/ethereum/go-ethereum/issues." +
				"The error was : %s"
			return fmt.Errorf(msg, tmpName, err)
		}
	}
	return os.Rename(tmpName, filename)
}

func (ks keyStorePassphrase) JoinPath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	return filepath.Join(ks.keysDirPath, filename)
}

// EncryptDataV3 encrypts the data given as 'data' with the password 'auth'.
func EncryptDataV3(data, auth []byte, scryptN, scryptP int) (CryptoJSON, error) {

	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	derivedKey, err := scrypt.Key(auth, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return CryptoJSON{}, err
	}
	encryptKey := derivedKey[:16]

	iv := make([]byte, aes.BlockSize) // 16
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	cipherText, err := aesCTRXOR(encryptKey, data, iv)
	if err != nil {
		return CryptoJSON{}, err
	}
	mac := crypto.Keccak256(derivedKey[16:32], cipherText)

	scryptParamsJSON := make(map[string]interface{}, 5)
	scryptParamsJSON["n"] = scryptN
	scryptParamsJSON["r"] = scryptR
	scryptParamsJSON["p"] = scryptP
	scryptParamsJSON["dklen"] = scryptDKLen
	scryptParamsJSON["salt"] = hex.EncodeToString(salt)
	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	cryptoStruct := CryptoJSON{
		Cipher:       "aes-128-ctr",
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    scryptParamsJSON,
		MAC:          hex.EncodeToString(mac),
	}
	return cryptoStruct, nil
}

// EncryptKey encrypts a key using the specified scrypt parameters into a json
// blob that can be decrypted later on.
func EncryptKey(key *Key, auth string, scryptN, scryptP int) ([]byte, error) {
	keyBytes := math.PaddedBigBytes(key.PrivateKey.D, 32)
	cryptoStruct, err := EncryptDataV3(keyBytes, []byte(auth), scryptN, scryptP)
	if err != nil {
		return nil, err
	}
	encryptedKeyJSONV3 := encryptedKeyJSONV3{
		hex.EncodeToString(key.Address[:]),
		cryptoStruct,
		key.ID.String(),
		version,
	}
	return json.Marshal(encryptedKeyJSONV3)
}

// DecryptKey decrypts a key from a json blob, returning the private key itself.
func DecryptKey(keyjson []byte, auth string) (*Key, error) {
	// Parse the json into a simple map to fetch the key version
	m := make(map[string]interface{})
	if err := json.Unmarshal(keyjson, &m); err != nil {
		return nil, err
	}
	// Depending on the version try to parse one way or another
	var (
		keyBytes, keyID []byte
		err             error
	)
	if version, ok := m["version"].(string); ok && version == "1" {
		k := new(encryptedKeyJSONV1)
		if err := json.Unmarshal(keyjson, k); err != nil {
			return nil, err
		}
		keyBytes, keyID, err = decryptKeyV1(k, auth)
	} else {
		k := new(encryptedKeyJSONV3)
		if err := json.Unmarshal(keyjson, k); err != nil {
			return nil, err
		}
		keyBytes, keyID, err = decryptKeyV3(k, auth)
	}
	// Handle any decryption errors and return the key
	if err != nil {
		return nil, err
	}
	key := crypto.ToECDSAUnsafe(keyBytes)

	return &Key{
		ID:         uuid.UUID(keyID),
		Address:    address.PubkeyToAddress(key.PublicKey),
		PrivateKey: key,
	}, nil
}

// DecryptDataV3 ...
func DecryptDataV3(cj CryptoJSON, auth string) ([]byte, error) {
	if cj.Cipher != "aes-128-ctr" {
		return nil, fmt.Errorf("Cipher not supported: %v", cj.Cipher)
	}
	mac, err := hex.DecodeString(cj.MAC)
	if err != nil {
		return nil, err
	}

	iv, err := hex.DecodeString(cj.CipherParams.IV)
	if err != nil {
		return nil, err
	}

	cipherText, err := hex.DecodeString(cj.CipherText)
	if err != nil {
		return nil, err
	}

	derivedKey, err := getKDFKey(cj, auth)
	if err != nil {
		return nil, err
	}

	calculatedMAC := crypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, ErrDecrypt
	}

	plainText, err := aesCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return nil, err
	}
	return plainText, err
}

func decryptKeyV3(keyProtected *encryptedKeyJSONV3, auth string) (keyBytes []byte, keyID []byte, err error) {
	if keyProtected.Version != version {
		return nil, nil, fmt.Errorf("Version not supported: %v", keyProtected.Version)
	}
	keyID = uuid.Parse(keyProtected.ID)
	plainText, err := DecryptDataV3(keyProtected.Crypto, auth)
	if err != nil {
		return nil, nil, err
	}
	return plainText, keyID, err
}

func decryptKeyV1(keyProtected *encryptedKeyJSONV1, auth string) (keyBytes []byte, keyID []byte, err error) {
	keyID = uuid.Parse(keyProtected.ID)
	mac, err := hex.DecodeString(keyProtected.Crypto.MAC)
	if err != nil {
		return nil, nil, err
	}

	iv, err := hex.DecodeString(keyProtected.Crypto.CipherParams.IV)
	if err != nil {
		return nil, nil, err
	}

	cipherText, err := hex.DecodeString(keyProtected.Crypto.CipherText)
	if err != nil {
		return nil, nil, err
	}

	derivedKey, err := getKDFKey(keyProtected.Crypto, auth)
	if err != nil {
		return nil, nil, err
	}

	calculatedMAC := crypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, nil, ErrDecrypt
	}

	plainText, err := aesCBCDecrypt(crypto.Keccak256(derivedKey[:16])[:16], cipherText, iv)
	if err != nil {
		return nil, nil, err
	}
	return plainText, keyID, err
}

// KDF parameter limits.
//
// getKDFKey consumes these values *before* the MAC is verified, so they are
// unauthenticated attacker-controlled input. Without upper bounds a crafted
// keystore file forces an arbitrarily expensive derivation — memory and CPU
// exhaustion — before anything establishes that the file is even genuine.
const (
	maxScryptN = 1 << 22
	maxScryptR = 64
	maxScryptP = 16
	// scrypt's working set is 128 * N * r bytes. StandardScryptN with r=8 needs
	// 256 MiB, so 1 GiB leaves generous headroom while still bounding the cost.
	maxScryptMemory = 1 << 30
	maxPBKDF2Count  = 1 << 24
	// The decrypt paths read derivedKey[:16] for the AES key and derivedKey[16:32]
	// for the MAC, so anything shorter than 32 is unusable. It does not panic today
	// only because scrypt.Key and pbkdf2.Key happen to return a slice with cap 32
	// even when dklen is 1, so the expression reads past len into the spare
	// capacity. Requiring the V3 spec's 32 removes that reliance.
	minDerivedKeyLen = 32
	maxDerivedKeyLen = 1024
)

// kdfString reads a string KDF parameter with a checked assertion. The params map
// is decoded from caller-supplied JSON, so a missing key yields nil and an
// unchecked assertion would panic.
func kdfString(params map[string]interface{}, key string) (string, error) {
	v, ok := params[key]
	if !ok {
		return "", fmt.Errorf("kdf params: missing %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("kdf params: %q must be a string, got %T", key, v)
	}
	return s, nil
}

// kdfInt reads a numeric KDF parameter and enforces [minv, maxv]. It replaces
// ensureInt, which asserted straight to float64 with no comma-ok and panicked on
// a missing key or any other type.
func kdfInt(params map[string]interface{}, key string, minv, maxv int) (int, error) {
	v, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("kdf params: missing %q", key)
	}
	var n int
	switch t := v.(type) {
	case int:
		n = t
	case int64:
		n = int(t)
	case float64:
		// encoding/json decodes every JSON number into float64. Range-check before
		// converting: converting an out-of-range float to int is undefined in Go.
		if t < float64(minv) || t > float64(maxv) {
			return 0, fmt.Errorf("kdf params: %q must be in [%d, %d], got %v", key, minv, maxv, t)
		}
		n = int(t)
		if float64(n) != t {
			return 0, fmt.Errorf("kdf params: %q must be a whole number, got %v", key, t)
		}
	default:
		return 0, fmt.Errorf("kdf params: %q must be a number, got %T", key, v)
	}
	if n < minv || n > maxv {
		return 0, fmt.Errorf("kdf params: %q must be in [%d, %d], got %d", key, minv, maxv, n)
	}
	return n, nil
}

func getKDFKey(cryptoJSON CryptoJSON, auth string) ([]byte, error) {
	authArray := []byte(auth)
	saltHex, err := kdfString(cryptoJSON.KDFParams, "salt")
	if err != nil {
		return nil, err
	}
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil, err
	}
	dkLen, err := kdfInt(cryptoJSON.KDFParams, "dklen", minDerivedKeyLen, maxDerivedKeyLen)
	if err != nil {
		return nil, err
	}

	switch cryptoJSON.KDF {
	case keyHeaderKDF:
		n, err := kdfInt(cryptoJSON.KDFParams, "n", 2, maxScryptN)
		if err != nil {
			return nil, err
		}
		r, err := kdfInt(cryptoJSON.KDFParams, "r", 1, maxScryptR)
		if err != nil {
			return nil, err
		}
		p, err := kdfInt(cryptoJSON.KDFParams, "p", 1, maxScryptP)
		if err != nil {
			return nil, err
		}
		if mem := 128 * int64(n) * int64(r); mem > maxScryptMemory {
			return nil, fmt.Errorf("kdf params: scrypt would allocate %d bytes, limit is %d", mem, maxScryptMemory)
		}
		return scrypt.Key(authArray, salt, n, r, p, dkLen)
	case "pbkdf2":
		c, err := kdfInt(cryptoJSON.KDFParams, "c", 1, maxPBKDF2Count)
		if err != nil {
			return nil, err
		}
		prf, err := kdfString(cryptoJSON.KDFParams, "prf")
		if err != nil {
			return nil, err
		}
		if prf != "hmac-sha256" {
			return nil, fmt.Errorf("Unsupported PBKDF2 PRF: %s", prf)
		}
		key := pbkdf2.Key(authArray, salt, c, dkLen, sha256.New)
		return key, nil
	default:
		return nil, fmt.Errorf("Unsupported KDF: %s", cryptoJSON.KDF)
	}
}
