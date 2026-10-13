package client_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestSigningPermission(t *testing.T) {
	malformed := permAccount()
	malformed.ActivePermission[0].Type = core.Permission_Witness

	noOwner := permAccount()
	noOwner.OwnerPermission = nil

	tests := []struct {
		name    string
		acc     *core.Account
		id      int32
		want    *core.Permission
		wantIs  error
		wantMsg []string
	}{
		{name: "stored owner", acc: permAccount(), id: 0, want: permAccount().GetOwnerPermission()},
		{name: "stored active id 2", acc: permAccount(), id: 2, want: permAccount().GetActivePermission()[0]},
		{name: "stored active id 3", acc: permAccount(), id: 3, want: permAccount().GetActivePermission()[1]},
		{
			name: "synthetic owner",
			acc:  noOwner,
			id:   0,
			want: &core.Permission{
				Type:           core.Permission_Owner,
				Id:             0,
				PermissionName: "owner",
				Threshold:      1,
				Keys:           []*core.Key{{Address: permAddr(1), Weight: 1}},
			},
		},
		{name: "witness id 1, stored", acc: permAccount(), id: 1, wantIs: client.ErrNoSigningPermission, wantMsg: []string{"signing permission:", "id 1"}},
		{name: "witness id 1, none stored", acc: &core.Account{Address: permAddr(1)}, id: 1, wantIs: client.ErrNoSigningPermission, wantMsg: []string{"id 1"}},
		{name: "missing active id 4", acc: permAccount(), id: 4, wantIs: client.ErrNoSigningPermission, wantMsg: []string{"signing permission:", "id 4"}},
		{name: "no active stored, id 2 not synthesized", acc: &core.Account{Address: permAddr(1)}, id: 2, wantIs: client.ErrNoSigningPermission, wantMsg: []string{"id 2"}},
		{name: "active id 2 of type witness", acc: malformed, id: 2, wantIs: client.ErrNoSigningPermission, wantMsg: []string{"id 2", "Witness"}},
		{name: "negative id", acc: permAccount(), id: -1, wantIs: client.ErrNoSigningPermission, wantMsg: []string{"id -1"}},
		{name: "nil account", acc: nil, id: 0, wantIs: client.ErrAccountNotFound, wantMsg: []string{"signing permission:", "id 0"}},
		{name: "empty account (getaccount for a missing address)", acc: &core.Account{}, id: 0, wantIs: client.ErrAccountNotFound},
		{name: "nil account, active id", acc: nil, id: 2, wantIs: client.ErrAccountNotFound, wantMsg: []string{"id 2"}},
		{name: "invalid address, no stored owner", acc: &core.Account{Address: []byte{0x42, 0x01}}, id: 0, wantIs: client.ErrInvalidAddress, wantMsg: []string{"id 0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.SigningPermission(tt.acc, tt.id)
			if tt.wantIs != nil || tt.wantMsg != nil {
				require.Error(t, err)
				assert.Nil(t, got)
				if tt.wantIs != nil {
					assert.ErrorIs(t, err, tt.wantIs)
				}
				for _, s := range tt.wantMsg {
					assert.Contains(t, err.Error(), s)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.True(t, proto.Equal(tt.want, got), "got %v, want %v", got, tt.want)
		})
	}
}

// TestSigningPermission_ReturnsClone checks that changing the returned
// permission, stored or synthesized, leaves the account untouched.
func TestSigningPermission_ReturnsClone(t *testing.T) {
	noOwner := permAccount()
	noOwner.OwnerPermission = nil

	tests := []struct {
		name   string
		acc    *core.Account
		id     int32
		stored func(*core.Account) *core.Permission
	}{
		{name: "stored owner", acc: permAccount(), id: 0, stored: (*core.Account).GetOwnerPermission},
		{name: "stored active", acc: permAccount(), id: 2, stored: func(a *core.Account) *core.Permission { return a.GetActivePermission()[0] }},
		{name: "synthetic owner", acc: noOwner, id: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := proto.CloneOf(tt.acc)
			got, err := client.SigningPermission(tt.acc, tt.id)
			require.NoError(t, err)
			require.NotNil(t, got)
			if tt.stored != nil {
				assert.NotSame(t, tt.stored(tt.acc), got)
			}
			require.NotEmpty(t, got.GetKeys())

			got.Threshold = 99
			got.PermissionName = "changed"
			got.Keys[0].Weight = 42
			got.Keys[0].Address[len(got.Keys[0].Address)-1] ^= 0xff
			got.Keys = append(got.Keys, &core.Key{Address: permAddr(9), Weight: 1})
			if len(got.Operations) > 0 {
				got.Operations[0] ^= 0xff
			}

			assert.True(t, proto.Equal(before, tt.acc), "account changed through the returned permission")
		})
	}
}

// TestSigningPermission_WithPermissionAllows checks the intended use: the
// permission SigningPermission returns, fed to PermissionAllows, gives the
// node's verdict for a contract type.
func TestSigningPermission_WithPermissionAllows(t *testing.T) {
	acc := permAccount()

	owner, err := client.SigningPermission(acc, 0)
	require.NoError(t, err)
	assert.True(t, client.PermissionAllows(owner, core.Transaction_Contract_AccountPermissionUpdateContract))

	active, err := client.SigningPermission(acc, 2)
	require.NoError(t, err)
	assert.True(t, client.PermissionAllows(active, core.Transaction_Contract_TriggerSmartContract))
	assert.False(t, client.PermissionAllows(active, core.Transaction_Contract_AccountPermissionUpdateContract))
}
