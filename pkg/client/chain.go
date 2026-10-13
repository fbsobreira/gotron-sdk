package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// Chain parameter keys that report TIP-899 post-quantum signature support.
// Nodes without TIP-899 do not return them at all.
const (
	chainParamAllowFnDsa512 = "getAllowFnDsa512"
	chainParamAllowMlDsa44  = "getAllowMlDsa44"
)

// DefaultPQCapabilitiesTTL is how long a GrpcClient reuses a successful
// TIP-899 capability probe before asking the node again. The scheme flags
// change only through a governance proposal, which takes effect at a
// maintenance interval hours apart, so a short TTL costs nothing in accuracy.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
const DefaultPQCapabilitiesTTL = time.Minute

// pqCapsCache holds the last successful TIP-899 capability probe of a
// GrpcClient. The zero value is ready to use with DefaultPQCapabilitiesTTL.
// Failed probes are never stored.
type pqCapsCache struct {
	mu        sync.Mutex
	caps      PQCapabilities
	fetchedAt time.Time
	valid     bool
	// gen is bumped by every invalidation, so a probe that was already in
	// flight cannot store its now stale result afterwards.
	gen    uint64
	ttl    time.Duration
	ttlSet bool
}

// ttlLocked returns the effective TTL; c.mu must be held.
func (c *pqCapsCache) ttlLocked() time.Duration {
	if !c.ttlSet {
		return DefaultPQCapabilitiesTTL
	}
	return c.ttl
}

// lookup returns the cached capabilities when they are still fresh, and the
// generation a new probe must present to store its result.
func (c *pqCapsCache) lookup() (caps PQCapabilities, ok bool, gen uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ttl := c.ttlLocked(); c.valid && ttl > 0 && time.Since(c.fetchedAt) < ttl {
		return c.caps, true, c.gen
	}
	return PQCapabilities{}, false, c.gen
}

// store records a successful probe started at fetchedAt, unless the cache was
// invalidated since lookup returned gen, caching is disabled, or a probe that
// started later is already stored.
func (c *pqCapsCache) store(caps PQCapabilities, fetchedAt time.Time, gen uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if gen != c.gen || c.ttlLocked() <= 0 {
		return
	}
	// Of two overlapping probes, keep the one that started last.
	if c.valid && !fetchedAt.After(c.fetchedAt) {
		return
	}
	c.caps = caps
	c.fetchedAt = fetchedAt
	c.valid = true
}

// invalidateLocked drops the cached probe; c.mu must be held.
func (c *pqCapsCache) invalidateLocked() {
	c.caps = PQCapabilities{}
	c.valid = false
	c.gen++
}

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
	w, err := g.wallet()
	if err != nil {
		return nil, err
	}
	params, err := w.GetChainParameters(ctx, new(api.EmptyMessage))
	if err != nil {
		return nil, fmt.Errorf("get chain parameters: %w", err)
	}
	return params, nil
}

// GetPQCapabilities queries the node for the TIP-899 post-quantum signature
// schemes it has enabled. A missing parameter or a value of 0 means the
// scheme is off, so nodes without TIP-899 report no capabilities.
//
// A successful result is cached on the client and reused for
// DefaultPQCapabilitiesTTL (see SetPQCapabilitiesTTL), so it may be up to
// that old; errors are never cached. Call InvalidatePQCapabilities to force
// the next call to ask the node.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (g *GrpcClient) GetPQCapabilities() (PQCapabilities, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetPQCapabilitiesCtx(ctx)
}

// GetPQCapabilitiesCtx is the context-aware version of GetPQCapabilities.
// It shares the client's capability cache: a fresh cached result is returned
// without an RPC, so it may be up to the configured TTL old. A cancelled or
// expired ctx is still reported as an error. Errors are never cached.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (g *GrpcClient) GetPQCapabilitiesCtx(ctx context.Context) (PQCapabilities, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return PQCapabilities{}, err
	}
	if _, err := g.wallet(); err != nil {
		return PQCapabilities{}, err
	}
	cached, ok, gen := g.pqCache.lookup()
	if ok {
		return cached, nil
	}
	// The lock is not held across network I/O, so concurrent misses may each
	// probe the node; an invalidation meanwhile changes gen and the result
	// is not stored.
	fetchedAt := time.Now()
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
	g.pqCache.store(caps, fetchedAt, gen)
	return caps, nil
}

// SetPQCapabilitiesTTL sets how long a successful TIP-899 capability probe is
// reused; the default is DefaultPQCapabilitiesTTL. A d of zero or less
// disables caching, so every GetPQCapabilities, PQSchemeAllowed and
// transaction.Controller post-quantum check asks the node. Any cached result
// is dropped. It is safe for concurrent use.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (g *GrpcClient) SetPQCapabilitiesTTL(d time.Duration) {
	if d < 0 {
		d = 0
	}
	g.pqCache.mu.Lock()
	defer g.pqCache.mu.Unlock()
	g.pqCache.ttl = d
	g.pqCache.ttlSet = true
	g.pqCache.invalidateLocked()
}

// InvalidatePQCapabilities drops the cached TIP-899 capability probe, so the
// next GetPQCapabilities or PQSchemeAllowed call asks the node. A probe
// already in flight when it is called does not repopulate the cache.
// Start, Stop and Reconnect call it. It is safe for concurrent use.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (g *GrpcClient) InvalidatePQCapabilities() {
	g.pqCache.mu.Lock()
	defer g.pqCache.mu.Unlock()
	g.pqCache.invalidateLocked()
}

// PQSchemeAllowed reports whether the node has enabled the given
// post-quantum signature scheme, using the client's configured timeout.
// transaction.Controller calls it before signing with a post-quantum signer.
// The answer comes from the capability cache described on GetPQCapabilities,
// so it may be up to the configured TTL old.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (g *GrpcClient) PQSchemeAllowed(scheme core.PQScheme) (bool, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.PQSchemeAllowedCtx(ctx, scheme)
}

// PQSchemeAllowedCtx reports whether the node has enabled the given
// post-quantum signature scheme. The txcore package probes for this method
// before signing with a post-quantum signer. The answer comes from the
// capability cache described on GetPQCapabilities, so it may be up to the
// configured TTL old.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (g *GrpcClient) PQSchemeAllowedCtx(ctx context.Context, scheme core.PQScheme) (bool, error) {
	caps, err := g.GetPQCapabilitiesCtx(ctx)
	if err != nil {
		return false, err
	}
	return caps.Allows(scheme), nil
}
