//go:build !falcon

package pqc

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFalconStub(t *testing.T) {
	require.ErrorIs(t, falcon.availErr(), ErrFalconUnavailable)

	k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512,
		mustHex(t, fnPairPrivKeyHex), mustHex(t, fnPairPubKeyHex))
	require.NoError(t, err)

	sig, err := k.Sign(mustHex(t, fnFixtureMessageHex))
	require.ErrorIs(t, err, ErrFalconUnavailable)
	assert.Nil(t, sig)

	ok, err := k.Public().Verify(mustHex(t, fnFixtureMessageHex), mustHex(t, fnFixtureSigHex))
	require.ErrorIs(t, err, ErrFalconUnavailable)
	assert.False(t, ok)

	gen, err := GenerateKey(core.PQScheme_FN_DSA_512, nil)
	require.ErrorIs(t, err, ErrFalconUnavailable)
	assert.Nil(t, gen)
}
