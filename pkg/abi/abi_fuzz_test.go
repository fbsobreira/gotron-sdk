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

func FuzzABIGetParser(f *testing.F) {
	f.Add(`[{"type":"function","name":"transfer","inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]}]`)
	f.Add(`[]`)
	f.Add(``)
	f.Add(`not valid json`)
	f.Add(`[{"type":"function","name":"balanceOf","inputs":[{"name":"owner","type":"address"}],"outputs":[{"name":"","type":"uint256"}]}]`)

	f.Fuzz(func(t *testing.T, abiJSON string) {
		abi := &core.SmartContract_ABI{}

		// Try to build an ABI from the fuzzed JSON-like string.
		// We can't easily parse arbitrary JSON into protobuf, so we test
		// GetParser with an empty/minimal ABI and the fuzzed string as method name.
		// The goal is to verify no panics.
		_, _ = GetParser(abi, abiJSON)
		_, _ = GetInputsParser(abi, abiJSON)
	})
}
