package transaction

import (
	"errors"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// OwnerAndPermission returns the owner_address, permission_id and type of tx's
// single contract: the account and permission the node checks the signatures
// against, and the type to check with client.PermissionAllows. It fails closed
// unless tx has exactly one contract whose parameter matches its type and has
// a valid owner_address (so a ShieldedTransferContract is an error). The
// address is a copy.
func OwnerAndPermission(tx *core.Transaction) (owner address.Address, permissionID int32, contractType core.Transaction_Contract_ContractType, err error) {
	if tx == nil {
		return nil, 0, 0, errors.New("transaction: nil transaction")
	}
	if tx.GetRawData() == nil {
		return nil, 0, 0, errors.New("transaction: missing raw data")
	}
	contracts := tx.GetRawData().GetContract()
	if len(contracts) != 1 {
		return nil, 0, 0, fmt.Errorf("transaction: expected exactly 1 contract, got %d contracts", len(contracts))
	}
	contract := contracts[0]
	param := contract.GetParameter()
	if param == nil {
		return nil, 0, 0, errors.New("transaction: contract has no parameter")
	}
	msg, err := param.UnmarshalNew()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("transaction: cannot unmarshal contract parameter %q: %w", param.GetTypeUrl(), err)
	}
	// java-tron unpacks by contract.type and rejects any other parameter type.
	if got, want := string(msg.ProtoReflect().Descriptor().FullName()), "protocol."+contract.GetType().String(); got != want {
		return nil, 0, 0, fmt.Errorf("transaction: contract type %s does not match parameter %q", contract.GetType(), param.GetTypeUrl())
	}
	fd := msg.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name("owner_address"))
	if fd == nil || fd.Kind() != protoreflect.BytesKind || fd.IsList() || fd.IsMap() {
		return nil, 0, 0, fmt.Errorf("transaction: contract %q has no owner_address bytes field", param.GetTypeUrl())
	}
	raw := msg.ProtoReflect().Get(fd).Bytes()
	if len(raw) == 0 {
		return nil, 0, 0, errors.New("transaction: contract has an empty owner_address")
	}
	owner = make(address.Address, len(raw))
	copy(owner, raw)
	if !owner.IsValid() {
		return nil, 0, 0, fmt.Errorf("transaction: owner_address is not a valid TRON address (%d bytes)", len(raw))
	}
	return owner, contract.GetPermissionId(), contract.GetType(), nil
}
