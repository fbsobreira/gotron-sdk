package keystore

// ForPath return keystore from path
func ForPath(p string) *KeyStore {
	return NewKeyStore(p, StandardScryptN, StandardScryptP)
}

// ForPathLight returns a keystore using lightweight scrypt parameters.
// This is significantly faster than ForPath and intended for testing.
func ForPathLight(p string) *KeyStore {
	return NewKeyStore(p, LightScryptN, LightScryptP)
}
