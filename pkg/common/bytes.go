package common

// ZeroBytes overwrites every element of b with zero.
// Use this to wipe sensitive key material from memory.
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
