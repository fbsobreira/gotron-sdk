package transaction

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "google.golang.org/protobuf/proto"
)

const sunPerTRX = 1_000_000

var (
	ErrNilTransaction      = errors.New("transaction is nil")
	ErrNoContracts         = errors.New("transaction has no contracts")
	ErrNilParameter        = errors.New("contract parameter is nil")
	ErrUnsupportedContract = errors.New("unsupported contract type")
	ErrUnmarshalContract   = errors.New("failed to unmarshal contract parameter")
)

// ContractData holds decoded contract parameters with human-readable values.
type ContractData struct {
	Type   string         // e.g. "TransferContract", "TriggerSmartContract"
	Fields map[string]any // decoded fields with base58 addresses and converted amounts
}

// DecodeContractData decodes the first contract parameter from a transaction
// into a ContractData struct with base58 addresses and human-readable amounts.
func DecodeContractData(tx *core.Transaction) (*ContractData, error) {
	if tx == nil || tx.GetRawData() == nil {
		return nil, ErrNilTransaction
	}

	contracts := tx.GetRawData().GetContract()
	if len(contracts) == 0 {
		return nil, ErrNoContracts
	}

	contract := contracts[0]
	if contract.GetParameter() == nil {
		return nil, ErrNilParameter
	}

	paramValue := contract.GetParameter().GetValue()
	contractType := contract.GetType()

	switch contractType {
	case core.Transaction_Contract_TransferContract:
		return decodeTransferContract(paramValue)
	case core.Transaction_Contract_TransferAssetContract:
		return decodeTransferAssetContract(paramValue)
	case core.Transaction_Contract_TriggerSmartContract:
		return decodeTriggerSmartContract(paramValue)
	case core.Transaction_Contract_FreezeBalanceV2Contract:
		return decodeFreezeBalanceV2Contract(paramValue)
	case core.Transaction_Contract_UnfreezeBalanceV2Contract:
		return decodeUnfreezeBalanceV2Contract(paramValue)
	case core.Transaction_Contract_VoteWitnessContract:
		return decodeVoteWitnessContract(paramValue)
	case core.Transaction_Contract_DelegateResourceContract:
		return decodeDelegateResourceContract(paramValue)
	case core.Transaction_Contract_UnDelegateResourceContract:
		return decodeUnDelegateResourceContract(paramValue)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedContract, contractType.String())
	}
}

func decodeTransferContract(data []byte) (*ContractData, error) {
	var c core.TransferContract
	if err := proto.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshalContract, err)
	}
	return &ContractData{
		Type: "TransferContract",
		Fields: map[string]any{
			"owner_address": address.Address(c.GetOwnerAddress()).String(),
			"to_address":    address.Address(c.GetToAddress()).String(),
			"amount":        sunToTRX(c.GetAmount()),
		},
	}, nil
}

func decodeTransferAssetContract(data []byte) (*ContractData, error) {
	var c core.TransferAssetContract
	if err := proto.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshalContract, err)
	}
	return &ContractData{
		Type: "TransferAssetContract",
		Fields: map[string]any{
			"owner_address": address.Address(c.GetOwnerAddress()).String(),
			"to_address":    address.Address(c.GetToAddress()).String(),
			"asset_name":    string(c.GetAssetName()),
			"amount":        c.GetAmount(),
		},
	}, nil
}

func decodeTriggerSmartContract(data []byte) (*ContractData, error) {
	var c core.TriggerSmartContract
	if err := proto.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshalContract, err)
	}
	return &ContractData{
		Type: "TriggerSmartContract",
		Fields: map[string]any{
			"owner_address":    address.Address(c.GetOwnerAddress()).String(),
			"contract_address": address.Address(c.GetContractAddress()).String(),
			"data":             hex.EncodeToString(c.GetData()),
			"call_value":       sunToTRX(c.GetCallValue()),
		},
	}, nil
}

func decodeFreezeBalanceV2Contract(data []byte) (*ContractData, error) {
	var c core.FreezeBalanceV2Contract
	if err := proto.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshalContract, err)
	}
	return &ContractData{
		Type: "FreezeBalanceV2Contract",
		Fields: map[string]any{
			"owner_address":  address.Address(c.GetOwnerAddress()).String(),
			"frozen_balance": sunToTRX(c.GetFrozenBalance()),
			"resource":       c.GetResource().String(),
		},
	}, nil
}

func decodeUnfreezeBalanceV2Contract(data []byte) (*ContractData, error) {
	var c core.UnfreezeBalanceV2Contract
	if err := proto.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshalContract, err)
	}
	return &ContractData{
		Type: "UnfreezeBalanceV2Contract",
		Fields: map[string]any{
			"owner_address":    address.Address(c.GetOwnerAddress()).String(),
			"unfreeze_balance": sunToTRX(c.GetUnfreezeBalance()),
			"resource":         c.GetResource().String(),
		},
	}, nil
}

func decodeVoteWitnessContract(data []byte) (*ContractData, error) {
	var c core.VoteWitnessContract
	if err := proto.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshalContract, err)
	}

	votes := make([]map[string]any, 0, len(c.GetVotes()))
	for _, v := range c.GetVotes() {
		votes = append(votes, map[string]any{
			"vote_address": address.Address(v.GetVoteAddress()).String(),
			"vote_count":   v.GetVoteCount(),
		})
	}

	return &ContractData{
		Type: "VoteWitnessContract",
		Fields: map[string]any{
			"owner_address": address.Address(c.GetOwnerAddress()).String(),
			"votes":         votes,
		},
	}, nil
}

func decodeDelegateResourceContract(data []byte) (*ContractData, error) {
	var c core.DelegateResourceContract
	if err := proto.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshalContract, err)
	}
	return &ContractData{
		Type: "DelegateResourceContract",
		Fields: map[string]any{
			"owner_address":    address.Address(c.GetOwnerAddress()).String(),
			"receiver_address": address.Address(c.GetReceiverAddress()).String(),
			"balance":          sunToTRX(c.GetBalance()),
			"resource":         c.GetResource().String(),
			"lock":             c.GetLock(),
			"lock_period":      c.GetLockPeriod(),
		},
	}, nil
}

func decodeUnDelegateResourceContract(data []byte) (*ContractData, error) {
	var c core.UnDelegateResourceContract
	if err := proto.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnmarshalContract, err)
	}
	return &ContractData{
		Type: "UnDelegateResourceContract",
		Fields: map[string]any{
			"owner_address":    address.Address(c.GetOwnerAddress()).String(),
			"receiver_address": address.Address(c.GetReceiverAddress()).String(),
			"balance":          sunToTRX(c.GetBalance()),
			"resource":         c.GetResource().String(),
		},
	}, nil
}

// sunToTRX converts a SUN amount (int64) to a TRX string with 6 decimal places.
func sunToTRX(sun int64) string {
	whole := sun / sunPerTRX
	frac := sun % sunPerTRX
	if frac < 0 {
		frac = -frac
	}
	return fmt.Sprintf("%d.%06d", whole, frac)
}
