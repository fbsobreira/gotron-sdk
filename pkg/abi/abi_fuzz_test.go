package abi

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

func FuzzABIEncode(f *testing.F) {
	f.Add("transfer(address,uint256)", `[{"address":"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1"},{"uint256":"1000000"}]`)
	f.Add("totalSupply()", "")
	f.Add("balanceOf(address)", `[{"address":"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1"}]`)
	f.Add("", "")
	f.Add("foo(uint256)", `[{"uint256":"42"}]`)
	f.Add("bar(bool)", `[{"bool":true}]`)

	f.Fuzz(func(t *testing.T, method, params string) {
		p, err := LoadFromJSON(params)
		if err != nil {
			return
		}
		// Just verify no panic.
		_, _ = Pack(method, p)
	})
}

// erc20ABI returns a minimal ERC-20 ABI with transfer and approve methods
// so the fuzzer can exercise GetParser and GetInputsParser beyond "not found".
func erc20ABI() *core.SmartContract_ABI {
	return &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "transfer",
				Type: core.SmartContract_ABI_Entry_Function,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "to", Type: "address"},
					{Name: "value", Type: "uint256"},
				},
				Outputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "", Type: "bool"},
				},
			},
			{
				Name: "approve",
				Type: core.SmartContract_ABI_Entry_Function,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "spender", Type: "address"},
					{Name: "value", Type: "uint256"},
				},
				Outputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "", Type: "bool"},
				},
			},
			{
				Name: "balanceOf",
				Type: core.SmartContract_ABI_Entry_Function,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "owner", Type: "address"},
				},
				Outputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "", Type: "uint256"},
				},
			},
		},
	}
}

func FuzzABIGetParser(f *testing.F) {
	f.Add("transfer")
	f.Add("approve")
	f.Add("balanceOf")
	f.Add("")
	f.Add("nonExistentMethod")
	f.Add("transfer(address,uint256)")

	f.Fuzz(func(t *testing.T, method string) {
		abi := erc20ABI()
		// Verify no panics; exercise both found and not-found paths.
		_, _ = GetParser(abi, method)
		_, _ = GetInputsParser(abi, method)
	})
}
