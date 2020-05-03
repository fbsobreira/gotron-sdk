package keystore

// ForPath return keystore from path
func ForPath(p string) *KeyStore {
	return NewKeyStore(p, StandardScryptN, StandardScryptP)
}
