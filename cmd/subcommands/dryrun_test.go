package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	pkgsigner "github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
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
	// confirm makes GetTransactionInfoById return a confirmed info.
	confirm bool
}

func (w *dryRunWallet) CreateTransaction2(_ context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
	return fakeTxExtension(core.Transaction_Contract_TransferContract, in)
}

// TriggerContract builds an unsigned smart-contract call, as a node would
// for `trc20 send` and a non-constant `contract trigger`.
func (w *dryRunWallet) TriggerContract(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return fakeTxExtension(core.Transaction_Contract_TriggerSmartContract, in)
}

// fakeTxExtension wraps contract msg of type ctype in an unsigned
// transaction with fixed reference-block fields and a matching txID.
func fakeTxExtension(ctype core.Transaction_Contract_ContractType, msg proto.Message) (*api.TransactionExtention, error) {
	param, err := anypb.New(msg)
	if err != nil {
		return nil, err
	}
	raw := &core.TransactionRaw{
		Contract: []*core.Transaction_Contract{{
			Type:      ctype,
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

func (w *dryRunWallet) GetTransactionInfoById(_ context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) { //nolint:staticcheck,revive // matches proto-generated method name
	w.txInfoCalls.Add(1)
	if w.confirm {
		// A confirmed info without the inner resource Receipt.
		return &core.TransactionInfo{Id: in.GetValue(), BlockNumber: 1}, nil
	}
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

// signingCommandPaths lists every command that signs and broadcasts a
// transaction through executeOrDryRun. Only these accept --dry-run and
// --timeout.
var signingCommandPaths = [][]string{
	{"account", "activate"},
	{"account", "send"},
	{"account", "withdraw"},
	{"account", "freeze"},
	{"account", "freezeV2"},
	{"account", "unfreezeV2"},
	{"account", "vote"},
	{"account", "permission"},
	{"contract", "deploy"},
	{"contract", "trigger"},
	{"exchange", "create"},
	{"exchange", "inject"},
	{"exchange", "withdraw"},
	{"exchange", "trade"},
	{"proposal", "approve"},
	{"proposal", "withdraw"},
	{"proposal", "create"},
	{"sr", "create"},
	{"sr", "brokerage"},
	{"trc10", "issue"},
	{"trc10", "send"},
	{"trc10", "ico"},
	{"trc20", "send"},
}

// TestDryRunAndTimeoutFlagsScopedToSigningCommands guards the flag scope:
// signing commands must accept --dry-run and --timeout, and every other
// command must reject them. When they were persistent root flags,
// `keys remove x --dry-run` deleted the key and `upgrade --dry-run`
// replaced the binary, both reporting success.
func TestDryRunAndTimeoutFlagsScopedToSigningCommands(t *testing.T) {
	prevDry, prevTimeout := dryRun, timeout
	t.Cleanup(func() { dryRun, timeout = prevDry, prevTimeout })

	for _, name := range []string{"dry-run", "timeout"} {
		assert.Nil(t, RootCmd.PersistentFlags().Lookup(name), "--%s must not be a persistent root flag", name)
	}

	signing := map[string]bool{}
	for _, path := range signingCommandPaths {
		sub, _, err := RootCmd.Find(path)
		require.NoError(t, err)
		require.Equal(t, path[len(path)-1], sub.Name(), "%v is not registered", path)
		signing[sub.CommandPath()] = true
		for _, name := range []string{"dry-run", "timeout"} {
			assert.NotNil(t, sub.LocalFlags().Lookup(name), "%s must accept --%s", sub.CommandPath(), name)
		}
		assert.Equal(t, "false", sub.Flags().Lookup("dry-run").DefValue, "%s --dry-run default", sub.CommandPath())
		assert.Equal(t, strconv.FormatUint(uint64(config.Timeout), 10), sub.Flags().Lookup("timeout").DefValue,
			"%s --timeout default must come from the loaded config", sub.CommandPath())
	}

	// Every other command, including ones that delete keys or replace the
	// binary, must reject both flags.
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if !signing[c.CommandPath()] {
			for _, name := range []string{"dry-run", "timeout"} {
				assert.Nil(t, c.Flags().Lookup(name), "%s must not accept --%s", c.CommandPath(), name)
			}
		}
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(RootCmd)

	for _, path := range [][]string{{"keys", "remove"}, {"keys", "add"}, {"keys", "list"}, {"upgrade"}} {
		sub, _, err := RootCmd.Find(path)
		require.NoError(t, err)
		err = sub.ParseFlags([]string{"--dry-run"})
		require.Error(t, err, "%s must reject --dry-run", sub.CommandPath())
		assert.Contains(t, err.Error(), "unknown flag: --dry-run")
	}
}

// TestKeysRemoveDryRunIsRejected runs `tronctl keys remove <name> --dry-run`
// through the real command tree. It must fail with "unknown flag" and leave
// the account in place; flag parsing fails before any hook runs, so nothing
// is dialled or removed.
func TestKeysRemoveDryRunIsRejected(t *testing.T) {
	pqTestEnv(t, "pw")
	t.Setenv("TRONCTL_KS_DIR", "")
	addECDSAAccount(t, "dryrun-survivor", "pw")
	savePQAccount(t, "dryrun-pq-survivor", "pw")

	prevRT, prevConn, prevDry, prevNode := *rt, conn, dryRun, node
	t.Cleanup(func() {
		if rt.Conn != nil && rt.Conn != prevRT.Conn {
			rt.Conn.Stop()
		}
		*rt, conn, dryRun, node = prevRT, prevConn, prevDry, prevNode
		RootCmd.SetArgs(nil)
		RootCmd.SetOut(nil)
		RootCmd.SetErr(nil)
	})
	node = "127.0.0.1:50051" // never dialled: grpc.NewClient is lazy

	for _, name := range []string{"dryrun-survivor", "dryrun-pq-survivor"} {
		t.Run(name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			RootCmd.SetOut(&out)
			RootCmd.SetErr(&errOut)
			RootCmd.SetArgs([]string{"keys", "remove", name, "--dry-run"})
			err := RootCmd.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unknown flag: --dry-run")
		})
	}
	assert.True(t, store.DoesNamedAccountExist("dryrun-survivor"), "the ECDSA account must survive")
	_, err := store.PQAccountByName("dryrun-pq-survivor")
	assert.NoError(t, err, "the post-quantum account must survive")
}

// TestSigningCommandsDryRun runs two more executeOrDryRun command shapes,
// `trc20 send` and a non-constant `contract trigger`, with --dry-run parsed
// from the command line against the fake node. Each must print the signed
// smart-contract transaction as JSON and make no broadcast or confirmation
// call.
func TestSigningCommandsDryRun(t *testing.T) {
	const contractAddr = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	tests := []struct {
		name string
		path []string
		args []string
		// setup runs after flag parsing, for globals RunE reads.
		setup func()
	}{
		{
			name: "trc20 send",
			path: []string{"trc20", "send"},
			args: []string{dryRunTo, "1", contractAddr},
			// Skip the decimals() call, which the fake node does not serve.
			setup: func() { tokenDecimalsFlag = 6 },
		},
		{
			name: "contract trigger",
			path: []string{"contract", "trigger"},
			args: []string{contractAddr, "ping()"},
			setup: func() {
				tAmount, tTokenID, tTokenAmount, estimate = "0", "", "0", false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const pw = "dry-run-pass"
			pqTestEnv(t, pw)
			from := addECDSAAccount(t, "dry-sc", pw)
			wallet := &dryRunWallet{}
			withDryRunRuntime(t, newDryRunClient(t, wallet), from, false, false)

			prevDry, prevTimeout, prevDecimals := dryRun, timeout, tokenDecimalsFlag
			prevAmount, prevTokenID, prevTokenAmount, prevEstimate, prevFeeLimit := tAmount, tTokenID, tTokenAmount, estimate, feeLimit
			t.Cleanup(func() {
				dryRun, timeout, tokenDecimalsFlag = prevDry, prevTimeout, prevDecimals
				tAmount, tTokenID, tTokenAmount, estimate, feeLimit = prevAmount, prevTokenID, prevTokenAmount, prevEstimate, prevFeeLimit
			})

			sub, _, err := RootCmd.Find(tt.path)
			require.NoError(t, err)
			require.NoError(t, sub.ParseFlags([]string{"--dry-run", "--feeLimit", "1000000"}))
			t.Cleanup(func() {
				require.NoError(t, sub.Flags().Set("dry-run", "false"))
				sub.Flags().Lookup("dry-run").Changed = false
				sub.Flags().Lookup("feeLimit").Changed = false
				sub.SetOut(nil)
				sub.SetErr(nil)
			})
			require.True(t, dryRun, "--dry-run must set the bound global")
			rt.DryRun = dryRun // as PersistentPreRunE does
			tt.setup()

			out, errOut, err := runKeysCmd(t, sub, tt.args...)
			require.NoError(t, err)
			assert.Zero(t, wallet.broadcastCalls.Load(), "dry run must not broadcast")
			assert.Zero(t, wallet.txInfoCalls.Load(), "dry run must not wait for confirmation")

			tx, err := transaction.FromJSON([]byte(out))
			require.NoError(t, err, "stdout must be the signed transaction JSON:\n%s", out)
			require.Len(t, tx.GetSignature(), 1)
			assert.Len(t, tx.GetSignature()[0], 65)
			require.Len(t, tx.GetRawData().GetContract(), 1)
			assert.Equal(t, core.Transaction_Contract_TriggerSmartContract, tx.GetRawData().GetContract()[0].GetType())
			assert.Equal(t, int64(1000000), tx.GetRawData().GetFeeLimit(), "--feeLimit must reach the signed transaction")
			assert.Contains(t, errOut, "NOT broadcast")
		})
	}
}

// TestAccountSendConfirmedWithoutResourceReceipt runs a real (non-dry-run)
// `account send` whose confirmation returns a TransactionInfo with no inner
// resource Receipt, as a node may. The command reads receipt fields through
// the nil-safe getters, so it must print the result instead of panicking.
func TestAccountSendConfirmedWithoutResourceReceipt(t *testing.T) {
	const pw = "confirm-pass"
	pqTestEnv(t, pw)
	from := addECDSAAccount(t, "confirm-ecdsa", pw)
	wallet := &dryRunWallet{confirm: true}
	withDryRunRuntime(t, newDryRunClient(t, wallet), from, false, false)
	rt.NoWait, rt.Timeout = false, 1

	// The command prints its result to os.Stdout; the point here is that it
	// gets that far without a nil-pointer panic.
	_, _, err := runKeysCmd(t, accountSendCmd(), dryRunTo, "1")
	require.NoError(t, err)
	assert.Equal(t, int32(1), wallet.broadcastCalls.Load())
	assert.Positive(t, wallet.txInfoCalls.Load(), "the command must wait for confirmation")
}
