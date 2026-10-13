package client_test

import (
	"context"
	"errors"
	"math/big"
	"net"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// notConnectedAddress is a valid TRON address, so methods that validate their
// address arguments first still reach the connection check.
const notConnectedAddress = "TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"

// notConnectedSkips lists the exported methods the not-connected sweep does
// not call, each with the reason. TestNotConnectedSkipListIsCurrent fails when
// a name no longer exists, so the list cannot go stale.
var notConnectedSkips = map[string]string{
	"Start":                     "lifecycle: dials rather than performing an RPC",
	"Stop":                      "lifecycle",
	"Reconnect":                 "lifecycle",
	"SetTimeout":                "setter, no RPC",
	"SetAPIKey":                 "setter, no RPC",
	"SetContext":                "setter, no RPC",
	"SetPQCapabilitiesTTL":      "setter, no RPC",
	"InvalidatePQCapabilities":  "cache reset, no RPC",
	"Account":                   "service accessor, returns the client itself",
	"Contract":                  "service accessor, returns the client itself",
	"TRC20":                     "service accessor, returns the client itself",
	"Resource":                  "service accessor, returns the client itself",
	"Governance":                "service accessor, returns the client itself",
	"Transfers":                 "service accessor, returns the client itself",
	"Assets":                    "service accessor, returns the client itself",
	"Exchange":                  "service accessor, returns the client itself",
	"Block":                     "service accessor, returns the client itself",
	"Network":                   "service accessor, returns the client itself",
	"Pending":                   "service accessor, returns the client itself",
	"UpdateHash":                "recomputes a transaction hash locally, no RPC",
	"ParseTRC20NumericProperty": "pure parser, no RPC",
	"ParseTRC20StringProperty":  "pure parser, no RPC",
}

// validatesFirst lists the methods (matched with and without the Ctx suffix)
// whose string arguments are not addresses and so need a format the generic
// plausible argument does not satisfy. TestNotConnectedValidatedMethods calls
// them with well-formed arguments instead.
var validatesFirst = map[string]string{
	"AssetIssue":                "start and end times are checked against the clock",
	"DeployContract":            "bytecode must be hex",
	"EstimateEnergy":            "arguments must be a JSON parameter list",
	"EstimateEnergyWithData":    "token id must be numeric",
	"GetBlockByID":              "id must be hex",
	"GetTransactionByID":        "id must be hex",
	"GetTransactionFromPending": "id must be hex",
	"GetTransactionInfoByID":    "id must be hex",
	"IsTransactionPending":      "id must be hex",
	"TRC20Call":                 "data must be hex",
	"TriggerConstantContract":   "arguments must be a JSON parameter list",
	"TriggerContract":           "arguments must be a JSON parameter list",
	"TriggerContractWithData":   "token id must be numeric",
	"UpdateAccountPermission":   "owner permission is mandatory",
}

// exportedMethods returns the sorted names of the exported methods of
// *client.GrpcClient.
func exportedMethods() []string {
	typ := reflect.TypeOf(&client.GrpcClient{})
	names := make([]string, 0, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		names = append(names, typ.Method(i).Name)
	}
	sort.Strings(names)
	return names
}

// plausibleArg returns a non-zero argument of type t that passes the common
// input validation (addresses, positive amounts), or the zero value when no
// better one is known.
func plausibleArg(t reflect.Type) reflect.Value {
	switch t {
	case reflect.TypeOf((*context.Context)(nil)).Elem():
		return reflect.ValueOf(context.Background())
	case reflect.TypeOf((*big.Int)(nil)):
		return reflect.ValueOf(big.NewInt(1))
	}
	switch t.Kind() {
	case reflect.String:
		return reflect.ValueOf(notConnectedAddress).Convert(t)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(int64(1)).Convert(t)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflect.ValueOf(uint64(1)).Convert(t)
	}
	return reflect.Zero(t)
}

// zeroArg is like plausibleArg for the zero-value sweep: everything is the
// zero value except contexts, which must be non-nil.
func zeroArg(t reflect.Type) reflect.Value {
	if t == reflect.TypeOf((*context.Context)(nil)).Elem() {
		return reflect.ValueOf(context.Background())
	}
	return reflect.Zero(t)
}

// callMethod calls the named method on recv with args built by mk for the
// fixed (non-variadic) parameters. It returns the final error result, whether
// the method has one, and any panic value.
func callMethod(recv reflect.Value, name string, mk func(reflect.Type) reflect.Value) (panicked any, hasErr bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicked = r
		}
	}()
	m := recv.MethodByName(name)
	mt := m.Type()
	n := mt.NumIn()
	if mt.IsVariadic() {
		n--
	}
	args := make([]reflect.Value, n)
	for i := range args {
		args[i] = mk(mt.In(i))
	}
	out := m.Call(args)
	if len(out) == 0 {
		return nil, false, nil
	}
	last := out[len(out)-1]
	if !last.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, false, nil
	}
	if last.IsNil() {
		return nil, true, nil
	}
	e, ok := last.Interface().(error)
	if !ok {
		return nil, false, nil
	}
	return nil, true, e
}

// notConnectedReceivers builds the three states that must report
// ErrNotConnected: never started, started then stopped, and nil.
func notConnectedReceivers(t *testing.T) map[string]*client.GrpcClient {
	t.Helper()
	stopped := client.NewGrpcClient("passthrough:///unused")
	require.NoError(t, stopped.Start(grpc.WithTransportCredentials(insecure.NewCredentials())))
	require.NotNil(t, stopped.Client)
	stopped.Stop()
	return map[string]*client.GrpcClient{
		"never started": client.NewGrpcClient("passthrough:///unused"),
		"stopped":       stopped,
		"nil":           nil,
	}
}

func TestNotConnectedSkipListIsCurrent(t *testing.T) {
	have := map[string]bool{}
	for _, n := range exportedMethods() {
		have[n] = true
	}
	for name, reason := range notConnectedSkips {
		assert.True(t, have[name], "skipped method %q (%s) no longer exists; remove it from notConnectedSkips", name, reason)
	}
}

// TestNotConnectedNeverPanics calls every RPC method of GrpcClient on a client
// that is not connected and requires an error, not a panic. With zero-valued
// arguments a method may reject its input before it reaches the connection
// check; with plausible arguments every method must report ErrNotConnected.
func TestNotConnectedNeverPanics(t *testing.T) {
	methods := exportedMethods()
	passes := []struct {
		name        string
		mk          func(reflect.Type) reflect.Value
		wantNotConn bool
	}{
		{name: "zero args", mk: zeroArg},
		{name: "plausible args", mk: plausibleArg, wantNotConn: true},
	}

	for state, c := range notConnectedReceivers(t) {
		recv := reflect.ValueOf(c)
		for _, pass := range passes {
			var checked, notConnected, other int
			for _, name := range methods {
				if _, skip := notConnectedSkips[name]; skip {
					continue
				}
				panicked, hasErr, callErr := callMethod(recv, name, pass.mk)
				if panicked != nil {
					t.Errorf("%s / %s: %s panicked: %v", state, pass.name, name, panicked)
					continue
				}
				if !hasErr {
					t.Errorf("%s / %s: %s has no error result; add it to notConnectedSkips with a reason or give it one", state, pass.name, name)
					continue
				}
				checked++
				switch {
				case callErr == nil:
					t.Errorf("%s / %s: %s returned nil error", state, pass.name, name)
				case errors.Is(callErr, client.ErrNotConnected):
					notConnected++
				default:
					other++
					if _, ok := validatesFirst[strings.TrimSuffix(name, "Ctx")]; pass.wantNotConn && !ok {
						t.Errorf("%s / %s: %s = %v, want ErrNotConnected", state, pass.name, name, callErr)
					}
				}
			}
			t.Logf("%s / %s: %d methods, %d ErrNotConnected, %d other errors", state, pass.name, checked, notConnected, other)
			if !pass.wantNotConn {
				// Zero arguments are mostly invalid addresses, so many methods
				// report that instead; the ones with nothing to validate must
				// still reach the check.
				assert.NotZero(t, notConnected, "%s / %s: no method reached the connection check", state, pass.name)
			}
		}
	}
}

func TestNotConnectedValidatedMethodsListIsCurrent(t *testing.T) {
	have := map[string]bool{}
	for _, n := range exportedMethods() {
		have[n] = true
	}
	for name, reason := range validatesFirst {
		assert.True(t, have[name] && have[name+"Ctx"], "validatesFirst entry %q (%s) no longer exists", name, reason)
	}
}

// TestNotConnectedValidatedMethods calls the methods that check their string
// arguments first with well-formed ones, so they reach the connection check.
func TestNotConnectedValidatedMethods(t *testing.T) {
	const hexID = "0000000000000000000000000000000000000000000000000000000000000001"
	ctx := context.Background()
	calls := map[string]func(c *client.GrpcClient) error{
		"GetBlockByIDCtx": func(c *client.GrpcClient) error {
			_, err := c.GetBlockByIDCtx(ctx, hexID)
			return err
		},
		"GetTransactionByIDCtx": func(c *client.GrpcClient) error {
			_, err := c.GetTransactionByIDCtx(ctx, hexID)
			return err
		},
		"GetTransactionInfoByIDCtx": func(c *client.GrpcClient) error {
			_, err := c.GetTransactionInfoByIDCtx(ctx, hexID)
			return err
		},
		"GetTransactionFromPendingCtx": func(c *client.GrpcClient) error {
			_, err := c.GetTransactionFromPendingCtx(ctx, hexID)
			return err
		},
		"IsTransactionPendingCtx": func(c *client.GrpcClient) error {
			_, err := c.IsTransactionPendingCtx(ctx, hexID)
			return err
		},
		"TRC20CallCtx": func(c *client.GrpcClient) error {
			_, err := c.TRC20CallCtx(ctx, notConnectedAddress, notConnectedAddress, "06fdde03", true, 0)
			return err
		},
		"TriggerConstantContractCtx": func(c *client.GrpcClient) error {
			_, err := c.TriggerConstantContractCtx(ctx, notConnectedAddress, notConnectedAddress, "name()", "[]")
			return err
		},
		"TriggerContractCtx": func(c *client.GrpcClient) error {
			_, err := c.TriggerContractCtx(ctx, notConnectedAddress, notConnectedAddress, "name()", "[]", 1, 0, "", 0)
			return err
		},
		"TriggerContractWithDataCtx": func(c *client.GrpcClient) error {
			_, err := c.TriggerContractWithDataCtx(ctx, notConnectedAddress, notConnectedAddress, []byte{1}, 1, 0, "", 0)
			return err
		},
		"EstimateEnergyCtx": func(c *client.GrpcClient) error {
			_, err := c.EstimateEnergyCtx(ctx, notConnectedAddress, notConnectedAddress, "name()", "[]", 0, "", 0)
			return err
		},
		"EstimateEnergyWithDataCtx": func(c *client.GrpcClient) error {
			_, err := c.EstimateEnergyWithDataCtx(ctx, notConnectedAddress, notConnectedAddress, []byte{1}, 0, "", 0)
			return err
		},
		"DeployContractCtx": func(c *client.GrpcClient) error {
			_, err := c.DeployContractCtx(ctx, notConnectedAddress, "C", &core.SmartContract_ABI{}, "6080", 1, 100, 1)
			return err
		},
		"AssetIssueCtx": func(c *client.GrpcClient) error {
			start := time.Now().Add(time.Hour).UnixMilli()
			end := time.Now().Add(2 * time.Hour).UnixMilli()
			_, err := c.AssetIssueCtx(ctx, notConnectedAddress, "Token", "desc", "TOK", "https://example.com",
				6, 1000, start, end, 0, 0, 1, 1, 0, nil)
			return err
		},
		"UpdateAccountPermissionCtx": func(c *client.GrpcClient) error {
			owner := map[string]interface{}{
				"threshold": int64(1),
				"keys":      map[string]int64{notConnectedAddress: 1},
			}
			_, err := c.UpdateAccountPermissionCtx(ctx, notConnectedAddress, owner, nil, nil)
			return err
		},
	}
	// Every method the sweep lets fail its own validation must be called
	// here with arguments that reach the connection check, or a missing
	// guard in it would go unnoticed.
	for name := range validatesFirst {
		_, ok := calls[name+"Ctx"]
		assert.True(t, ok, "validatesFirst entry %q has no well-formed call in TestNotConnectedValidatedMethods", name)
	}
	for state, c := range notConnectedReceivers(t) {
		for name, call := range calls {
			t.Run(state+"/"+name, func(t *testing.T) {
				assert.ErrorIs(t, call(c), client.ErrNotConnected)
			})
		}
	}
}

// TestNotConnectedThroughServices covers the service accessors, which hand
// back the same client behind narrower interfaces.
func TestNotConnectedThroughServices(t *testing.T) {
	for state, c := range notConnectedReceivers(t) {
		t.Run(state, func(t *testing.T) {
			ctx := context.Background()
			_, err := c.Block().GetNowBlockCtx(ctx)
			assert.ErrorIs(t, err, client.ErrNotConnected)

			_, err = c.Account().GetAccountCtx(ctx, notConnectedAddress)
			assert.ErrorIs(t, err, client.ErrNotConnected)

			_, err = c.Network().GetNodeInfoCtx(ctx)
			assert.ErrorIs(t, err, client.ErrNotConnected)

			_, err = c.TRC20().TRC20GetNameCtx(ctx, notConnectedAddress)
			assert.ErrorIs(t, err, client.ErrNotConnected)
		})
	}
}

// TestNotConnectedProxyResolution covers the path that goes through
// callForAddress, which swallows RPC errors: a proxy lookup on a client that
// is not connected must not panic and must report the missing connection.
func TestNotConnectedProxyResolution(t *testing.T) {
	for state, c := range notConnectedReceivers(t) {
		t.Run(state, func(t *testing.T) {
			require.NotPanics(t, func() {
				_, err := c.GetContractABI(notConnectedAddress)
				assert.ErrorIs(t, err, client.ErrNotConnected)
			})
		})
	}
}

func TestNotConnectedPQCapabilities(t *testing.T) {
	for state, c := range notConnectedReceivers(t) {
		t.Run(state, func(t *testing.T) {
			_, err := c.GetPQCapabilities()
			assert.ErrorIs(t, err, client.ErrNotConnected)
			_, err = c.PQSchemeAllowed(0)
			assert.ErrorIs(t, err, client.ErrNotConnected)
		})
	}
}

// TestStopThenReconnect shows a stopped client reports ErrNotConnected and
// that Reconnect brings it back to a working state.
func TestStopThenReconnect(t *testing.T) {
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	mock := &mockWalletServer{
		GetNowBlock2Func: func(context.Context, *api.EmptyMessage) (*api.BlockExtention, error) {
			return &api.BlockExtention{}, nil
		},
	}
	api.RegisterWalletServer(srv, mock)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() {
		srv.GracefulStop()
		_ = lis.Close()
	})

	c := client.NewGrpcClient("passthrough:///bufconn")
	require.NoError(t, c.Start(
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	))
	t.Cleanup(c.Stop)

	_, err := c.GetNowBlock()
	require.NoError(t, err)

	c.Stop()
	_, err = c.GetNowBlock()
	require.ErrorIs(t, err, client.ErrNotConnected)
	// Stop leaves the exported Client in place: code that calls it directly
	// gets the closed connection's error, as before, not a nil panic.
	require.NotNil(t, c.Client)
	require.NotPanics(t, func() {
		_, err = c.Client.GetNowBlock2(context.Background(), new(api.EmptyMessage))
	})
	require.Error(t, err)

	require.NoError(t, c.Reconnect(""))
	_, err = c.GetNowBlock()
	require.NoError(t, err)

	// Stop is idempotent.
	c.Stop()
	c.Stop()
	_, err = c.GetNowBlock()
	assert.ErrorIs(t, err, client.ErrNotConnected)
}

// TestStopConcurrentWithRPC checks that Stop may run while RPCs are in flight:
// under -race this fails if Stop writes state the RPC methods read without
// synchronisation.
func TestStopConcurrentWithRPC(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{
		GetNowBlock2Func: func(context.Context, *api.EmptyMessage) (*api.BlockExtention, error) {
			return &api.BlockExtention{}, nil
		},
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			_, err := c.GetNowBlockCtx(ctx)
			cancel()
			if errors.Is(err, client.ErrNotConnected) {
				return
			}
		}
	}()
	time.Sleep(5 * time.Millisecond)
	c.Stop()
	<-done

	_, err := c.GetNowBlock()
	assert.ErrorIs(t, err, client.ErrNotConnected)
}

func TestStopNilReceiver(t *testing.T) {
	var c *client.GrpcClient
	assert.NotPanics(t, c.Stop)
}
