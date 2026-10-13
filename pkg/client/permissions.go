package client

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// permissionOperationsLen is the size of a permission's operations bitmap:
// 256 bits, one per contract type.
const permissionOperationsLen = 32

// PermissionOperations decodes a permission's operations bitmap into the
// contract-type names UpdateAccountPermissionCtx takes in an active
// permission's "operations" map. Bit n (bitmap[n/8] & (1<<(n%8))) stands for
// contract type n.
//
// An empty or nil bitmap gives an empty, non-nil map. It fails closed: a set
// bit with no name in this SDK's core.Transaction_Contract_ContractType_name
// is an error that lists every such bit, because skipping it would drop that
// operation from the permission on the next update. A bitmap longer than 32
// bytes is also an error.
func PermissionOperations(bitmap []byte) (map[string]bool, error) {
	if len(bitmap) > permissionOperationsLen {
		return nil, fmt.Errorf("operations bitmap is %d bytes, longer than %d", len(bitmap), permissionOperationsLen)
	}
	ops := make(map[string]bool)
	var unknown []string
	for i, b := range bitmap {
		for j := 0; j < 8; j++ {
			if b&(1<<j) == 0 {
				continue
			}
			n := i*8 + j
			name, ok := core.Transaction_Contract_ContractType_name[int32(n)]
			if !ok {
				unknown = append(unknown, strconv.Itoa(n))
				continue
			}
			ops[name] = true
		}
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("operations bitmap sets bits with no contract type known to this SDK: %s",
			strings.Join(unknown, ", "))
	}
	return ops, nil
}

// PermissionUpdateMaps reads acc's current permissions into the maps
// UpdateAccountPermissionCtx takes, so a caller can add or change a key and
// pass them back without losing a key, a weight, a threshold or an operation.
// An account update replaces every permission, so anything left out is gone.
//
// owner is always set; witness is nil when acc has no witness permission;
// actives follow acc's order, which keeps their permission ids (2, 3, ...) on
// re-submit. Keys are Base58 addresses mapped to int64 weights.
//
// An account with no stored owner permission, or no active permission, gives
// an error wrapping ErrNoStoredPermissions. getaccount reports such accounts
// when they were created before multi-sign and never updated: the node applies
// implicit defaults to them, which cannot be read back. Build the update
// yourself: owner threshold 1 with the account address at weight 1, and one
// active permission named "active" (id 2, threshold 1, the same key) with the
// chain's default active operations (chain parameter
// getActiveDefaultOperations). A DefaultPermissionUpdateMaps helper is
// planned.
//
// It returns an error, naming the permission ("owner", "witness",
// "active[1]"), when acc is nil, a key address is
// not a valid TRON address, an address appears twice in one permission, an
// operations bitmap sets a bit this SDK cannot name (see PermissionOperations),
// an active permission has no operations (UpdateAccountPermissionCtx cannot
// express an empty but present bitmap), or an active permission's id is not
// the one UpdateAccountPermissionCtx would assign it.
//
// UpdateAccountPermissionCtx names the owner and witness permissions "owner"
// and "witness", so a custom name on either is not kept.
func PermissionUpdateMaps(acc *core.Account) (owner map[string]interface{}, witness map[string]interface{}, actives []map[string]interface{}, err error) {
	if acc == nil {
		return nil, nil, nil, errors.New("permission update maps: nil account")
	}

	op := acc.GetOwnerPermission()
	if op == nil {
		return nil, nil, nil, fmt.Errorf("owner permission: missing: %w", ErrNoStoredPermissions)
	}
	if len(acc.GetActivePermission()) == 0 {
		return nil, nil, nil, fmt.Errorf("active permission: none stored: %w", ErrNoStoredPermissions)
	}
	ownerKeys, err := permissionKeys(op)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("owner permission: %w", err)
	}
	owner = map[string]interface{}{
		"threshold": op.GetThreshold(),
		"keys":      ownerKeys,
	}

	if wp := acc.GetWitnessPermission(); wp != nil {
		witnessKeys, err := permissionKeys(wp)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("witness permission: %w", err)
		}
		witness = map[string]interface{}{
			"threshold": wp.GetThreshold(),
			"keys":      witnessKeys,
		}
	}

	for i, ap := range acc.GetActivePermission() {
		scope := fmt.Sprintf("active[%d] permission", i)
		if want := int32(2 + i); ap.GetId() != want {
			return nil, nil, nil, fmt.Errorf("%s: id %d, want %d (UpdateAccountPermissionCtx numbers actives from 2 in order)",
				scope, ap.GetId(), want)
		}
		ops, err := PermissionOperations(ap.GetOperations())
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", scope, err)
		}
		if len(ops) == 0 {
			return nil, nil, nil, fmt.Errorf("%s: no operations set, which UpdateAccountPermissionCtx cannot re-submit", scope)
		}
		keys, err := permissionKeys(ap)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", scope, err)
		}
		actives = append(actives, map[string]interface{}{
			"name":       ap.GetPermissionName(),
			"threshold":  ap.GetThreshold(),
			"operations": ops,
			"keys":       keys,
		})
	}

	return owner, witness, actives, nil
}

// permissionKeys converts p's keys to Base58 address → weight. A duplicate
// address is an error: the map would silently merge the two entries.
func permissionKeys(p *core.Permission) (map[string]int64, error) {
	keys := make(map[string]int64, len(p.GetKeys()))
	for i, k := range p.GetKeys() {
		addr := address.Address(k.GetAddress())
		if !addr.IsValid() {
			return nil, fmt.Errorf("key[%d]: invalid key address %x", i, k.GetAddress())
		}
		b58 := addr.String()
		if _, dup := keys[b58]; dup {
			return nil, fmt.Errorf("key[%d]: duplicate key address %s", i, b58)
		}
		keys[b58] = k.GetWeight()
	}
	return keys, nil
}
