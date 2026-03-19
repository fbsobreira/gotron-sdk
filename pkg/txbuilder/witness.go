package txbuilder

import (
	"context"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

// VoteTx is a vote transaction builder with fluent methods for adding votes.
// It embeds *Tx so all terminal operations (Build, Send, SendAndConfirm) are
// available directly.
type VoteTx struct {
	*Tx
	votes map[string]int64
}

// WithMemo attaches a memo to this vote transaction.
func (v *VoteTx) WithMemo(memo string) *VoteTx {
	v.Tx.WithMemo(memo)
	return v
}

// WithPermissionID sets the permission ID for this vote transaction.
func (v *VoteTx) WithPermissionID(id int32) *VoteTx {
	v.Tx.WithPermissionID(id)
	return v
}

// Vote adds a vote for the given witness address. Can be called multiple times
// to build up the vote set. Returns itself for chaining.
func (v *VoteTx) Vote(witnessAddress string, count int64) *VoteTx {
	v.votes[witnessAddress] = count
	return v
}

// Votes sets all votes at once from a map. Useful for programmatic vote
// construction. Merges with any previously added votes.
func (v *VoteTx) Votes(votes map[string]int64) *VoteTx {
	for addr, count := range votes {
		v.votes[addr] = count
	}
	return v
}

// VoteWitness creates a witness vote transaction. Add votes using the
// fluent .Vote() or .Votes() methods on the returned VoteTx.
func (b *Builder) VoteWitness(from string, opts ...Option) *VoteTx {
	vt := &VoteTx{
		votes: make(map[string]int64),
	}
	vt.Tx = b.newTx(func(ctx context.Context) (*api.TransactionExtention, error) {
		return b.client.VoteWitnessAccountCtx(ctx, from, vt.votes)
	}, opts)
	return vt
}
