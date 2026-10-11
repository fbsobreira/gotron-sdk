package pqc

import (
	"bytes"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// fnPersistedSeedSize is the length of the FN-DSA-512 key-generation seed
// that wallet-cli stores next to the extended private key.
const fnPersistedSeedSize = 48

// EncryptKeystore encrypts k into a password-protected keystore in the
// format of wallet-cli's post-quantum wallets: a V3 keystore (scrypt,
// AES-128-CTR, Keccak-256 MAC) with a "scheme" field.
//
// The persisted private key is the 2560-byte expanded key for ML-DSA-44 and
// the 2176-byte f||g||F||h for FN-DSA-512. An ML-DSA-44 seed, when k has one,
// is stored as a second segment under the same password. Pass
// keystore.StandardScryptN/StandardScryptP, or LightScryptN/LightScryptP for
// a faster, weaker derivation.
//
// The password is used as UTF-8. wallet-cli encodes characters outside the
// Basic Multilingual Plane (such as emoji) differently, so a password
// containing them will not open the same file in both tools.
func EncryptKeystore(k PrivateKey, password string, scryptN, scryptP int) ([]byte, error) {
	if k == nil {
		return nil, fmt.Errorf("%w: nil private key", ErrInvalidKey)
	}
	name, err := schemeName(k.Scheme())
	if err != nil {
		return nil, err
	}
	var ext, seed []byte
	switch k.Scheme() {
	case core.PQScheme_ML_DSA_44:
		ext = k.Bytes()
		seed = k.Seed()
	case core.PQScheme_FN_DSA_512:
		priv := k.Bytes()
		ext = make([]byte, 0, FNDSA512ExtendedPrivateKeySize)
		ext = append(append(ext, priv...), k.Public().Bytes()...)
		clear(priv)
	}
	defer clear(ext)
	defer clear(seed)

	out, err := keystore.EncryptPQKeyV3(name, k.Public().Address().String(), ext, seed,
		[]byte(password), scryptN, scryptP)
	if err != nil {
		return nil, fmt.Errorf("pqc: encrypting keystore: %w", err)
	}
	return out, nil
}

// DecryptKeystore decrypts a post-quantum keystore written by EncryptKeystore
// or by wallet-cli. A wrong password returns an error wrapping
// keystore.ErrDecrypt.
//
// The key is rebuilt from the persisted private key when present, otherwise
// from the ML-DSA-44 seed. When both are present, the seed must derive the
// same key (else ErrKeyMismatch) and the returned key retains it. An
// FN-DSA-512 seed cannot be expanded portably: it is authenticated and
// length-checked but otherwise ignored, and a keystore holding only an
// FN-DSA-512 seed is rejected. The clear "address" field must be present and
// equal the address of the decrypted key, else ErrKeyMismatch.
func DecryptKeystore(data []byte, password string) (PrivateKey, error) {
	name, addr, ext, seed, err := keystore.DecryptPQKeyV3(data, password)
	if err != nil {
		return nil, fmt.Errorf("pqc: decrypting keystore: %w", err)
	}
	defer clear(ext)
	defer clear(seed)

	scheme, err := schemeFromName(name)
	if err != nil {
		return nil, err
	}
	k, err := keyFromSegments(scheme, ext, seed)
	if err != nil {
		return nil, err
	}
	if addr == "" {
		return nil, fmt.Errorf("%w: keystore has no address", ErrKeyMismatch)
	}
	if err := checkAddress(addr, k); err != nil {
		return nil, err
	}
	return k, nil
}

// keyFromSegments rebuilds a key from the decrypted keystore segments.
func keyFromSegments(scheme core.PQScheme, ext, seed []byte) (PrivateKey, error) {
	if ext == nil {
		if scheme != core.PQScheme_ML_DSA_44 {
			return nil, fmt.Errorf("%w: an FN-DSA-512 keystore without a private key cannot be loaded (Falcon seeds are not portable)",
				ErrInvalidKey)
		}
		return NewMLDSA44FromSeed(seed)
	}

	// Only the persisted forms are accepted: ParsePrivateKey would also take
	// an ML-DSA-44 seed or a bare FN-DSA-512 f||g||F here.
	want := MLDSA44PrivateKeySize
	if scheme == core.PQScheme_FN_DSA_512 {
		want = FNDSA512ExtendedPrivateKeySize
	}
	if len(ext) != want {
		return nil, fmt.Errorf("%w: %s keystore private key must be %d bytes, got %d",
			ErrInvalidKey, scheme, want, len(ext))
	}
	k, err := ParsePrivateKey(scheme, ext, nil)
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return k, nil
	}
	if scheme == core.PQScheme_FN_DSA_512 {
		if len(seed) != fnPersistedSeedSize {
			return nil, fmt.Errorf("%w: FN-DSA-512 keystore seed must be %d bytes, got %d",
				ErrInvalidKey, fnPersistedSeedSize, len(seed))
		}
		return k, nil
	}
	sk, err := NewMLDSA44FromSeed(seed)
	if err != nil {
		return nil, err
	}
	if !sk.Public().Equal(k.Public()) {
		return nil, fmt.Errorf("%w: keystore seed and private key describe different keys", ErrKeyMismatch)
	}
	return sk, nil
}

// checkAddress reports ErrKeyMismatch unless declared is the Base58Check
// address of k.
func checkAddress(declared string, k PrivateKey) error {
	a, err := address.Base58ToAddress(declared)
	if err != nil {
		return fmt.Errorf("%w: address %.40q is not a valid Base58Check address", ErrKeyMismatch, declared)
	}
	if !bytes.Equal(a, k.Public().Address()) {
		return fmt.Errorf("%w: address %.40q does not match the key's address %s",
			ErrKeyMismatch, declared, k.Public().Address())
	}
	return nil
}

// schemeName returns the TIP-899 name of a supported scheme.
func schemeName(scheme core.PQScheme) (string, error) {
	switch scheme {
	case core.PQScheme_FN_DSA_512, core.PQScheme_ML_DSA_44:
		return scheme.String(), nil
	default:
		return "", unsupported(scheme)
	}
}

// schemeFromName parses a supported TIP-899 scheme name ("FN_DSA_512" or
// "ML_DSA_44"); matching is case-sensitive.
func schemeFromName(name string) (core.PQScheme, error) {
	switch name {
	case core.PQScheme_FN_DSA_512.String():
		return core.PQScheme_FN_DSA_512, nil
	case core.PQScheme_ML_DSA_44.String():
		return core.PQScheme_ML_DSA_44, nil
	}
	if len(name) > 32 {
		name = name[:32] + "..."
	}
	return core.PQScheme_UNKNOWN_PQ_SCHEME, fmt.Errorf("%w: %q", ErrUnsupportedScheme, name)
}
