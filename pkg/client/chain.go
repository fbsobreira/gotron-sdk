package client

import (
	"context"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// Chain parameter keys that report TIP-899 post-quantum signature support.
// Nodes without TIP-899 do not return them at all.
const (
	chainParamAllowFnDsa512 = "getAllowFnDsa512"
	chainParamAllowMlDsa44  = "getAllowMlDsa44"
)

// PQCapabilities reports which TIP-899 post-quantum signature schemes the
// connected node has enabled.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
type PQCapabilities struct {
	FNDSA512 bool
	MLDSA44  bool
}

// Allows reports whether the given scheme is enabled. An unknown or
// unrecognised scheme is never allowed.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (c PQCapabilities) Allows(scheme core.PQScheme) bool {
	switch scheme {
	case core.PQScheme_FN_DSA_512:
		return c.FNDSA512
	case core.PQScheme_ML_DSA_44:
		return c.MLDSA44
	default:
		return false
	}
}

// GetChainParameters returns the dynamic chain parameters of the connected node.
func (g *GrpcClient) GetChainParameters() (*core.ChainParameters, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetChainParametersCtx(ctx)
}

// GetChainParametersCtx is the context-aware version of GetChainParameters.
func (g *GrpcClient) GetChainParametersCtx(ctx context.Context) (*core.ChainParameters, error) {
	ctx = g.withAPIKey(ctx)
	params, err := g.Client.GetChainParameters(ctx, new(api.EmptyMessage))
	if err != nil {
		return nil, fmt.Errorf("get chain parameters: %w", err)
	}
	return params, nil
}

// GetPQCapabilities queries the node for the TIP-899 post-quantum signature
// schemes it has enabled. A missing parameter or a value of 0 means the
// scheme is off, so nodes without TIP-899 report no capabilities.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (g *GrpcClient) GetPQCapabilities() (PQCapabilities, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetPQCapabilitiesCtx(ctx)
}

// GetPQCapabilitiesCtx is the context-aware version of GetPQCapabilities.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (g *GrpcClient) GetPQCapabilitiesCtx(ctx context.Context) (PQCapabilities, error) {
	params, err := g.GetChainParametersCtx(ctx)
	if err != nil {
		return PQCapabilities{}, err
	}

	var caps PQCapabilities
	for _, p := range params.GetChainParameter() {
		switch p.GetKey() {
		case chainParamAllowFnDsa512:
			caps.FNDSA512 = p.GetValue() != 0
		case chainParamAllowMlDsa44:
			caps.MLDSA44 = p.GetValue() != 0
		}
	}
	return caps, nil
}

// PQSchemeAllowedCtx reports whether the node has enabled the given
// post-quantum signature scheme. The txcore package probes for this method
// before signing with a post-quantum signer.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (g *GrpcClient) PQSchemeAllowedCtx(ctx context.Context, scheme core.PQScheme) (bool, error) {
	caps, err := g.GetPQCapabilitiesCtx(ctx)
	if err != nil {
		return false, err
	}
	return caps.Allows(scheme), nil
}
