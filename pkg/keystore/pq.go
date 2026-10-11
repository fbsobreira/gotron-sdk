package keystore

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/scrypt"
)

// Post-quantum (TIP-899) keystore, in the format written by wallet-cli
// (feature/post-quantum, org.tron.keystore.Wallet.createPQ). It is a V3
// keystore with a top-level "scheme" and up to two encrypted segments that
// share one scrypt derivation:
//
//   - ext:  "ciphertext", "cipherparams", "mac" — the persisted private key
//   - seed: "seedciphertext", "seedcipherparams", "seedmac" — the keygen seed
//
// The scrypt output DK keys AES-128-CTR with DK[0:16] for both segments; each
// segment has its own random IV and its own MAC Keccak-256(DK[16:32] ||
// ciphertext). This layer is scheme-agnostic: it neither interprets the
// segments nor validates the address; package pqc does both.

// maxPQPlaintextLen bounds each post-quantum keystore segment. The largest
// TIP-899 persisted private key is 2560 bytes (ML-DSA-44 expanded). It is
// separate from maxCiphertextLen, which keeps bounding ECDSA keystores.
const maxPQPlaintextLen = 4096

const pqCipher = "aes-128-ctr"

type pqCipherParamsJSON struct {
	IV *string `json:"iv,omitempty"`
}

// pqCryptoJSON uses pointers so that an absent field (or an explicit JSON
// null, as Jackson writes it) can be told apart from an empty one.
type pqCryptoJSON struct {
	Cipher           string                 `json:"cipher"`
	CipherText       *string                `json:"ciphertext,omitempty"`
	CipherParams     *pqCipherParamsJSON    `json:"cipherparams,omitempty"`
	KDF              string                 `json:"kdf"`
	KDFParams        map[string]interface{} `json:"kdfparams"`
	MAC              *string                `json:"mac,omitempty"`
	SeedCipherText   *string                `json:"seedciphertext,omitempty"`
	SeedCipherParams *pqCipherParamsJSON    `json:"seedcipherparams,omitempty"`
	SeedMAC          *string                `json:"seedmac,omitempty"`
}

// pqKeyJSONV3 is the top level of a post-quantum keystore. encoding/json
// matches keys case-insensitively, so the legacy "Crypto" spelling that
// wallet-cli also reads is accepted.
type pqKeyJSONV3 struct {
	Address string        `json:"address"`
	ID      string        `json:"id"`
	Version int           `json:"version"`
	Scheme  string        `json:"scheme"`
	Crypto  *pqCryptoJSON `json:"crypto"`
}

// pqSegment is one decoded, not yet authenticated, encrypted segment.
type pqSegment struct {
	mac, iv, cipherText []byte
}

// EncryptPQKeyV3 encrypts post-quantum key material into a wallet-cli
// compatible keystore. scheme is the TIP-899 scheme name ("FN_DSA_512",
// "ML_DSA_44") and address the Base58Check address; both are stored in clear
// and are not validated here. ext is the persisted private key and seed the
// key-generation seed; at least one must be non-empty, an empty one is left
// out of the file, and each is limited to 4096 bytes. scryptN and scryptP
// must be within the limits DecryptPQKeyV3 accepts (for example
// StandardScryptN/StandardScryptP or LightScryptN/LightScryptP).
//
// Experimental: this is a byte-level helper for package pqc; use
// pqc.EncryptKeystore instead. It may change with TIP-899.
func EncryptPQKeyV3(scheme, address string, ext, seed, auth []byte, scryptN, scryptP int) ([]byte, error) {
	if scheme == "" {
		return nil, errors.New("pq keystore: scheme is required")
	}
	if address == "" {
		return nil, errors.New("pq keystore: address is required")
	}
	if len(ext) == 0 && len(seed) == 0 {
		return nil, errors.New("pq keystore: at least one of the private key and the seed is required")
	}
	if len(ext) > maxPQPlaintextLen || len(seed) > maxPQPlaintextLen {
		return nil, fmt.Errorf("pq keystore: plaintext length exceeds limit %d", maxPQPlaintextLen)
	}
	// Refuse parameters the reader would reject, so a file is never written
	// that this package cannot open again.
	if scryptN < 2 || scryptN > maxScryptN {
		return nil, fmt.Errorf("pq keystore: scrypt N must be in [2, %d], got %d", maxScryptN, scryptN)
	}
	if scryptP < 1 || scryptP > maxScryptP {
		return nil, fmt.Errorf("pq keystore: scrypt P must be in [1, %d], got %d", maxScryptP, scryptP)
	}
	if mem := 128 * int64(scryptN) * int64(scryptR); mem > maxScryptMemory {
		return nil, fmt.Errorf("pq keystore: scrypt would allocate %d bytes, limit is %d", mem, maxScryptMemory)
	}

	salt, err := pqRandom(32)
	if err != nil {
		return nil, err
	}
	derivedKey, err := scrypt.Key(auth, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, fmt.Errorf("pq keystore: %w", err)
	}
	defer clear(derivedKey)

	cj := &pqCryptoJSON{
		Cipher: pqCipher,
		KDF:    keyHeaderKDF,
		KDFParams: map[string]interface{}{
			"dklen": scryptDKLen,
			"n":     scryptN,
			"p":     scryptP,
			"r":     scryptR,
			"salt":  hex.EncodeToString(salt),
		},
	}

	var extIV []byte
	if len(ext) > 0 {
		if extIV, err = pqRandom(aes.BlockSize); err != nil {
			return nil, err
		}
		ct, params, mac, err := pqEncryptSegment(derivedKey, extIV, ext)
		if err != nil {
			return nil, err
		}
		cj.CipherText, cj.CipherParams, cj.MAC = ct, params, mac
	}
	if len(seed) > 0 {
		// AES-CTR under one key with a repeated IV leaks the XOR of the
		// plaintexts, so the seed IV must differ from the ext IV.
		var seedIV []byte
		for seedIV == nil || bytes.Equal(seedIV, extIV) {
			if seedIV, err = pqRandom(aes.BlockSize); err != nil {
				return nil, err
			}
		}
		ct, params, mac, err := pqEncryptSegment(derivedKey, seedIV, seed)
		if err != nil {
			return nil, err
		}
		cj.SeedCipherText, cj.SeedCipherParams, cj.SeedMAC = ct, params, mac
	}

	return json.Marshal(pqKeyJSONV3{
		Address: address,
		ID:      uuid.NewRandom().String(),
		Version: version,
		Scheme:  scheme,
		Crypto:  cj,
	})
}

// DecryptPQKeyV3 decrypts a wallet-cli post-quantum keystore and returns its
// clear scheme and address fields with the decrypted segments; an absent
// segment is returned as nil. The address is not validated here.
//
// The file must be version 3, aes-128-ctr and scrypt, with a scheme and at
// least one complete segment; a segment with only some of its fields is
// rejected, as are two segments sharing an IV. Every present MAC is verified
// before any segment is decrypted. A MAC mismatch (normally a wrong password)
// returns ErrDecrypt.
//
// Experimental: this is a byte-level helper for package pqc; use
// pqc.DecryptKeystore instead. It may change with TIP-899.
func DecryptPQKeyV3(keyjson []byte, auth string) (scheme, address string, ext, seed []byte, err error) {
	var k pqKeyJSONV3
	if err := json.Unmarshal(keyjson, &k); err != nil {
		return "", "", nil, nil, fmt.Errorf("pq keystore: %w", err)
	}
	if k.Version != version {
		return "", "", nil, nil, fmt.Errorf("pq keystore: version %d not supported", k.Version)
	}
	if k.Scheme == "" {
		return "", "", nil, nil, errors.New("pq keystore: missing scheme")
	}
	cj := k.Crypto
	if cj == nil {
		return "", "", nil, nil, errors.New("pq keystore: missing crypto section")
	}
	if cj.Cipher != pqCipher {
		return "", "", nil, nil, fmt.Errorf("pq keystore: cipher %q not supported", cj.Cipher)
	}
	if cj.KDF != keyHeaderKDF {
		return "", "", nil, nil, fmt.Errorf("pq keystore: kdf %q not supported, scrypt is required", cj.KDF)
	}

	extSeg, err := pqDecodeSegment("private key", cj.CipherText, cj.CipherParams, cj.MAC)
	if err != nil {
		return "", "", nil, nil, err
	}
	seedSeg, err := pqDecodeSegment("seed", cj.SeedCipherText, cj.SeedCipherParams, cj.SeedMAC)
	if err != nil {
		return "", "", nil, nil, err
	}
	if extSeg == nil && seedSeg == nil {
		return "", "", nil, nil, errors.New("pq keystore: no encrypted segment")
	}
	if extSeg != nil && seedSeg != nil && bytes.Equal(extSeg.iv, seedSeg.iv) {
		return "", "", nil, nil, errors.New("pq keystore: private key and seed segments reuse the same IV")
	}

	derivedKey, err := getKDFKey(CryptoJSON{KDF: cj.KDF, KDFParams: cj.KDFParams}, auth)
	if err != nil {
		return "", "", nil, nil, err
	}
	defer clear(derivedKey)

	// Authenticate every segment before decrypting any of them.
	for _, s := range []*pqSegment{extSeg, seedSeg} {
		if s == nil {
			continue
		}
		mac := crypto.Keccak256(derivedKey[16:32], s.cipherText)
		if subtle.ConstantTimeCompare(mac, s.mac) != 1 {
			return "", "", nil, nil, ErrDecrypt
		}
	}

	if extSeg != nil {
		if ext, err = aesCTRXOR(derivedKey[:16], extSeg.cipherText, extSeg.iv); err != nil {
			return "", "", nil, nil, err
		}
	}
	if seedSeg != nil {
		if seed, err = aesCTRXOR(derivedKey[:16], seedSeg.cipherText, seedSeg.iv); err != nil {
			clear(ext)
			return "", "", nil, nil, err
		}
	}
	return k.Scheme, k.Address, ext, seed, nil
}

// pqDecodeSegment returns nil for an absent segment (all three fields
// missing), an error for a partial one, and the bounded, hex-decoded fields
// otherwise. It runs before the KDF so malformed input is rejected cheaply.
func pqDecodeSegment(name string, cipherText *string, params *pqCipherParamsJSON, mac *string) (*pqSegment, error) {
	if cipherText == nil && params == nil && mac == nil {
		return nil, nil
	}
	if cipherText == nil || params == nil || params.IV == nil || mac == nil {
		return nil, fmt.Errorf("pq keystore: %s segment is incomplete", name)
	}
	m, iv, ct, err := decodeCipherFieldsMax(*mac, *params.IV, *cipherText, false, maxPQPlaintextLen)
	if err != nil {
		return nil, fmt.Errorf("pq keystore: %s segment: %w", name, err)
	}
	return &pqSegment{mac: m, iv: iv, cipherText: ct}, nil
}

func pqEncryptSegment(derivedKey, iv, plain []byte) (cipherText *string, params *pqCipherParamsJSON, mac *string, err error) {
	ct, err := aesCTRXOR(derivedKey[:16], plain, iv)
	if err != nil {
		return nil, nil, nil, err
	}
	ctHex := hex.EncodeToString(ct)
	ivHex := hex.EncodeToString(iv)
	macHex := hex.EncodeToString(crypto.Keccak256(derivedKey[16:32], ct))
	return &ctHex, &pqCipherParamsJSON{IV: &ivHex}, &macHex, nil
}

func pqRandom(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("pq keystore: reading random bytes: %w", err)
	}
	return b, nil
}
