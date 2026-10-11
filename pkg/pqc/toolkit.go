package pqc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// toolkitKeyJSON is the plaintext key file of java-tron's
// "Toolkit.jar pq-key new" (PqKeyNew) as loaded by the node's witness
// configuration (PqKeyFile). Field order is the order Toolkit writes.
type toolkitKeyJSON struct {
	Scheme     string `json:"scheme"`
	Seed       string `json:"seed,omitempty"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	Address    string `json:"address"`
}

// MarshalToolkitJSON encodes k as an unencrypted key file in the format of
// java-tron's "Toolkit.jar pq-key new": scheme, seed (ML-DSA-44 keys that
// have one), privateKey, publicKey and address, with lowercase hex, a
// two-space indent and a trailing newline.
//
// privateKey is the 2560-byte expanded key for ML-DSA-44 and the 1280-byte
// f||g||F for FN-DSA-512. The output holds the private key in clear: protect
// it like the key itself, or use EncryptKeystore.
func MarshalToolkitJSON(k PrivateKey) ([]byte, error) {
	if k == nil {
		return nil, fmt.Errorf("%w: nil private key", ErrInvalidKey)
	}
	name, err := schemeName(k.Scheme())
	if err != nil {
		return nil, err
	}
	priv := k.Bytes()
	defer clear(priv)
	seed := k.Seed()
	defer clear(seed)

	f := toolkitKeyJSON{
		Scheme:     name,
		PrivateKey: hex.EncodeToString(priv),
		PublicKey:  hex.EncodeToString(k.Public().Bytes()),
		Address:    k.Public().Address().String(),
	}
	if seed != nil {
		f.Seed = hex.EncodeToString(seed)
	}
	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("pqc: encoding key file: %w", err)
	}
	return append(out, '\n'), nil
}

// ParseToolkitJSON parses an unencrypted key file in the format of java-tron's
// "Toolkit.jar pq-key new", following the node's loader rules (stricter
// where that is safe). Unknown fields are ignored; hex fields may carry a
// 0x or 0X prefix and use either case; a blank field counts as absent.
//
// At least one of seed and privateKey is required, and privateKey wins:
//
//   - FN-DSA-512: privateKey (1280-byte f||g||F) and publicKey (896 bytes)
//     are both required. seed is ignored entirely, because Falcon key
//     generation is not portable between implementations.
//   - ML-DSA-44: privateKey is the 2560-byte expanded key; a 32-byte seed
//     alone is enough. When both are given they must describe the same key,
//     and the returned key retains the seed.
//
// A publicKey or address that is given must match the key, else
// ErrKeyMismatch. Errors never contain key material.
func ParseToolkitJSON(data []byte) (PrivateKey, error) {
	f, err := decodeToolkitJSON(data)
	if err != nil {
		return nil, fmt.Errorf("pqc: parsing key file: %w", err)
	}
	scheme, err := schemeFromName(f.Scheme)
	if err != nil {
		return nil, err
	}
	hasSeed, hasPriv, hasPub := present(f.Seed), present(f.PrivateKey), present(f.PublicKey)
	if !hasSeed && !hasPriv {
		return nil, fmt.Errorf("%w: key file must define at least one of seed or privateKey", ErrInvalidKey)
	}

	var k PrivateKey
	switch scheme {
	case core.PQScheme_FN_DSA_512:
		k, err = toolkitFNDSA512(f, hasPriv, hasPub)
	default:
		k, err = toolkitMLDSA44(f, hasSeed, hasPriv, hasPub)
	}
	if err != nil {
		return nil, err
	}
	if present(f.Address) {
		if err := checkAddress(f.Address, k); err != nil {
			return nil, err
		}
	}
	return k, nil
}

func toolkitFNDSA512(f toolkitKeyJSON, hasPriv, hasPub bool) (PrivateKey, error) {
	if !hasPriv || !hasPub {
		return nil, fmt.Errorf("%w: an FN-DSA-512 key file needs privateKey and publicKey (Falcon seeds are not portable)",
			ErrInvalidKey)
	}
	priv, err := decodeKeyHex("privateKey", f.PrivateKey, FNDSA512PrivateKeySize)
	if err != nil {
		return nil, err
	}
	defer clear(priv)
	pub, err := decodeKeyHex("publicKey", f.PublicKey, FNDSA512PublicKeySize)
	if err != nil {
		return nil, err
	}
	return ParsePrivateKey(core.PQScheme_FN_DSA_512, priv, pub)
}

func toolkitMLDSA44(f toolkitKeyJSON, hasSeed, hasPriv, hasPub bool) (PrivateKey, error) {
	var pub []byte
	if hasPub {
		p, err := decodeKeyHex("publicKey", f.PublicKey, MLDSA44PublicKeySize)
		if err != nil {
			return nil, err
		}
		pub = p
	}

	var k PrivateKey
	if hasPriv {
		priv, err := decodeKeyHex("privateKey", f.PrivateKey, MLDSA44PrivateKeySize)
		if err != nil {
			return nil, err
		}
		defer clear(priv)
		if k, err = ParsePrivateKey(core.PQScheme_ML_DSA_44, priv, pub); err != nil {
			return nil, err
		}
	}
	if !hasSeed {
		return k, nil
	}

	seed, err := decodeKeyHex("seed", f.Seed, MLDSA44SeedSize)
	if err != nil {
		return nil, err
	}
	defer clear(seed)
	sk, err := ParsePrivateKey(core.PQScheme_ML_DSA_44, seed, pub)
	if err != nil {
		return nil, err
	}
	if k != nil && !sk.Public().Equal(k.Public()) {
		return nil, fmt.Errorf("%w: seed and privateKey describe different keys", ErrKeyMismatch)
	}
	return sk, nil
}

// present reports whether a key file field is set (not empty or blank).
func present(s string) bool { return strings.TrimSpace(s) != "" }

// decodeToolkitJSON reads only the exact, case-sensitive field names, as the
// node's Jackson loader does. encoding/json alone would also accept "SEED" or
// "PrivateKey", so a file could load differently here than on the node.
func decodeToolkitJSON(data []byte) (toolkitKeyJSON, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return toolkitKeyJSON{}, err
	}
	var f toolkitKeyJSON
	fields := map[string]*string{
		"scheme":     &f.Scheme,
		"seed":       &f.Seed,
		"privateKey": &f.PrivateKey,
		"publicKey":  &f.PublicKey,
		"address":    &f.Address,
	}
	for name, dst := range fields {
		v, ok := raw[name]
		if !ok {
			continue
		}
		if err := json.Unmarshal(v, dst); err != nil {
			return toolkitKeyJSON{}, fmt.Errorf("field %s: %w", name, err)
		}
	}
	return f, nil
}

// decodeKeyHex decodes a hex key file field of exactly size bytes, with an
// optional 0x/0X prefix. Errors name the field but never echo its content.
func decodeKeyHex(field, s string, size int) ([]byte, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	if len(s) != 2*size {
		return nil, fmt.Errorf("%w: %s must be %d hex characters, got %d", ErrInvalidKey, field, 2*size, len(s))
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %s is not valid hex", ErrInvalidKey, field)
	}
	return b, nil
}
