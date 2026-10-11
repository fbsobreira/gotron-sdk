package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	pkgsigner "github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// dryRunWallet is a fake node: it builds transfer transactions, reports
// ML-DSA-44 as enabled and counts broadcast and confirmation calls.
type dryRunWallet struct {
	api.UnimplementedWalletServer
	broadcastCalls atomic.Int32
	txInfoCalls    atomic.Int32
}

func (w *dryRunWallet) CreateTransaction2(_ context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
	param, err := anypb.New(in)
	if err != nil {
		return nil, err
	}
	raw := &core.TransactionRaw{
		Contract: []*core.Transaction_Contract{{
			Type:      core.Transaction_Contract_TransferContract,
			Parameter: param,
		}},
		RefBlockBytes: []byte{0x01, 0x02},
		RefBlockHash:  []byte{0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a},
		Expiration:    1_700_000_060_000,
		Timestamp:     1_700_000_000_000,
	}
	rawBytes, err := proto.Marshal(raw)
	if err != nil {
		return nil, err
	}
	id := sha256.Sum256(rawBytes)
	return &api.TransactionExtention{
		Transaction: &core.Transaction{RawData: raw},
		Txid:        id[:],
		Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
	}, nil
}

func (w *dryRunWallet) BroadcastTransaction(context.Context, *core.Transaction) (*api.Return, error) {
	w.broadcastCalls.Add(1)
	return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
}

func (w *dryRunWallet) GetTransactionInfoById(context.Context, *api.BytesMessage) (*core.TransactionInfo, error) { //nolint:staticcheck,revive // matches proto-generated method name
	w.txInfoCalls.Add(1)
	return &core.TransactionInfo{}, nil
}

func (w *dryRunWallet) GetChainParameters(context.Context, *api.EmptyMessage) (*core.ChainParameters, error) {
	return &core.ChainParameters{
		ChainParameter: []*core.ChainParameters_ChainParameter{{Key: "getAllowMlDsa44", Value: 1}},
	}, nil
}

// newDryRunClient serves w over an in-memory connection.
func newDryRunClient(t *testing.T, w *dryRunWallet) *client.GrpcClient {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, w)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() {
		srv.GracefulStop()
		_ = lis.Close()
	})

	cc, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cc.Close() })

	g := client.NewGrpcClient("bufconn")
	g.Conn = cc
	g.Client = api.NewWalletClient(cc)
	return g
}

// withDryRunRuntime sets the CLI globals a signing command reads and
// restores them when the test ends.
func withDryRunRuntime(t *testing.T, g *client.GrpcClient, from string, dry, noPretty bool) {
	t.Helper()
	prevRT, prevConn, prevSigner, prevNoPretty := *rt, conn, signerAddress, noPrettyOutput
	t.Cleanup(func() {
		*rt, conn, signerAddress, noPrettyOutput = prevRT, prevConn, prevSigner, prevNoPretty
	})
	conn = g
	require.NoError(t, signerAddress.Set(from))
	rt.DryRun, rt.NoWait, rt.Timeout = dry, true, 0
	rt.NoPrettyOutput, noPrettyOutput = noPretty, noPretty
}

const dryRunTo = "TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"

// TestAccountSendDryRun runs the real `account send` command with --dry-run
// against a fake node, for an ECDSA and a post-quantum signer. Before the
// fix the command dereferenced the nil broadcast Result and panicked. It must
// print the signed transaction as JSON, report the txID on stderr, and make
// no broadcast or confirmation call.
func TestAccountSendDryRun(t *testing.T) {
	tests := []struct {
		name     string
		pq       bool
		noPretty bool
	}{
		{name: "ecdsa"},
		{name: "ecdsa no-pretty", noPretty: true},
		{name: "post-quantum", pq: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const pw = "dry-run-pass"
			pqTestEnv(t, pw)
			var from string
			if tt.pq {
				from = savePQAccount(t, "dry-pq", pw).Address
			} else {
				from = addECDSAAccount(t, "dry-ecdsa", pw)
			}
			wallet := &dryRunWallet{}
			withDryRunRuntime(t, newDryRunClient(t, wallet), from, true, tt.noPretty)

			out, errOut, err := runKeysCmd(t, accountSendCmd(), dryRunTo, "1")
			require.NoError(t, err)

			assert.Zero(t, wallet.broadcastCalls.Load(), "dry run must not broadcast")
			assert.Zero(t, wallet.txInfoCalls.Load(), "dry run must not wait for confirmation")

			if tt.noPretty {
				assert.Equal(t, 1, strings.Count(strings.TrimSpace(out), "\n")+1, "--no-pretty prints one JSON line")
			} else {
				assert.Contains(t, out, "\n  \"", "default output is indented")
			}

			tx, err := transaction.FromJSON([]byte(out))
			require.NoError(t, err, "stdout must be the signed transaction JSON:\n%s", out)
			if tt.pq {
				require.Len(t, tx.GetPqAuthSig(), 1)
				assert.Contains(t, out, `"pq_auth_sig"`)
			} else {
				require.Len(t, tx.GetSignature(), 1)
				assert.Len(t, tx.GetSignature()[0], 65)
				assert.Contains(t, out, `"signature"`)
			}

			var env struct {
				TxID string `json:"txID"`
			}
			require.NoError(t, json.Unmarshal([]byte(out), &env))
			require.NotEmpty(t, env.TxID)
			assert.Contains(t, errOut, "0x"+env.TxID)
			assert.Contains(t, errOut, "NOT broadcast")
		})
	}
}

// TestExecuteOrDryRunNilClient drives the helper over a Controller with no
// client at all: any broadcast or confirmation attempt would panic, so a pass
// proves the dry-run path never touches the network for an ECDSA signer.
func TestExecuteOrDryRunNilClient(t *testing.T) {
	prevRT, prevNoPretty := *rt, noPrettyOutput
	t.Cleanup(func() { *rt, noPrettyOutput = prevRT, prevNoPretty })
	rt.DryRun, rt.NoWait, rt.Timeout = true, false, 30
	noPrettyOutput = false

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	s, err := pkgsigner.NewPrivateKeySigner(key)
	require.NoError(t, err)
	tx := &core.Transaction{RawData: &core.TransactionRaw{
		Contract:  []*core.Transaction_Contract{{Type: core.Transaction_Contract_TransferContract, Parameter: &anypb.Any{}}},
		Timestamp: 1_700_000_000_000,
	}}
	ctrlr := transaction.NewController(nil, nil, nil, tx, transaction.WithSigner(s), opts)

	var out, errOut bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	var stop bool
	require.NotPanics(t, func() { stop, err = executeOrDryRun(cmd, ctrlr) })
	require.NoError(t, err)
	assert.True(t, stop)
	assert.Contains(t, out.String(), `"signature"`)
	signed, err := transaction.FromJSON(out.Bytes())
	require.NoError(t, err)
	require.Len(t, signed.GetSignature(), 1)
	hash, err := ctrlr.TransactionHash()
	require.NoError(t, err)
	assert.Contains(t, errOut.String(), hash)
}

// TestExecuteOrDryRunBroadcasts checks the helper leaves the normal path
// alone: without --dry-run it broadcasts, prints nothing itself, and tells the
// caller to go on and print the broadcast result.
func TestExecuteOrDryRunBroadcasts(t *testing.T) {
	const pw = "dry-run-pass"
	pqTestEnv(t, pw)
	from := addECDSAAccount(t, "live-ecdsa", pw)
	wallet := &dryRunWallet{}
	withDryRunRuntime(t, newDryRunClient(t, wallet), from, false, false)

	ext, err := conn.Transfer(from, dryRunTo, 1)
	require.NoError(t, err)
	ctrlr, err := newController(signerAddress, ext.Transaction)
	require.NoError(t, err)

	var out, errOut bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	stop, err := executeOrDryRun(cmd, ctrlr)
	require.NoError(t, err)
	assert.False(t, stop)
	assert.Empty(t, out.String())
	assert.Empty(t, errOut.String())
	assert.Equal(t, int32(1), wallet.broadcastCalls.Load())
	require.NotNil(t, ctrlr.Result)
	require.NotNil(t, ctrlr.Receipt)
}

// TestDryRunAndTimeoutFlagsArePersistent guards the root.go change: the
// flags must be inherited by subcommands, or `account send --dry-run` is
// rejected as an unknown flag.
func TestDryRunAndTimeoutFlagsArePersistent(t *testing.T) {
	sub, _, err := RootCmd.Find([]string{"account", "send"})
	require.NoError(t, err)
	require.Equal(t, "send", sub.Name())
	for _, name := range []string{"dry-run", "timeout"} {
		assert.NotNil(t, sub.InheritedFlags().Lookup(name), "account send must inherit --%s", name)
		assert.NotNil(t, RootCmd.PersistentFlags().Lookup(name), "--%s must be a persistent root flag", name)
	}
}
