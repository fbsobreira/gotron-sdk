package contract_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/contract"
)

// This example demonstrates the deferred-error pattern used by ContractCall.
// Validation errors from builder methods (e.g. invalid addresses) are
// accumulated internally and surfaced when a terminal operation is called.
func Example_deferredErrors() {
	// Suppose we have a ContractCall with validation errors set during
	// construction (e.g. by a TRC20 helper that validates addresses):
	var client contract.Client // nil — we never reach the RPC call

	call := contract.New(client, "TContractAddr").
		Method("transfer(address,uint256)").
		From("TFromAddr")

	// Simulate two validation errors that would be set by a higher-level
	// helper (e.g. trc20.Transfer):
	call.SetError(errors.New("invalid to address"))
	call.SetError(errors.New("amount must be positive"))

	// The error can be checked early without invoking a terminal:
	if call.Err() != nil {
		fmt.Println("early check:", call.Err())
	}

	// Or it surfaces automatically at any terminal:
	_, err := call.Build(context.Background())
	if err != nil {
		fmt.Println("build error:", err)
	}

	// Output:
	// early check: invalid to address
	// amount must be positive
	// build error: invalid to address
	// amount must be positive
}
