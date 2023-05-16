package contract

import (
	"encoding/json"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// JSONABI data format
type JSONABI struct {
	Anonymous bool `json:"anonymous"`
	Constant  bool `json:"constant"`
	Inputs    []struct {
		Indexed bool   `json:"indexed"`
		Name    string `json:"name"`
		Type    string `json:"type"`
	} `json:"inputs"`
	Name    string `json:"name"`
	Outputs []struct {
		Indexed bool   `json:"indexed"`
		Name    string `json:"name"`
		Type    string `json:"type"`
	} `json:"outputs"`
	Payable         bool   `json:"payable"`
	StateMutability string `json:"stateMutability"`
	Type            string `json:"type"`
}

func getState(str string) core.SmartContract_ABI_Entry_StateMutabilityType {
	switch str {
	case "pure":
		return core.SmartContract_ABI_Entry_Pure
	case "view":
		return core.SmartContract_ABI_Entry_View
	case "nonpayable":
		return core.SmartContract_ABI_Entry_Nonpayable
	case "payable":
		return core.SmartContract_ABI_Entry_Payable
	default:
		return core.SmartContract_ABI_Entry_UnknownMutabilityType
	}
}
func getType(str string) core.SmartContract_ABI_Entry_EntryType {
	switch str {
	case "constructor":
		return core.SmartContract_ABI_Entry_Constructor
	case "function":
		return core.SmartContract_ABI_Entry_Function
	case "event":
		return core.SmartContract_ABI_Entry_Event
	case "fallback":
		return core.SmartContract_ABI_Entry_Fallback
	default:
		return core.SmartContract_ABI_Entry_UnknownEntryType
	}
}

// JSONtoABI converts json string to ABI entry
func JSONtoABI(jsonSTR string) (*core.SmartContract_ABI, error) {
	jABI := []JSONABI{}
	if err := json.Unmarshal([]byte(jsonSTR), &jABI); err != nil {
		return nil, err
	}
	ABI := &core.SmartContract_ABI{}

	for _, v := range jABI {
		inputs := []*core.SmartContract_ABI_Entry_Param{}
		for _, input := range v.Inputs {
			inputs = append(inputs, &core.SmartContract_ABI_Entry_Param{
				Indexed: input.Indexed,
				Name:    input.Name,
				Type:    input.Type,
			})
		}
		outputs := []*core.SmartContract_ABI_Entry_Param{}
		for _, output := range v.Outputs {
			outputs = append(outputs, &core.SmartContract_ABI_Entry_Param{
				Indexed: output.Indexed,
				Name:    output.Name,
				Type:    output.Type,
			})
		}
		ABI.Entrys = append(ABI.Entrys,
			&core.SmartContract_ABI_Entry{
				Anonymous:       v.Anonymous,
				Constant:        v.Constant,
				Name:            v.Name,
				Payable:         v.Payable,
				Inputs:          inputs,
				Outputs:         outputs,
				Type:            getType(v.Type),
				StateMutability: getState(v.StateMutability),
			})
	}
	return ABI, nil
}
