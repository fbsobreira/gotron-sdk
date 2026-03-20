package keystore

// ForPath returns a KeyStore backed by the directory at p using standard scrypt parameters.
func ForPath(p string) *KeyStore {
	return NewKeyStore(p, StandardScryptN, StandardScryptP)
}

// ForPathLight returns a keystore using lightweight scrypt parameters.
// This is significantly faster than ForPath but provides weaker key
// derivation. It should only be used in tests; production code should
// use [ForPath] or [NewKeyStore] with [StandardScryptN]/[StandardScryptP].
func ForPathLight(p string) *KeyStore {
	return NewKeyStore(p, LightScryptN, LightScryptP)
}
