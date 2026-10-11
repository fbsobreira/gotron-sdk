package client_test

import (
	"context"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/txcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func chainParams(kv ...any) *core.ChainParameters {
	cp := &core.ChainParameters{}
	for i := 0; i+1 < len(kv); i += 2 {
		cp.ChainParameter = append(cp.ChainParameter, &core.ChainParameters_ChainParameter{
			Key:   kv[i].(string),
			Value: kv[i+1].(int64),
		})
	}
	return cp
}

func chainParamsMock(cp *core.ChainParameters) *mockWalletServer {
	return &mockWalletServer{
		GetChainParametersFunc: func(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
			return cp, nil
		},
	}
}

func TestGetChainParameters(t *testing.T) {
	c := newMockClient(t, chainParamsMock(chainParams("getMaintenanceTimeInterval", int64(21600000))))
	cp, err := c.GetChainParameters()
	require.NoError(t, err)
	require.Len(t, cp.GetChainParameter(), 1)
	assert.Equal(t, "getMaintenanceTimeInterval", cp.GetChainParameter()[0].GetKey())
	assert.Equal(t, int64(21600000), cp.GetChainParameter()[0].GetValue())
}

func TestGetPQCapabilities(t *testing.T) {
	tests := []struct {
		name string
		cp   *core.ChainParameters
		want client.PQCapabilities
	}{
		{
			name: "both on",
			cp:   chainParams("getAllowFnDsa512", int64(1), "getAllowMlDsa44", int64(1)),
			want: client.PQCapabilities{FNDSA512: true, MLDSA44: true},
		},
		{
			name: "FN-DSA-512 only (Nile today)",
			cp:   chainParams("getAllowFnDsa512", int64(1), "getAllowMlDsa44", int64(0)),
			want: client.PQCapabilities{FNDSA512: true},
		},
		{
			name: "ML-DSA-44 only",
			cp:   chainParams("getAllowMlDsa44", int64(1)),
			want: client.PQCapabilities{MLDSA44: true},
		},
		{
			name: "both explicitly off",
			cp:   chainParams("getAllowFnDsa512", int64(0), "getAllowMlDsa44", int64(0)),
			want: client.PQCapabilities{},
		},
		{
			name: "keys absent (node without TIP-899)",
			cp:   chainParams("getMaintenanceTimeInterval", int64(21600000)),
			want: client.PQCapabilities{},
		},
		{
			name: "empty parameters",
			cp:   &core.ChainParameters{},
			want: client.PQCapabilities{},
		},
		{
			name: "unrelated keys with similar names are ignored",
			cp:   chainParams("getAllowFnDsa5120", int64(1), "allowMlDsa44", int64(1), "getAllowFnDsa512 ", int64(1)),
			want: client.PQCapabilities{},
		},
		{
			name: "any non-zero value means on",
			cp:   chainParams("getAllowFnDsa512", int64(7), "getAllowMlDsa44", int64(-1)),
			want: client.PQCapabilities{FNDSA512: true, MLDSA44: true},
		},
		{
			name: "nil entries are skipped",
			cp: &core.ChainParameters{ChainParameter: []*core.ChainParameters_ChainParameter{
				nil,
				{Key: "getAllowFnDsa512", Value: 1},
			}},
			want: client.PQCapabilities{FNDSA512: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newMockClient(t, chainParamsMock(tt.cp))
			got, err := c.GetPQCapabilities()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPQCapabilities_NilResponse(t *testing.T) {
	c := newMockClient(t, chainParamsMock(nil))
	got, err := c.GetPQCapabilitiesCtx(context.Background())
	require.NoError(t, err)
	assert.Equal(t, client.PQCapabilities{}, got)
}

func TestGetPQCapabilities_Error(t *testing.T) {
	mock := &mockWalletServer{
		GetChainParametersFunc: func(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
			return nil, status.Error(codes.Unavailable, "node down")
		},
	}
	c := newMockClient(t, mock)

	got, err := c.GetPQCapabilitiesCtx(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "node down")
	assert.Equal(t, client.PQCapabilities{}, got)

	allowed, err := c.PQSchemeAllowedCtx(context.Background(), core.PQScheme_FN_DSA_512)
	require.Error(t, err)
	assert.False(t, allowed)

	_, err = c.GetChainParameters()
	require.Error(t, err)
}

func TestPQCapabilities_Allows(t *testing.T) {
	tests := []struct {
		name   string
		caps   client.PQCapabilities
		scheme core.PQScheme
		want   bool
	}{
		{"fn on, fn asked", client.PQCapabilities{FNDSA512: true}, core.PQScheme_FN_DSA_512, true},
		{"fn on, ml asked", client.PQCapabilities{FNDSA512: true}, core.PQScheme_ML_DSA_44, false},
		{"ml on, ml asked", client.PQCapabilities{MLDSA44: true}, core.PQScheme_ML_DSA_44, true},
		{"ml on, fn asked", client.PQCapabilities{MLDSA44: true}, core.PQScheme_FN_DSA_512, false},
		{"none on, fn asked", client.PQCapabilities{}, core.PQScheme_FN_DSA_512, false},
		{"none on, ml asked", client.PQCapabilities{}, core.PQScheme_ML_DSA_44, false},
		{"unknown scheme, all on", client.PQCapabilities{FNDSA512: true, MLDSA44: true}, core.PQScheme_UNKNOWN_PQ_SCHEME, false},
		{"unrecognised scheme, all on", client.PQCapabilities{FNDSA512: true, MLDSA44: true}, core.PQScheme(99), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.caps.Allows(tt.scheme))
		})
	}
}

func TestPQSchemeAllowedCtx(t *testing.T) {
	c := newMockClient(t, chainParamsMock(chainParams("getAllowFnDsa512", int64(1), "getAllowMlDsa44", int64(0))))

	allowed, err := c.PQSchemeAllowedCtx(context.Background(), core.PQScheme_FN_DSA_512)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = c.PQSchemeAllowedCtx(context.Background(), core.PQScheme_ML_DSA_44)
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = c.PQSchemeAllowedCtx(context.Background(), core.PQScheme_UNKNOWN_PQ_SCHEME)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestGetCanDelegatedMaxSizePQ_CarriesScheme(t *testing.T) {
	var got *api.CanDelegatedMaxSizeRequestMessage
	mock := &mockWalletServer{
		GetCanDelegatedMaxSizeFunc: func(_ context.Context, in *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
			got = in
			return &api.CanDelegatedMaxSizeResponseMessage{MaxSize: 42}, nil
		},
	}
	c := newMockClient(t, mock)

	resp, err := c.GetCanDelegatedMaxSizePQ(accountAddress, int32(core.ResourceCode_BANDWIDTH), core.PQScheme_FN_DSA_512)
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.GetMaxSize())
	require.NotNil(t, got)
	assert.Equal(t, core.PQScheme_FN_DSA_512, got.GetPqScheme())
	assert.Equal(t, int32(core.ResourceCode_BANDWIDTH), got.GetType())
	assert.NotEmpty(t, got.GetOwnerAddress())

	// The existing entry points keep sending no scheme.
	_, err = c.GetCanDelegatedMaxSize(accountAddress, int32(core.ResourceCode_ENERGY))
	require.NoError(t, err)
	assert.Equal(t, core.PQScheme_UNKNOWN_PQ_SCHEME, got.GetPqScheme())

	_, err = c.GetCanDelegatedMaxSizePQCtx(context.Background(), accountAddress, int32(core.ResourceCode_BANDWIDTH), core.PQScheme_ML_DSA_44)
	require.NoError(t, err)
	assert.Equal(t, core.PQScheme_ML_DSA_44, got.GetPqScheme())
}

func TestGetCanDelegatedMaxSizePQ_WrapsGRPCError(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{
		GetCanDelegatedMaxSizeFunc: func(_ context.Context, _ *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
			return nil, status.Error(codes.Unavailable, "node down")
		},
	})
	_, err := c.GetCanDelegatedMaxSizePQ(accountAddress, int32(core.ResourceCode_BANDWIDTH), core.PQScheme_FN_DSA_512)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get can-delegated max size")
	assert.Equal(t, codes.Unavailable, status.Code(err), "the gRPC status must survive wrapping")
}

func TestGetCanDelegatedMaxSizePQ_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.GetCanDelegatedMaxSizePQ("not-an-address", int32(core.ResourceCode_BANDWIDTH), core.PQScheme_FN_DSA_512)
	require.Error(t, err)
}

// txcore.Send probes the broadcaster for txcore.PQProber; if GrpcClient's
// method drifted, the capability check would silently stop running.
var _ txcore.PQProber = (*client.GrpcClient)(nil)

func TestPQSchemeAllowed(t *testing.T) {
	tests := []struct {
		name   string
		cp     *core.ChainParameters
		scheme core.PQScheme
		want   bool
	}{
		{"ML-DSA-44 enabled", chainParams("getAllowMlDsa44", int64(1)), core.PQScheme_ML_DSA_44, true},
		{"ML-DSA-44 disabled", chainParams("getAllowMlDsa44", int64(0)), core.PQScheme_ML_DSA_44, false},
		{"FN-DSA-512 enabled", chainParams("getAllowFnDsa512", int64(1)), core.PQScheme_FN_DSA_512, true},
		{"parameter missing", chainParams("getMaintenanceTimeInterval", int64(21600000)), core.PQScheme_ML_DSA_44, false},
		{"unknown scheme", chainParams("getAllowMlDsa44", int64(1)), core.PQScheme_UNKNOWN_PQ_SCHEME, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newMockClient(t, chainParamsMock(tt.cp))
			allowed, err := c.PQSchemeAllowed(tt.scheme)
			require.NoError(t, err)
			assert.Equal(t, tt.want, allowed)
		})
	}
}

func TestPQSchemeAllowed_Error(t *testing.T) {
	mock := &mockWalletServer{
		GetChainParametersFunc: func(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
			return nil, status.Error(codes.Unavailable, "node down")
		},
	}
	c := newMockClient(t, mock)

	allowed, err := c.PQSchemeAllowed(core.PQScheme_ML_DSA_44)
	require.Error(t, err)
	assert.False(t, allowed)
	assert.Contains(t, err.Error(), "node down")
}
