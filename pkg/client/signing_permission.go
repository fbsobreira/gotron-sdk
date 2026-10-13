package client

import (
	"bytes"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// SigningPermission returns a copy of the permission java-tron checks a
// transaction's signatures against, for its owner account acc and permission
// id: id 0 is the stored owner, or the implicit owner (acc's address, threshold
// 1); id 1 never signs; id 2 and up must be a stored Active permission. Errors
// wrap ErrNoSigningPermission, ErrAccountNotFound or ErrInvalidAddress.
func SigningPermission(acc *core.Account, permissionID int32) (*core.Permission, error) {
	if len(acc.GetAddress()) == 0 {
		return nil, fmt.Errorf("signing permission: id %d: %w", permissionID, ErrAccountNotFound)
	}
	switch {
	case permissionID < 0:
		return nil, fmt.Errorf("signing permission: id %d: negative id: %w", permissionID, ErrNoSigningPermission)
	case permissionID == 0:
		if op := acc.GetOwnerPermission(); op != nil {
			return proto.CloneOf(op), nil
		}
		addr := address.Address(acc.GetAddress())
		if !addr.IsValid() {
			return nil, fmt.Errorf("signing permission: id 0: no stored owner permission, and account address %x cannot stand in for it: %w",
				acc.GetAddress(), ErrInvalidAddress)
		}
		return &core.Permission{
			Type:           core.Permission_Owner,
			Id:             0,
			PermissionName: "owner",
			Threshold:      1,
			Keys:           []*core.Key{{Address: bytes.Clone(addr), Weight: 1}},
		}, nil
	case permissionID == 1:
		return nil, fmt.Errorf("signing permission: id 1: the witness permission cannot sign a transaction: %w", ErrNoSigningPermission)
	}
	for _, ap := range acc.GetActivePermission() {
		if ap.GetId() != permissionID {
			continue
		}
		if ap.GetType() != core.Permission_Active {
			return nil, fmt.Errorf("signing permission: id %d: stored permission has type %s, want Active: %w",
				permissionID, ap.GetType(), ErrNoSigningPermission)
		}
		return proto.CloneOf(ap), nil
	}
	return nil, fmt.Errorf("signing permission: id %d: no active permission with that id: %w", permissionID, ErrNoSigningPermission)
}
