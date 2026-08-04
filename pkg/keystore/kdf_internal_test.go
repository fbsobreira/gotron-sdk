package keystore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKdfInt_IntAndInt64(t *testing.T) {
	params := map[string]interface{}{
		"n":  int(4096),
		"r":  int64(8),
		"p":  float64(1),
		"lo": 0,
	}
	n, err := kdfInt(params, "n", 2, maxScryptN)
	require.NoError(t, err)
	require.Equal(t, 4096, n)

	r, err := kdfInt(params, "r", 1, maxScryptR)
	require.NoError(t, err)
	require.Equal(t, 8, r)

	p, err := kdfInt(params, "p", 1, maxScryptP)
	require.NoError(t, err)
	require.Equal(t, 1, p)

	_, err = kdfInt(params, "lo", 1, 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be in")
}
