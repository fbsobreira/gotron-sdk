package contract

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"google.golang.org/protobuf/proto"
)

// zeroAddress is the default owner address used for read-only calls when no
// From address is specified. This is the base58 encoding of 21 zero bytes
// (0x410000000000000000000000000000000000000000).
const zeroAddress = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb"

// ContractCall is a builder for constructing and executing TRON smart-contract
// calls. Use New to start a builder chain, configure it with the fluent
// setter methods, and finish with a terminal operation (Call, EstimateEnergy,
// Build, Send, or SendAndConfirm).
type ContractCall struct {
	client          Client
	contractAddress string
	from            string
	method          string
	jsonParams      string
	data            []byte // pre-packed ABI data (alternative to method+jsonParams)
	abiJSON         string // parsed ABI (for future use)
	cfg             callConfig
	// err holds a deferred validation error that surfaces at any terminal call
	// (Call, Build, Send, etc.).
	//
	// Design note: builder methods like trc20.Transfer() validate addresses at
	// construction time but cannot return errors without breaking the fluent
	// chain pattern:
	//
	//   // This one-liner would be impossible with (*ContractCall, error):
	//   token.Transfer(from, to, amount).Send(ctx, signer)
	//
	// Instead, validation errors are stored via SetError and deferred until a
	// terminal is invoked — following the same pattern as bufio.Scanner.Err()
	// and database/sql Rows.Err(). Only the first error is kept; subsequent
	// SetError calls are no-ops.
	err error
}

// SetError records a deferred error that will be returned by any terminal
// operation (Call, Build, Send, etc.). Only the first error is stored;
// subsequent calls are no-ops. This preserves fluent method chaining while
// ensuring validation errors are never silently lost.
func (c *ContractCall) SetError(err error) *ContractCall {
	if c.err == nil {
		c.err = err
	}
	return c
}

// Err returns any deferred error stored in the builder, or nil if none.
// Use this to check for validation errors without invoking a terminal:
//
//	call := token.Transfer(from, "INVALID", amount)
//	if call.Err() != nil {
//	    log.Fatal(call.Err())  // "invalid to address INVALID: ..."
//	}
func (c *ContractCall) Err() error {
	return c.err
}

// New creates a new ContractCall builder targeting the given contract address.
func New(c Client, contractAddress string) *ContractCall {
	return &ContractCall{
		client:          c,
		contractAddress: contractAddress,
	}
}

// Method sets the contract method signature (e.g. "transfer(address,uint256)").
func (c *ContractCall) Method(sig string) *ContractCall {
	c.method = sig
	return c
}

// From sets the caller address. If not set, a zero address is used for
// read-only calls and an error is returned for state-changing calls.
func (c *ContractCall) From(addr string) *ContractCall {
	c.from = addr
	return c
}

// Params sets the JSON-encoded parameters for the contract method.
func (c *ContractCall) Params(jsonString string) *ContractCall {
	c.jsonParams = jsonString
	return c
}

// WithData sets pre-packed ABI call data, bypassing the method+params
// encoding pipeline. When data is set, Method and Params are ignored.
func (c *ContractCall) WithData(data []byte) *ContractCall {
	c.data = data
	return c
}

// WithABI stores the ABI JSON for future use (e.g. result decoding).
func (c *ContractCall) WithABI(abiJSON string) *ContractCall {
	c.abiJSON = abiJSON
	return c
}

// Apply applies one or more Options to the call configuration.
func (c *ContractCall) Apply(opts ...Option) *ContractCall {
	for _, o := range opts {
		o(&c.cfg)
	}
	return c
}

// WithPermissionID sets the permission ID for multi-signature transactions.
// Returns itself for chaining.
func (c *ContractCall) WithPermissionID(id int32) *ContractCall {
	return c.Apply(WithPermissionID(id))
}

// WithFeeLimit sets the maximum TRX (in SUN) the caller is willing to spend
// on energy for a state-changing contract call. Returns itself for chaining.
func (c *ContractCall) WithFeeLimit(limit int64) *ContractCall {
	return c.Apply(WithFeeLimit(limit))
}

// WithCallValue sets the TRX amount (in SUN) sent along with the call.
// Used by both read-only Call and state-changing Build/Send.
// Returns itself for chaining.
func (c *ContractCall) WithCallValue(value int64) *ContractCall {
	return c.Apply(WithCallValue(value))
}

// WithTokenValue sets the TRC10 token ID and amount sent with the call.
// Only affects state-changing operations (Build, Send, SendAndConfirm) and
// EstimateEnergy; read-only Call does not forward token parameters.
// Returns itself for chaining.
func (c *ContractCall) WithTokenValue(tokenID string, amount int64) *ContractCall {
	return c.Apply(WithTokenValue(tokenID, amount))
}

// fromOrZero returns the configured from address, falling back to the zero
// address for read-only operations.
func (c *ContractCall) fromOrZero() string {
	if c.from != "" {
		return c.from
	}
	return zeroAddress
}

// Call executes a constant (read-only) contract call and returns the raw results.
func (c *ContractCall) Call(ctx context.Context) (*CallResult, error) {
	if c.err != nil {
		return nil, c.err
	}
	var (
		tx  *api.TransactionExtention
		err error
	)

	from := c.fromOrZero()

	var opts []client.ConstantCallOption
	if c.cfg.callValue > 0 {
		opts = append(opts, client.WithCallValue(c.cfg.callValue))
	}

	if len(c.data) > 0 {
		tx, err = c.client.TriggerConstantContractWithDataCtx(ctx, from, c.contractAddress, c.data, opts...)
	} else {
		tx, err = c.client.TriggerConstantContractCtx(ctx, from, c.contractAddress, c.method, c.jsonParams, opts...)
	}
	if err != nil {
		return nil, err
	}

	result := &CallResult{
		RawResults: tx.GetConstantResult(),
		EnergyUsed: tx.GetEnergyUsed(),
	}

	return result, nil
}

// EstimateEnergy returns the estimated energy required for the contract call.
// From address is required for accurate estimation.
func (c *ContractCall) EstimateEnergy(ctx context.Context) (int64, error) {
	if c.err != nil {
		return 0, c.err
	}
	if c.from == "" {
		return 0, errors.New("From address is required for energy estimation")
	}

	var (
		estimate *api.EstimateEnergyMessage
		err      error
	)

	if len(c.data) > 0 {
		estimate, err = c.client.EstimateEnergyWithDataCtx(
			ctx, c.from, c.contractAddress, c.data,
			c.cfg.callValue, c.cfg.tokenID, c.cfg.tokenAmount,
		)
	} else {
		estimate, err = c.client.EstimateEnergyCtx(
			ctx, c.from, c.contractAddress,
			c.method, c.jsonParams,
			c.cfg.callValue, c.cfg.tokenID, c.cfg.tokenAmount,
		)
	}
	if err != nil {
		return 0, err
	}

	return estimate.GetEnergyRequired(), nil
}

// Decode builds the transaction and decodes the contract parameters into
// human-readable fields (base58 addresses, TRX-formatted amounts). Useful for
// inspecting what a contract call does before signing.
func (c *ContractCall) Decode(ctx context.Context) (*transaction.ContractData, error) {
	ext, err := c.Build(ctx)
	if err != nil {
		return nil, err
	}
	return transaction.DecodeContractData(ext.Transaction)
}

// Build creates a state-changing transaction without signing or broadcasting.
// The returned TransactionExtention can be inspected or signed externally.
func (c *ContractCall) Build(ctx context.Context) (*api.TransactionExtention, error) {
	if c.err != nil {
		return nil, c.err
	}
	if c.from == "" {
		return nil, errors.New("From address is required for state-changing calls")
	}

	var (
		tx  *api.TransactionExtention
		err error
	)

	if len(c.data) > 0 {
		tx, err = c.client.TriggerContractWithDataCtx(
			ctx, c.from, c.contractAddress, c.data,
			c.cfg.feeLimit, c.cfg.callValue, c.cfg.tokenID, c.cfg.tokenAmount,
		)
	} else {
		tx, err = c.client.TriggerContractCtx(
			ctx, c.from, c.contractAddress, c.method, c.jsonParams,
			c.cfg.feeLimit, c.cfg.callValue, c.cfg.tokenID, c.cfg.tokenAmount,
		)
	}
	if err != nil {
		return nil, err
	}

	// Apply permission ID if set.
	if c.cfg.permissionID != nil {
		if err := tx.SetPermissionId(*c.cfg.permissionID); err != nil {
			return nil, fmt.Errorf("set permission ID: %w", err)
		}
	}

	return tx, nil
}

// Send builds, signs, and broadcasts a state-changing transaction.
func (c *ContractCall) Send(ctx context.Context, s signer.Signer) (*Receipt, error) {
	tx, err := c.Build(ctx)
	if err != nil {
		return nil, err
	}

	signed, err := s.Sign(tx.GetTransaction())
	if err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	txID, err := transactionID(signed)
	if err != nil {
		return nil, fmt.Errorf("computing tx ID: %w", err)
	}
	receipt := &Receipt{TxID: txID}

	result, err := c.client.BroadcastCtx(ctx, signed)
	if err != nil {
		return receipt, fmt.Errorf("broadcast transaction: %w", err)
	}

	if result.Code != 0 {
		receipt.Error = string(result.GetMessage())
	}

	return receipt, nil
}

// SendAndConfirm is like Send but additionally polls for transaction
// confirmation on-chain. It relies on the context for timeout control.
func (c *ContractCall) SendAndConfirm(ctx context.Context, s signer.Signer) (*Receipt, error) {
	receipt, err := c.Send(ctx, s)
	if err != nil {
		return receipt, err
	}
	if receipt.Error != "" {
		return receipt, nil
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return receipt, fmt.Errorf("waiting for confirmation: %w", ctx.Err())
		case <-ticker.C:
			info, infoErr := c.client.GetTransactionInfoByIDCtx(ctx, receipt.TxID)
			if infoErr != nil {
				if strings.Contains(infoErr.Error(), "not found") {
					continue
				}
				return receipt, fmt.Errorf("checking confirmation: %w", infoErr)
			}
			if info.GetBlockNumber() == 0 {
				continue
			}

			receipt.BlockNumber = info.GetBlockNumber()
			receipt.Fee = info.GetFee()
			receipt.Confirmed = true

			if r := info.GetReceipt(); r != nil {
				receipt.EnergyUsed = r.GetEnergyUsageTotal()
				receipt.BandwidthUsed = r.GetNetUsage()
			}
			if len(info.GetContractResult()) > 0 {
				receipt.Result = info.GetContractResult()[0]
			}
			if info.GetResult() == core.TransactionInfo_FAILED {
				receipt.Error = string(info.GetResMessage())
			}
			return receipt, nil
		}
	}
}

// transactionID computes the hex-encoded SHA-256 hash of the marshalled
// RawData, which is the canonical TRON transaction ID.
func transactionID(tx *core.Transaction) (string, error) {
	raw, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(raw)
	return common.BytesToHexString(h[:]), nil
}
