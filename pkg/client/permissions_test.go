package client_test

import (
	"bytes"
	"context"
	"sort"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// opsBitmap returns a 32-byte operations bitmap with the given bits set,
// using the layout makePermission writes: bit n is bitmap[n/8] & (1<<(n%8)).
func opsBitmap(bits ...int) []byte {
	b := make([]byte, 32)
	for _, n := range bits {
		b[n/8] |= 1 << (n % 8)
	}
	return b
}

// permAddr returns a valid 21-byte TRON address whose last byte is n.
func permAddr(n byte) []byte {
	a := make([]byte, address.AddressLength)
	a[0] = address.TronBytePrefix
	a[1] = 0xA0
	a[address.AddressLength-1] = n
	return a
}

func TestPermissionOperations(t *testing.T) {
	tests := []struct {
		name    string
		bitmap  []byte
		want    map[string]bool
		wantErr []string
	}{
		{name: "nil", bitmap: nil, want: map[string]bool{}},
		{name: "empty", bitmap: []byte{}, want: map[string]bool{}},
		{name: "all zero 32 bytes", bitmap: make([]byte, 32), want: map[string]bool{}},
		{
			name:   "transfer and trigger",
			bitmap: opsBitmap(1, 31),
			want:   map[string]bool{"TransferContract": true, "TriggerSmartContract": true},
		},
		{
			name:   "highest named type",
			bitmap: opsBitmap(59),
			want:   map[string]bool{"CancelAllUnfreezeV2Contract": true},
		},
		{
			name:   "short bitmap",
			bitmap: []byte{0x02},
			want:   map[string]bool{"TransferContract": true},
		},
		{
			name:    "unknown bit in a gap",
			bitmap:  opsBitmap(1, 47),
			wantErr: []string{"47"},
		},
		{
			name:    "every unknown bit is named",
			bitmap:  opsBitmap(1, 21, 47, 255),
			wantErr: []string{"21", "47", "255"},
		},
		{
			name:    "longer than 32 bytes",
			bitmap:  make([]byte, 33),
			wantErr: []string{"33"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.PermissionOperations(tt.bitmap)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Nil(t, got)
				for _, s := range tt.wantErr {
					assert.Contains(t, err.Error(), s)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}

// permAccount builds an account with an owner, a witness and two active
// permissions, shaped the way the node returns them.
func permAccount() *core.Account {
	return &core.Account{
		Address: permAddr(1),
		OwnerPermission: &core.Permission{
			Type:           core.Permission_Owner,
			Id:             0,
			PermissionName: "owner",
			Threshold:      2,
			Keys: []*core.Key{
				{Address: permAddr(1), Weight: 1},
				{Address: permAddr(2), Weight: 1},
				{Address: permAddr(3), Weight: 1},
			},
		},
		WitnessPermission: &core.Permission{
			Type:           core.Permission_Witness,
			Id:             1,
			PermissionName: "witness",
			Threshold:      1,
			Keys:           []*core.Key{{Address: permAddr(4), Weight: 1}},
		},
		ActivePermission: []*core.Permission{
			{
				Type:           core.Permission_Active,
				Id:             2,
				PermissionName: "transfers",
				Threshold:      1,
				Operations:     opsBitmap(1, 2, 31, 59),
				Keys: []*core.Key{
					{Address: permAddr(1), Weight: 1},
					{Address: permAddr(5), Weight: 2},
				},
			},
			{
				Type:           core.Permission_Active,
				Id:             3,
				PermissionName: "votes",
				Threshold:      3,
				Operations:     opsBitmap(4, 54, 55, 57, 58),
				Keys: []*core.Key{
					{Address: permAddr(6), Weight: 3},
				},
			},
		},
	}
}

// sortedPermission returns a copy of p with its keys in address order, so a
// permission rebuilt from a map (random key order) compares equal.
func sortedPermission(p *core.Permission) *core.Permission {
	if p == nil {
		return nil
	}
	c := proto.Clone(p).(*core.Permission)
	sort.Slice(c.Keys, func(i, j int) bool {
		return bytes.Compare(c.Keys[i].GetAddress(), c.Keys[j].GetAddress()) < 0
	})
	return c
}

func TestPermissionUpdateMaps_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*core.Account)
	}{
		{name: "owner witness and two actives", mutate: func(*core.Account) {}},
		{name: "no witness", mutate: func(a *core.Account) { a.WitnessPermission = nil }},
		{
			name: "active with every named type",
			mutate: func(a *core.Account) {
				var bits []int
				for v := range core.Transaction_Contract_ContractType_name {
					bits = append(bits, int(v))
				}
				a.ActivePermission[0].Operations = opsBitmap(bits...)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc := permAccount()
			tt.mutate(acc)

			var sent *core.AccountPermissionUpdateContract
			mock := &mockWalletServer{
				AccountPermissionUpdateFunc: func(_ context.Context, in *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
					sent = in
					return fakeTxExtention(), nil
				},
			}
			c := newMockClient(t, mock)

			owner, witness, actives, err := client.PermissionUpdateMaps(acc)
			require.NoError(t, err)
			require.NotNil(t, owner)
			if acc.WitnessPermission == nil {
				assert.Nil(t, witness)
			}
			assert.Len(t, actives, len(acc.ActivePermission))

			_, err = c.UpdateAccountPermissionCtx(context.Background(), accountAddress, owner, witness, actives)
			require.NoError(t, err)
			require.NotNil(t, sent)

			assert.True(t, proto.Equal(sortedPermission(acc.OwnerPermission), sortedPermission(sent.GetOwner())),
				"owner: want %v, got %v", acc.OwnerPermission, sent.GetOwner())
			assert.True(t, proto.Equal(sortedPermission(acc.WitnessPermission), sortedPermission(sent.GetWitness())),
				"witness: want %v, got %v", acc.WitnessPermission, sent.GetWitness())
			require.Len(t, sent.GetActives(), len(acc.ActivePermission))
			for i, want := range acc.ActivePermission {
				got := sent.GetActives()[i]
				assert.Equal(t, want.GetOperations(), got.GetOperations(), "active[%d] operations", i)
				assert.True(t, proto.Equal(sortedPermission(want), sortedPermission(got)),
					"active[%d]: want %v, got %v", i, want, got)
			}
		})
	}
}

func TestPermissionUpdateMaps_Shapes(t *testing.T) {
	owner, witness, actives, err := client.PermissionUpdateMaps(permAccount())
	require.NoError(t, err)

	assert.Equal(t, int64(2), owner["threshold"])
	ownerKeys, ok := owner["keys"].(map[string]int64)
	require.True(t, ok, "owner keys must be map[string]int64, got %T", owner["keys"])
	assert.Len(t, ownerKeys, 3)
	assert.Equal(t, int64(1), ownerKeys[address.Address(permAddr(1)).String()])

	assert.Equal(t, int64(1), witness["threshold"])
	_, ok = witness["keys"].(map[string]int64)
	assert.True(t, ok)

	require.Len(t, actives, 2)
	assert.Equal(t, "transfers", actives[0]["name"])
	assert.Equal(t, int64(1), actives[0]["threshold"])
	assert.Equal(t, map[string]bool{
		"TransferContract":            true,
		"TransferAssetContract":       true,
		"TriggerSmartContract":        true,
		"CancelAllUnfreezeV2Contract": true,
	}, actives[0]["operations"])
	assert.Equal(t, map[string]int64{
		address.Address(permAddr(1)).String(): 1,
		address.Address(permAddr(5)).String(): 2,
	}, actives[0]["keys"])
	assert.Equal(t, "votes", actives[1]["name"])
}

func TestPermissionUpdateMaps_Errors(t *testing.T) {
	badPrefix := permAddr(9)
	badPrefix[0] = 0x00

	tests := []struct {
		name    string
		acc     func() *core.Account
		wantErr []string
		wantIs  error
	}{
		{
			name:    "nil account",
			acc:     func() *core.Account { return nil },
			wantErr: []string{"nil account"},
		},
		{
			name: "nil owner",
			acc: func() *core.Account {
				a := permAccount()
				a.OwnerPermission = nil
				return a
			},
			wantErr: []string{"owner"},
			wantIs:  client.ErrNoStoredPermissions,
		},
		{
			name: "nil owner and no other permission",
			acc: func() *core.Account {
				return &core.Account{Address: permAddr(1)}
			},
			wantErr: []string{"owner"},
			wantIs:  client.ErrNoStoredPermissions,
		},
		{
			name: "no actives",
			acc: func() *core.Account {
				a := permAccount()
				a.ActivePermission = nil
				return a
			},
			wantErr: []string{"active"},
			wantIs:  client.ErrNoStoredPermissions,
		},
		{
			name: "empty actives",
			acc: func() *core.Account {
				a := permAccount()
				a.ActivePermission = []*core.Permission{}
				return a
			},
			wantErr: []string{"active"},
			wantIs:  client.ErrNoStoredPermissions,
		},
		{
			name: "duplicate owner key",
			acc: func() *core.Account {
				a := permAccount()
				a.OwnerPermission.Keys[2].Address = permAddr(1)
				return a
			},
			wantErr: []string{"owner", "duplicate"},
		},
		{
			name: "duplicate active key",
			acc: func() *core.Account {
				a := permAccount()
				a.ActivePermission[1].Keys = append(a.ActivePermission[1].Keys,
					&core.Key{Address: permAddr(6), Weight: 1})
				return a
			},
			wantErr: []string{"active[1]", "duplicate"},
		},
		{
			name: "short key address",
			acc: func() *core.Account {
				a := permAccount()
				a.WitnessPermission.Keys[0].Address = permAddr(4)[1:]
				return a
			},
			wantErr: []string{"witness", "invalid key address"},
		},
		{
			name: "key address without 0x41 prefix",
			acc: func() *core.Account {
				a := permAccount()
				a.ActivePermission[0].Keys[1].Address = badPrefix
				return a
			},
			wantErr: []string{"active[0]", "invalid key address"},
		},
		{
			name: "empty key address",
			acc: func() *core.Account {
				a := permAccount()
				a.OwnerPermission.Keys[0].Address = nil
				return a
			},
			wantErr: []string{"owner", "invalid key address"},
		},
		{
			name: "unknown operation bit",
			acc: func() *core.Account {
				a := permAccount()
				a.ActivePermission[1].Operations = opsBitmap(4, 47)
				return a
			},
			wantErr: []string{"active[1]", "47"},
		},
		{
			name: "active id out of order",
			acc: func() *core.Account {
				a := permAccount()
				a.ActivePermission[0].Id, a.ActivePermission[1].Id = 3, 2
				return a
			},
			wantErr: []string{"active[0]", "id"},
		},
		{
			name: "active with an all-zero operations bitmap",
			acc: func() *core.Account {
				a := permAccount()
				a.ActivePermission[1].Operations = make([]byte, 32)
				return a
			},
			wantErr: []string{"active[1]", "no operations"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, witness, actives, err := client.PermissionUpdateMaps(tt.acc())
			require.Error(t, err)
			assert.Nil(t, owner)
			assert.Nil(t, witness)
			assert.Nil(t, actives)
			for _, s := range tt.wantErr {
				assert.Contains(t, err.Error(), s)
			}
			if tt.wantIs != nil {
				assert.ErrorIs(t, err, tt.wantIs)
			} else {
				assert.NotErrorIs(t, err, client.ErrNoStoredPermissions)
			}
		})
	}
}

// UpdateAccountPermissionCtx always names the owner and witness permissions
// "owner" and "witness", so a custom name read back by PermissionUpdateMaps is
// reset on re-submit; everything else is kept.
func TestPermissionUpdateMaps_ResetsOwnerAndWitnessNames(t *testing.T) {
	acc := permAccount()
	acc.OwnerPermission.PermissionName = "custom-owner"
	acc.WitnessPermission.PermissionName = "custom-witness"

	var sent *core.AccountPermissionUpdateContract
	mock := &mockWalletServer{
		AccountPermissionUpdateFunc: func(_ context.Context, in *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
			sent = in
			return fakeTxExtention(), nil
		},
	}
	c := newMockClient(t, mock)

	owner, witness, actives, err := client.PermissionUpdateMaps(acc)
	require.NoError(t, err)
	_, err = c.UpdateAccountPermissionCtx(context.Background(), accountAddress, owner, witness, actives)
	require.NoError(t, err)
	require.NotNil(t, sent)

	assert.Equal(t, "owner", sent.GetOwner().GetPermissionName())
	assert.Equal(t, "witness", sent.GetWitness().GetPermissionName())

	wantOwner := sortedPermission(acc.OwnerPermission)
	wantOwner.PermissionName = "owner"
	assert.True(t, proto.Equal(wantOwner, sortedPermission(sent.GetOwner())),
		"owner: want %v, got %v", wantOwner, sent.GetOwner())
	wantWitness := sortedPermission(acc.WitnessPermission)
	wantWitness.PermissionName = "witness"
	assert.True(t, proto.Equal(wantWitness, sortedPermission(sent.GetWitness())),
		"witness: want %v, got %v", wantWitness, sent.GetWitness())
}

// nileDefaultActiveOps is the default active operations bitmap Nile gives a
// freshly created account (7fff1fc0033efb0f, padded to 32 bytes), as read on
// 2026-10-13.
func nileDefaultActiveOps() []byte {
	b := make([]byte, 32)
	copy(b, []byte{0x7f, 0xff, 0x1f, 0xc0, 0x03, 0x3e, 0xfb, 0x0f})
	return b
}

func TestPermissionAllows(t *testing.T) {
	active := func(ops []byte) *core.Permission {
		return &core.Permission{Type: core.Permission_Active, Id: 2, Threshold: 1, Operations: ops}
	}
	tests := []struct {
		name string
		perm *core.Permission
		ct   core.Transaction_Contract_ContractType
		want bool
	}{
		{name: "owner, transfer", perm: &core.Permission{Type: core.Permission_Owner}, ct: core.Transaction_Contract_TransferContract, want: true},
		{name: "owner, permission update", perm: &core.Permission{Type: core.Permission_Owner}, ct: core.Transaction_Contract_AccountPermissionUpdateContract, want: true},
		{name: "owner, operations ignored", perm: &core.Permission{Type: core.Permission_Owner, Operations: []byte{0x01}}, ct: core.Transaction_Contract_TriggerSmartContract, want: true},
		{name: "active with the bit", perm: active(opsBitmap(1, 31)), ct: core.Transaction_Contract_TriggerSmartContract, want: true},
		{name: "active without the bit", perm: active(opsBitmap(1, 31)), ct: core.Transaction_Contract_TransferAssetContract, want: false},
		{name: "active, no operations", perm: active(nil), ct: core.Transaction_Contract_TransferContract, want: false},
		{name: "active, unnameable bit elsewhere", perm: active(opsBitmap(1, 47, 255)), ct: core.Transaction_Contract_TransferContract, want: true},
		{name: "active, 31-byte bitmap", perm: active(opsBitmap(1)[:31]), ct: core.Transaction_Contract_TransferContract, want: false},
		{name: "active, 33-byte bitmap", perm: active(append(opsBitmap(1), 0)), ct: core.Transaction_Contract_TransferContract, want: false},
		{name: "active, highest bit", perm: active(opsBitmap(255)), ct: core.Transaction_Contract_ContractType(255), want: true},
		{name: "witness", perm: &core.Permission{Type: core.Permission_Witness, Id: 1, Operations: opsBitmap(1)}, ct: core.Transaction_Contract_TransferContract, want: false},
		{name: "unknown permission type", perm: &core.Permission{Type: core.Permission_PermissionType(7), Operations: opsBitmap(1)}, ct: core.Transaction_Contract_TransferContract, want: false},
		{name: "nil permission", perm: nil, ct: core.Transaction_Contract_TransferContract, want: false},
		{name: "contract type -1, active", perm: active(opsBitmap(0, 255)), ct: -1, want: false},
		{name: "contract type 256, active", perm: active(opsBitmap(0, 255)), ct: 256, want: false},
		{name: "contract type -1, owner", perm: &core.Permission{Type: core.Permission_Owner}, ct: -1, want: false},
		{name: "contract type 256, owner", perm: &core.Permission{Type: core.Permission_Owner}, ct: 256, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, client.PermissionAllows(tt.perm, tt.ct))
		})
	}
}

// TestPermissionAllows_NileDefaultActive checks PermissionAllows against the
// default active permission Nile gives new accounts.
func TestPermissionAllows_NileDefaultActive(t *testing.T) {
	ops := nileDefaultActiveOps()
	perm := &core.Permission{Type: core.Permission_Active, Id: 2, Threshold: 1, Operations: ops}
	tests := []struct {
		ct   core.Transaction_Contract_ContractType
		want bool
	}{
		{ct: core.Transaction_Contract_TransferContract, want: true},
		{ct: core.Transaction_Contract_TriggerSmartContract, want: true},
		{ct: core.Transaction_Contract_FreezeBalanceV2Contract, want: true},
		{ct: core.Transaction_Contract_AccountPermissionUpdateContract, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.ct.String(), func(t *testing.T) {
			// Read the bit straight off the bitmap, so the expectation is
			// checked against the data and not only against PermissionAllows.
			n := int(tt.ct)
			require.Equal(t, tt.want, ops[n/8]&(1<<(n%8)) != 0, "bitmap bit %d", n)
			assert.Equal(t, tt.want, client.PermissionAllows(perm, tt.ct))
		})
	}
}
