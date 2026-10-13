package txcore

import (
	"context"
	"errors"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemeProber is a PQProber whose answer depends on the scheme asked about.
type schemeProber struct {
	allowed  map[core.PQScheme]bool
	probeErr error
	probed   []core.PQScheme
	// onProbe, if set, runs after each probe is recorded.
	onProbe func()
}

func (p *schemeProber) PQSchemeAllowedCtx(_ context.Context, scheme core.PQScheme) (bool, error) {
	p.probed = append(p.probed, scheme)
	if p.onProbe != nil {
		p.onProbe()
	}
	if p.probeErr != nil {
		return false, p.probeErr
	}
	return p.allowed[scheme], nil
}

func txWithPQSchemes(schemes ...core.PQScheme) *core.Transaction {
	tx := newDummyTx()
	for _, s := range schemes {
		tx.PqAuthSig = append(tx.PqAuthSig, &core.PQAuthSig{
			Scheme:    s,
			PublicKey: []byte("pk"),
			Signature: []byte("sig"),
		})
	}
	return tx
}

func TestCheckPQSchemes(t *testing.T) {
	fn := core.PQScheme_FN_DSA_512
	ml := core.PQScheme_ML_DSA_44
	probeErr := errors.New("node unreachable")

	tests := []struct {
		name       string
		tx         *core.Transaction
		prober     *schemeProber
		nilProber  bool
		wantProbed []core.PQScheme
		wantIs     error
		wantMsg    []string
	}{
		{
			name:   "no PQ signatures",
			tx:     txWithPQSchemes(),
			prober: &schemeProber{},
		},
		{
			name:      "no PQ signatures and nil prober",
			tx:        txWithPQSchemes(),
			nilProber: true,
		},
		{
			name:       "one scheme allowed",
			tx:         txWithPQSchemes(fn),
			prober:     &schemeProber{allowed: map[core.PQScheme]bool{fn: true}},
			wantProbed: []core.PQScheme{fn},
		},
		{
			name:       "same scheme twice is asked once",
			tx:         txWithPQSchemes(fn, fn),
			prober:     &schemeProber{allowed: map[core.PQScheme]bool{fn: true}},
			wantProbed: []core.PQScheme{fn},
		},
		{
			name:       "two schemes each asked once in first-seen order",
			tx:         txWithPQSchemes(ml, fn, ml, fn),
			prober:     &schemeProber{allowed: map[core.PQScheme]bool{fn: true, ml: true}},
			wantProbed: []core.PQScheme{ml, fn},
		},
		{
			name:       "one scheme disabled",
			tx:         txWithPQSchemes(ml),
			prober:     &schemeProber{allowed: map[core.PQScheme]bool{}},
			wantProbed: []core.PQScheme{ml},
			wantIs:     ErrPQSchemeDisabled,
			wantMsg:    []string{"ML_DSA_44"},
		},
		{
			name:       "second scheme disabled",
			tx:         txWithPQSchemes(fn, ml),
			prober:     &schemeProber{allowed: map[core.PQScheme]bool{fn: true}},
			wantProbed: []core.PQScheme{fn, ml},
			wantIs:     ErrPQSchemeDisabled,
			wantMsg:    []string{"ML_DSA_44"},
		},
		{
			name:       "probe error",
			tx:         txWithPQSchemes(fn),
			prober:     &schemeProber{probeErr: probeErr},
			wantProbed: []core.PQScheme{fn},
			wantIs:     probeErr,
			wantMsg:    []string{"checking post-quantum capability"},
		},
		{
			name:      "nil prober with PQ signatures",
			tx:        txWithPQSchemes(fn),
			nilProber: true,
			wantMsg:   []string{"prober"},
		},
		{
			name:    "nil transaction",
			tx:      nil,
			prober:  &schemeProber{},
			wantMsg: []string{"transaction"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p PQProber
			if !tt.nilProber {
				p = tt.prober
			}
			err := CheckPQSchemes(context.Background(), p, tt.tx)
			if tt.prober != nil {
				assert.Equal(t, tt.wantProbed, tt.prober.probed)
			}
			if tt.wantIs == nil && tt.wantMsg == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			if tt.wantIs != nil {
				assert.ErrorIs(t, err, tt.wantIs)
			}
			for _, s := range tt.wantMsg {
				assert.Contains(t, err.Error(), s)
			}
		})
	}
}

func TestCheckPQSchemes_ContextCancelled(t *testing.T) {
	tests := []struct {
		name          string
		cancelBefore  bool
		wantProbedLen int
	}{
		{name: "cancelled before the first probe", cancelBefore: true, wantProbedLen: 0},
		{name: "cancelled after the first probe", cancelBefore: false, wantProbedLen: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p := &schemeProber{allowed: map[core.PQScheme]bool{
				core.PQScheme_FN_DSA_512: true,
				core.PQScheme_ML_DSA_44:  true,
			}}
			if tt.cancelBefore {
				cancel()
			} else {
				p.onProbe = cancel
			}

			err := CheckPQSchemes(ctx, p, txWithPQSchemes(core.PQScheme_FN_DSA_512, core.PQScheme_ML_DSA_44))
			require.Error(t, err)
			assert.ErrorIs(t, err, context.Canceled)
			assert.Contains(t, err.Error(), "checking post-quantum capability")
			assert.Len(t, p.probed, tt.wantProbedLen)
		})
	}
}
