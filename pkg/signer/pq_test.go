package signer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func newPQTestTx() *core.Transaction {
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0x01, 0x02},
			RefBlockHash:  []byte{0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a},
			Expiration:    1_700_000_060_000,
			Timestamp:     1_700_000_000_000,
		},
	}
}

func txID(t *testing.T, tx *core.Transaction) []byte {
	t.Helper()
	raw, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err)
	h := sha256.Sum256(raw)
	return h[:]
}

func newMLDSAKey(t *testing.T) pqc.PrivateKey {
	t.Helper()
	k, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, nil)
	require.NoError(t, err)
	return k
}

// fnFixtureKey parses the tron-grpc FN-DSA-512 key pair, which needs no
// Falcon backend to parse.
func fnFixtureKey(t *testing.T) pqc.PrivateKey {
	t.Helper()
	sk, err := hex.DecodeString(fnFixturePrivKeyHex)
	require.NoError(t, err)
	pk, err := hex.DecodeString(fnFixturePubKeyHex)
	require.NoError(t, err)
	k, err := pqc.ParsePrivateKey(core.PQScheme_FN_DSA_512, sk, pk)
	require.NoError(t, err)
	return k
}

// requirePQSigVerifies checks that entry was made by key over the txid of tx.
func requirePQSigVerifies(t *testing.T, tx *core.Transaction, entry *core.PQAuthSig, key pqc.PrivateKey) {
	t.Helper()
	require.Equal(t, key.Scheme(), entry.GetScheme())
	require.Equal(t, key.Public().Bytes(), entry.GetPublicKey())
	pub, err := pqc.ParsePublicKey(entry.GetScheme(), entry.GetPublicKey())
	require.NoError(t, err)
	ok, err := pub.Verify(txID(t, tx), entry.GetSignature())
	require.NoError(t, err)
	require.True(t, ok, "PQ signature does not verify against SHA-256(raw_data)")
}

func TestSignTransactionPQ_MLDSA44(t *testing.T) {
	key := newMLDSAKey(t)
	tx := newPQTestTx()
	before := txID(t, tx)

	signed, err := SignTransactionPQ(tx, key)
	require.NoError(t, err)

	assert.Same(t, tx, signed)
	assert.Empty(t, signed.GetSignature(), "ECDSA signatures must not be touched")
	require.Len(t, signed.GetPqAuthSig(), 1)
	assert.Len(t, signed.GetPqAuthSig()[0].GetSignature(), pqc.MLDSA44SignatureSize)
	assert.Equal(t, before, txID(t, signed), "txid changed by signing")
	requirePQSigVerifies(t, signed, signed.GetPqAuthSig()[0], key)
}

func TestSignTransactionPQ_Hybrid(t *testing.T) {
	ecKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	ec, err := NewPrivateKeySigner(ecKey)
	require.NoError(t, err)
	pq, err := NewPQSigner(newMLDSAKey(t))
	require.NoError(t, err)

	tx := newPQTestTx()
	before := txID(t, tx)
	tx, err = ec.Sign(tx)
	require.NoError(t, err)
	require.Len(t, tx.GetSignature(), 1)
	ecSig := bytes.Clone(tx.GetSignature()[0])

	tx, err = pq.Sign(tx)
	require.NoError(t, err)

	require.Len(t, tx.GetSignature(), 1)
	assert.Equal(t, ecSig, tx.GetSignature()[0], "ECDSA signature changed")
	require.Len(t, tx.GetPqAuthSig(), 1)
	assert.Equal(t, before, txID(t, tx))
	requirePQSigVerifies(t, tx, tx.GetPqAuthSig()[0], pq.(*pqSigner).key)
}

func TestSignTransactionPQ_TwoKeys(t *testing.T) {
	k1, k2 := newMLDSAKey(t), newMLDSAKey(t)
	tx := newPQTestTx()

	_, err := SignTransactionPQ(tx, k1)
	require.NoError(t, err)
	_, err = SignTransactionPQ(tx, k2)
	require.NoError(t, err)

	require.Len(t, tx.GetPqAuthSig(), 2)
	requirePQSigVerifies(t, tx, tx.GetPqAuthSig()[0], k1)
	requirePQSigVerifies(t, tx, tx.GetPqAuthSig()[1], k2)
}

func TestSignTransactionPQ_SameKeyTwiceRefused(t *testing.T) {
	key := newMLDSAKey(t)
	tx := newPQTestTx()
	_, err := SignTransactionPQ(tx, key)
	require.NoError(t, err)
	snapshot := proto.Clone(tx)

	signed, err := SignTransactionPQ(tx, key)
	require.ErrorIs(t, err, ErrPQAlreadySigned)
	assert.Nil(t, signed)
	assert.True(t, proto.Equal(snapshot, tx), "refused signing modified the transaction")
}

func TestSignTransactionPQ_SelfVerifyFailureRefused(t *testing.T) {
	errBackend := errors.New("backend fault")
	tests := []struct {
		name   string
		ok     bool
		err    error
		wantIs error
	}{
		{name: "verify returns false", ok: false},
		{name: "verify returns error", ok: false, err: errBackend, wantIs: errBackend},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := verifyPQ
			t.Cleanup(func() { verifyPQ = orig })
			verifyPQ = func(*pqc.PublicKey, []byte, []byte) (bool, error) { return tt.ok, tt.err }

			tx := newPQTestTx()
			snapshot := proto.Clone(tx)
			signed, err := SignTransactionPQ(tx, newMLDSAKey(t))
			require.Error(t, err)
			if tt.wantIs != nil {
				require.ErrorIs(t, err, tt.wantIs)
			}
			assert.Nil(t, signed)
			assert.True(t, proto.Equal(snapshot, tx), "unverified signature was attached")
		})
	}
}

func TestSignTransactionPQ_SignatureCap(t *testing.T) {
	require.Equal(t, 5, MaxTotalSignatures)

	tests := []struct {
		name    string
		ecdsa   int
		pq      int
		refused bool
	}{
		{"empty", 0, 0, false},
		{"4 ECDSA", 4, 0, false},
		{"4 PQ", 0, 4, false},
		{"2 ECDSA + 2 PQ", 2, 2, false},
		{"5 ECDSA", 5, 0, true},
		{"5 PQ", 0, 5, true},
		{"3 ECDSA + 2 PQ", 3, 2, true},
		{"6 ECDSA", 6, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx := newPQTestTx()
			for i := 0; i < tc.ecdsa; i++ {
				tx.Signature = append(tx.Signature, bytes.Repeat([]byte{byte(i + 1)}, 65))
			}
			for i := 0; i < tc.pq; i++ {
				tx.PqAuthSig = append(tx.PqAuthSig, &core.PQAuthSig{
					Scheme:    core.PQScheme_ML_DSA_44,
					PublicKey: bytes.Repeat([]byte{byte(i + 1)}, pqc.MLDSA44PublicKeySize),
					Signature: []byte{0x01},
				})
			}
			snapshot := proto.Clone(tx)

			signed, err := SignTransactionPQ(tx, newMLDSAKey(t))
			if tc.refused {
				require.ErrorIs(t, err, ErrTooManySignatures)
				assert.Nil(t, signed)
				assert.True(t, proto.Equal(snapshot, tx), "refused signing modified the transaction")
				return
			}
			require.NoError(t, err)
			assert.Len(t, tx.GetSignature(), tc.ecdsa)
			assert.Len(t, tx.GetPqAuthSig(), tc.pq+1)
		})
	}
}

func TestSignTransactionPQ_NilInputs(t *testing.T) {
	key := newMLDSAKey(t)
	tests := []struct {
		name string
		tx   *core.Transaction
		key  pqc.PrivateKey
	}{
		{"nil tx", nil, key},
		{"nil raw_data", &core.Transaction{}, key},
		{"nil key", newPQTestTx(), nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var snapshot proto.Message
			if tc.tx != nil {
				snapshot = proto.Clone(tc.tx)
			}
			signed, err := SignTransactionPQ(tc.tx, tc.key)
			require.Error(t, err)
			assert.Nil(t, signed)
			if tc.tx != nil {
				assert.True(t, proto.Equal(snapshot, tc.tx))
			}
		})
	}
}

func TestSignTransactionPQ_DestroyedKey(t *testing.T) {
	key := newMLDSAKey(t)
	key.Destroy()
	tx := newPQTestTx()

	signed, err := SignTransactionPQ(tx, key)
	require.ErrorIs(t, err, pqc.ErrKeyDestroyed)
	assert.Nil(t, signed)
	assert.Empty(t, tx.GetPqAuthSig())

	s, err := NewPQSigner(newMLDSAKey(t))
	require.NoError(t, err)
	s.(*pqSigner).key.Destroy()
	_, err = s.Sign(newPQTestTx())
	require.ErrorIs(t, err, pqc.ErrKeyDestroyed)
}

func TestSignTransactionPQ_FNDSA512WithoutBackend(t *testing.T) {
	if pqc.SchemeAvailable(core.PQScheme_FN_DSA_512) == nil {
		t.Skip("Falcon backend compiled in; covered by the falcon-tagged tests")
	}
	key := fnFixtureKey(t)

	_, err := NewPQSigner(key)
	require.ErrorIs(t, err, pqc.ErrFalconUnavailable)

	tx := newPQTestTx()
	signed, err := SignTransactionPQ(tx, key)
	require.ErrorIs(t, err, pqc.ErrFalconUnavailable)
	assert.Nil(t, signed)
	assert.Empty(t, tx.GetPqAuthSig())
}

func TestNewPQSigner(t *testing.T) {
	t.Run("nil key", func(t *testing.T) {
		s, err := NewPQSigner(nil)
		require.Error(t, err)
		assert.Nil(t, s)
	})

	t.Run("ML-DSA-44", func(t *testing.T) {
		key := newMLDSAKey(t)
		s, err := NewPQSigner(key)
		require.NoError(t, err)

		want := key.Public().Address()
		addr := s.Address()
		assert.Equal(t, want, addr)
		addr[1] ^= 0xff
		assert.Equal(t, want, s.Address(), "Address must return a copy")

		schemed, ok := s.(interface{ PQScheme() core.PQScheme })
		require.True(t, ok, "PQ signer must expose PQScheme()")
		assert.Equal(t, core.PQScheme_ML_DSA_44, schemed.PQScheme())

		tx, err := s.Sign(newPQTestTx())
		require.NoError(t, err)
		require.Len(t, tx.GetPqAuthSig(), 1)
		requirePQSigVerifies(t, tx, tx.GetPqAuthSig()[0], key)
	})

	t.Run("ECDSA signer has no PQScheme", func(t *testing.T) {
		ecKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		s, err := NewPrivateKeySigner(ecKey)
		require.NoError(t, err)
		_, ok := s.(interface{ PQScheme() core.PQScheme })
		assert.False(t, ok)
	})
}

// FN-DSA-512 key pair from tronsave/tron-grpc, the same as fnPairPrivKeyHex and
// fnPairPubKeyHex in pkg/pqc: the canonical 1280-byte f||g||F and the bare
// 896-byte h.
const (
	fnFixturePrivKeyHex = "07c0bee0a040f81087fbd0c7f02101f82f3c1bef03f82e40fba101f421bef460850c21ff0c4f43f09e4223b000dbbfbeebd042f3bf040bff7f17e043ffe080f42185dfd0beec1fc01781450fe04003b1430fc13f20404318313c000ffb07cdfbec1fc900313efbef7bfc103d03f0bff7ddfd0c7f82e3de3e245f80e7df85185fc3fc3fb80c61c0fc8f7d23c00007c0bf040f851c8081100000e0507ef7cd83e00fbfe8317c078082f81042204139083f01f00083d431411bbffff48000ffcf8003c177f3e0ff03bd01f840c4ebf03c1bbfff0000bb0411ff03df80081002fc1002041f3e07ce3e13c0c61c413e0870c203cf81042041e3b03b0c0081f830bbfbff7e23f0030470481060ba1010bfdc117c27bf4417e1bef01fffffbfbb0c6e82146ebe13d1c4081082f3a07d00510707e001080f02ffdf83f01f010ff0befc8ec2ebe038f4703e10207d17cf02f00100f0304604213edfe0c4f7f081ffd000fbf009000c830fff7ffbbe4507d085ec1e850c803d184fc2ffeebefbf1441be03d0c904418ad80043003ffaf49fbce7f0fe08203a002076eff04007ff3ff060b9efb002001005fbb045ffc13f23c0020790c0f401bf000e7ffbe0c4e010c30c51390420bdec4ffaf421fdfbfeff03ff3b044f7ff81d3bd860420b60fff7f0800010fd086fbc140efd141183084fbb0420c113fec2042e7e0451c00b8f4308214003d07e141086fc0ec313bebef7914003d0fae840be077f8213c1c3ec0fff085006f800000bedc5f4127f0fe1860c6ec21450010be204ec1f43fbfd8410603dec903d14207d041140f3bfc30fbefef3c041ebd0bffc1fbe03a07cffb07cfc5f80f43e48fc2102e7ce3d008f7d13e18718103ff01f3e081e81ffe101f7cf060c813ffbfebeec2f3b040f45007fbf0f8f81d8404317fec4082fbc000f370bafcdf860bef3f040f7a2042f907ff41ebc1460bb0bb1bf040fbf13e1b8003f41f02f75f3bf85fb90fb08503d13c13ff82040080e81f83001fbb0c303ef3e140f43fbe07b005d41004fff0c704003ef7cfc6f7c03ff021bb080006f3af431bdfc113e179340615e4f10c02120f0114f1b436fbfa04d30c07f402ead4e7e3072019ccdf11171df30df41c09fbfa06fe001020f8e525f904041aa30c0d1f0604da0be0051930ecf012effa1110f3ecfe1f171ef7dcd7eb011408fd06edc92e0929bdfdfe0812fbee16bf0e35f2e0ef10091813ff05f60ae9ecf3fd03ecf218f120f5110dfc050217f90a063bd7e9dc0ced2415f8e9ffd90e26d806e4ff2f1cec10f125320cf010e8dd1507eff7d3f01f144427d90f06171a0cd9da18241022fdb8e70323f203e6e0eb2810fd28ddbe0bf4d8eafceaed131811e0d502e6dc09090aeb0d21fefeeb1af004140af8d6e9ea08e1fc09f9f611f627f1e3fe20f4010b0fff1bf2e40413eaf9e823dcf8001a1d36eaefeff112130c26c4dff2141906d7e0081def1e0bbcfd02d92ff5cc160b0a34ec15dcf20dd3f6090001fff9140415f8fbe20c1920e30b1f0325f80f2916ebd7e821341102ef06f024fc16ce15e9f712dd0e0cd2fce9ec0d021ccbea061a0420f70312d11720d1d306e812f0fe0c02dd2b3a02fe01e11913f01f230e3bf91413eddd13e21225ec0402e5dd04fa0df3ead51ff603f203d414fd0a04e7060afef9ec1a12bfe327f7dbfc20ebe7f5f9f717d6fd22ee2103f91a111a1ef0030818ef0c0c02fe27240900ec0d0807e9ddecfefbf60bddf508c3f70d06d7fa1e1a042019bcd512e01eebf4e71810fc131ccd0228111705"
	fnFixturePubKeyHex  = "62a91c087cc08a7968d830d4ea17662566f606055a39103741a3569b0155291b0f9dfb1bce8bc6806085961d17f7dd9aa6661abfa3e2dc1d8114be2a75ecdf5a4ea7fb6e28e6101877788cca2b275577066c86ab6b72927409db743f91a681e7ee73a7fe80cbf70e9b6e4ce57b671098794140c44f5daba6523671e34b84655affd44adad72a62a2c0fedd4abc411eb569629a4e66789025c2049c5829d67923f44b202f76aeac2120fc59f9b048769d9ad6657baa76b9f919e09f0897a620b4a66ecf7879d894aa3fa1d7987c20d5eb57715c10ab060bcb71a1d1074ba503af0dae73c4519c5f60d6d1429e2fb62534888673da9ab0363945348f2614d6e859bbaa6334693eb0b867e073ba37395c53a655557daa8058b8427d55c5e78301b586b1d80f45e478c27e41cad4d86768b598c2a1c9877c14e8e39c56b84982a62869531206a9a4f8036e9e20a716efc93640fa1e675b857f3d9193954449060b3ca5a07053e10571ef0b324a9ca5d6a2c9a58c1fbe54cb1aae039b6b4d88d14ca2fe82c8f0514ee3443b86caf98167ff6a947c47f8e96eafb19015aa632cb4691c660be4a3bf5a48814b84042632292a2399f5bc5da367c881886c3d43175915325542b6d25229269631ca964b160206a915648a00954c695ae095992c7d553fb1c3af77715c02754e191d4f911ae6ba6f435b2d09836b658b4fb25ef28ae5290aa946e72adb084ca44130ac6e7a0970e384d7c573aca8ec539be26aa175e43236d2a736e8f986faea0692088149eba99b99ac7d659e00261e70f6432ca3855cb0b44639d4de3cae31c9cfede2a93175486f596e04b1c19b54dc78523d1685d465497b9113d91548b6a21c85d1c2a79c1bc01fc13356cf1cc8763a8519b7b8d83b0546ec2e8c64c2a36f1c4898b089f19e8b2d5ee8279a30562149927001d05aef40b41f2451829c127e2a528588c31dea3334ce685e9f5082e275d6d0ace08405e046fe7a48e8d4d427b4a609c4ba620ed6a108d6787396b69020430a775674795a42e36e504956e8729b550bba8084b7ba16acd08524fdc84618e274f1e8eb57d5d03225d24bc4e411425ad4c2db25ea7ebd2ab21082b150b5546699cef64034c9db4beef74391bfd451ccce7078ba52b4007e91e877437d98620a6a6f70ebc1ec3e48be42101412114024c891c66b1b0dc714f48b7b6f09ebc8101d4e4838f1901f8237a242d600bc17f985490b4baf5a8741d92a415fab33a0de4c9025de00"
)
