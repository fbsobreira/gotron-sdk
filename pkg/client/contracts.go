package client

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// defaultFromAddress is the zero address used as the default "from" for
// read-only contract calls. Decoded once at init to avoid repeated parsing.
var defaultFromAddress, _ = address.HexToAddress("410000000000000000000000000000000000000000")

// ConstantCallOption configures optional fields on a TriggerSmartContract
// used by constant (read-only) contract calls. This allows callers to set
// values like CallValue or TokenValue without breaking backward compatibility.
type ConstantCallOption func(*core.TriggerSmartContract)

// WithCallValue sets the TRX call value (in sun) on a constant contract call.
// This is required to accurately simulate payable functions that depend on msg.value.
func WithCallValue(value int64) ConstantCallOption {
	return func(ct *core.TriggerSmartContract) {
		ct.CallValue = value
	}
}

// WithTokenValue sets the TRC10 token ID and amount on a constant contract call.
// It returns an error if tokenID is not a valid integer.
func WithTokenValue(tokenID string, amount int64) (ConstantCallOption, error) {
	id, err := strconv.ParseInt(tokenID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid token ID %q: %w", tokenID, err)
	}
	return func(ct *core.TriggerSmartContract) {
		ct.TokenId = id
		ct.CallTokenValue = amount
	}, nil
}

func applyConstantCallOptions(ct *core.TriggerSmartContract, opts []ConstantCallOption) {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(ct)
	}
}

// UpdateEnergyLimitContract updates the energy limit of a deployed smart contract.
func (g *GrpcClient) UpdateEnergyLimitContract(from, contractAddress string, value int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.UpdateEnergyLimitContractCtx(ctx, from, contractAddress, value)
}

// UpdateEnergyLimitContractCtx is the context-aware version of UpdateEnergyLimitContract.
func (g *GrpcClient) UpdateEnergyLimitContractCtx(ctx context.Context, from, contractAddress string, value int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	ct := &core.UpdateEnergyLimitContract{
		OwnerAddress:      fromDesc.Bytes(),
		ContractAddress:   contractDesc.Bytes(),
		OriginEnergyLimit: value,
	}

	tx, err := g.Client.UpdateEnergyLimit(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}

	return tx, err
}

// UpdateSettingContract changes the user resource consumption ratio of a deployed contract.
func (g *GrpcClient) UpdateSettingContract(from, contractAddress string, value int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.UpdateSettingContractCtx(ctx, from, contractAddress, value)
}

// UpdateSettingContractCtx is the context-aware version of UpdateSettingContract.
func (g *GrpcClient) UpdateSettingContractCtx(ctx context.Context, from, contractAddress string, value int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	ct := &core.UpdateSettingContract{
		OwnerAddress:               fromDesc.Bytes(),
		ContractAddress:            contractDesc.Bytes(),
		ConsumeUserResourcePercent: value,
	}

	tx, err := g.Client.UpdateSetting(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}

	return tx, err
}

// TriggerConstantContract executes a read-only smart contract call and returns the result.
func (g *GrpcClient) TriggerConstantContract(from, contractAddress, method, jsonString string, opts ...ConstantCallOption) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TriggerConstantContractCtx(ctx, from, contractAddress, method, jsonString, opts...)
}

// TriggerConstantContractCtx is the context-aware version of TriggerConstantContract.
func (g *GrpcClient) TriggerConstantContractCtx(ctx context.Context, from, contractAddress, method, jsonString string, opts ...ConstantCallOption) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	var err error
	fromDesc := defaultFromAddress
	if len(from) > 0 {
		fromDesc, err = address.Base58ToAddress(from)
		if err != nil {
			return nil, err
		}
	}
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	param, err := abi.LoadFromJSONWithMethod(method, jsonString)
	if err != nil {
		return nil, err
	}

	dataBytes, err := abi.Pack(method, param)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            dataBytes,
	}
	applyConstantCallOptions(ct, opts)

	return g.triggerConstantContract(ctx, ct)
}

// triggerConstantContract and return tx result
func (g *GrpcClient) triggerConstantContract(ctx context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	return g.Client.TriggerConstantContract(ctx, ct)
}

// TriggerContract executes a state-changing smart contract call and returns the unsigned transaction.
func (g *GrpcClient) TriggerContract(from, contractAddress, method, jsonString string,
	feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TriggerContractCtx(ctx, from, contractAddress, method, jsonString, feeLimit, tAmount, tTokenID, tTokenAmount)
}

// TriggerContractCtx is the context-aware version of TriggerContract.
func (g *GrpcClient) TriggerContractCtx(ctx context.Context, from, contractAddress, method, jsonString string,
	feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	param, err := abi.LoadFromJSONWithMethod(method, jsonString)
	if err != nil {
		return nil, err
	}

	dataBytes, err := abi.Pack(method, param)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            dataBytes,
	}
	if tAmount > 0 {
		ct.CallValue = tAmount
	}
	if len(tTokenID) > 0 && tTokenAmount > 0 {
		ct.CallTokenValue = tTokenAmount
		ct.TokenId, err = strconv.ParseInt(tTokenID, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	return g.triggerContract(ctx, ct, feeLimit)
}

// triggerContract and return tx result
func (g *GrpcClient) triggerContract(ctx context.Context, ct *core.TriggerSmartContract, feeLimit int64) (*api.TransactionExtention, error) {
	tx, err := g.Client.TriggerContract(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}
	if feeLimit > 0 {
		tx.Transaction.RawData.FeeLimit = feeLimit
		// update hash
		err = g.UpdateHash(tx)
	}
	return tx, err
}

// TriggerConstantContractWithData calls a constant contract method using
// pre-packed ABI data, bypassing the JSON string → parse → pack pipeline.
// This is useful when callers already have packed data from go-ethereum's
// abi.Pack() or similar tooling.
func (g *GrpcClient) TriggerConstantContractWithData(from, contractAddress string, data []byte, opts ...ConstantCallOption) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TriggerConstantContractWithDataCtx(ctx, from, contractAddress, data, opts...)
}

// TriggerConstantContractWithDataCtx is the context-aware version of TriggerConstantContractWithData.
func (g *GrpcClient) TriggerConstantContractWithDataCtx(ctx context.Context, from, contractAddress string, data []byte, opts ...ConstantCallOption) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	var err error
	fromDesc := defaultFromAddress
	if len(from) > 0 {
		fromDesc, err = address.Base58ToAddress(from)
		if err != nil {
			return nil, err
		}
	}
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            data,
	}
	applyConstantCallOptions(ct, opts)

	return g.triggerConstantContract(ctx, ct)
}

// TriggerContractWithData triggers a contract method using pre-packed ABI
// data, bypassing the JSON string → parse → pack pipeline. This is useful
// when callers already have packed data from go-ethereum's abi.Pack() or
// similar tooling.
func (g *GrpcClient) TriggerContractWithData(from, contractAddress string, data []byte,
	feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TriggerContractWithDataCtx(ctx, from, contractAddress, data, feeLimit, tAmount, tTokenID, tTokenAmount)
}

// TriggerContractWithDataCtx is the context-aware version of TriggerContractWithData.
func (g *GrpcClient) TriggerContractWithDataCtx(ctx context.Context, from, contractAddress string, data []byte,
	feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            data,
	}
	if tAmount > 0 {
		ct.CallValue = tAmount
	}
	if len(tTokenID) > 0 && tTokenAmount > 0 {
		ct.CallTokenValue = tTokenAmount
		ct.TokenId, err = strconv.ParseInt(tTokenID, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	return g.triggerContract(ctx, ct, feeLimit)
}

// EstimateEnergy returns the estimated energy required for a contract call.
func (g *GrpcClient) EstimateEnergy(from, contractAddress, method, jsonString string,
	tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.EstimateEnergyCtx(ctx, from, contractAddress, method, jsonString, tAmount, tTokenID, tTokenAmount)
}

// EstimateEnergyCtx is the context-aware version of EstimateEnergy.
func (g *GrpcClient) EstimateEnergyCtx(ctx context.Context, from, contractAddress, method, jsonString string,
	tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error) {
	ctx = g.withAPIKey(ctx)

	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	param, err := abi.LoadFromJSONWithMethod(method, jsonString)
	if err != nil {
		return nil, err
	}

	dataBytes, err := abi.Pack(method, param)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            dataBytes,
	}
	if tAmount > 0 {
		ct.CallValue = tAmount
	}
	if len(tTokenID) > 0 && tTokenAmount > 0 {
		ct.CallTokenValue = tTokenAmount
		ct.TokenId, err = strconv.ParseInt(tTokenID, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	return g.estimateEnergy(ctx, ct)
}

// EstimateEnergyWithData returns the estimated energy using pre-packed ABI
// data, bypassing the JSON string → parse → pack pipeline.
func (g *GrpcClient) EstimateEnergyWithData(from, contractAddress string, data []byte,
	tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.EstimateEnergyWithDataCtx(ctx, from, contractAddress, data, tAmount, tTokenID, tTokenAmount)
}

// EstimateEnergyWithDataCtx is the context-aware version of EstimateEnergyWithData.
func (g *GrpcClient) EstimateEnergyWithDataCtx(ctx context.Context, from, contractAddress string, data []byte,
	tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error) {
	ctx = g.withAPIKey(ctx)

	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            data,
	}
	if tAmount > 0 {
		ct.CallValue = tAmount
	}
	if len(tTokenID) > 0 && tTokenAmount > 0 {
		ct.CallTokenValue = tTokenAmount
		ct.TokenId, err = strconv.ParseInt(tTokenID, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	return g.estimateEnergy(ctx, ct)
}

// triggerContract and return tx result
func (g *GrpcClient) estimateEnergy(ctx context.Context, ct *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	tx, err := g.Client.EstimateEnergy(ctx, ct)
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.Unimplemented {
			return nil, fmt.Errorf("%w: %w", ErrEstimateEnergyNotSupported, err)
		}
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}

	return tx, err
}

// DeployContract deploys a new smart contract and returns the unsigned transaction.
func (g *GrpcClient) DeployContract(from, contractName string,
	abi *core.SmartContract_ABI, codeStr string,
	feeLimit, curPercent, oeLimit int64,
) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.DeployContractCtx(ctx, from, contractName, abi, codeStr, feeLimit, curPercent, oeLimit)
}

// DeployContractCtx is the context-aware version of DeployContract.
func (g *GrpcClient) DeployContractCtx(ctx context.Context, from, contractName string,
	abi *core.SmartContract_ABI, codeStr string,
	feeLimit, curPercent, oeLimit int64,
) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	var err error

	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	if curPercent > 100 || curPercent < 0 {
		return nil, fmt.Errorf("consume_user_resource_percent should be >= 0 and <= 100")
	}
	if oeLimit <= 0 {
		return nil, fmt.Errorf("origin_energy_limit must > 0")
	}

	bc, err := common.FromHex(codeStr)
	if err != nil {
		return nil, err
	}

	ct := &core.CreateSmartContract{
		OwnerAddress: fromDesc.Bytes(),
		NewContract: &core.SmartContract{
			OriginAddress:              fromDesc.Bytes(),
			Abi:                        abi,
			Name:                       contractName,
			ConsumeUserResourcePercent: curPercent,
			OriginEnergyLimit:          oeLimit,
			Bytecode:                   bc,
		},
	}

	tx, err := g.Client.DeployContract(ctx, ct)
	if err != nil {
		return nil, err
	}
	if feeLimit > 0 {
		tx.Transaction.RawData.FeeLimit = feeLimit
		// update hash
		err = g.UpdateHash(tx)
	}
	return tx, err
}

// UpdateHash recalculates the transaction hash after local modifications (e.g. setting fee limit).
func (g *GrpcClient) UpdateHash(tx *api.TransactionExtention) error {
	return tx.UpdateHash()
}

// GetContractABI returns the ABI of a deployed smart contract.
func (g *GrpcClient) GetContractABI(contractAddress string) (*core.SmartContract_ABI, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetContractABICtx(ctx, contractAddress)
}

// GetContractABICtx is the context-aware version of GetContractABI.
func (g *GrpcClient) GetContractABICtx(ctx context.Context, contractAddress string) (*core.SmartContract_ABI, error) {
	ctx = g.withAPIKey(ctx)

	var err error
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	sm, err := g.Client.GetContract(ctx, GetMessageBytes(contractDesc))
	if err != nil {
		return nil, err
	}
	if sm == nil {
		return nil, fmt.Errorf("invalid contract abi")
	}

	return sm.Abi, nil
}

// proxySelectors lists the EVM function selectors tried (in order) when
// resolving a proxy contract's implementation address.  Each entry is a
// 4-byte keccak256 prefix of a well-known getter exposed by different
// proxy patterns.
var proxySelectors = [][4]byte{
	{0x5c, 0x60, 0xda, 0x1b}, // implementation()              — ERC-1967 / OpenZeppelin / UUPS
	{0xbb, 0x82, 0xaa, 0x5e}, // comptrollerImplementation()   — Compound-style (Unitroller, etc.)
	{0xaa, 0xf1, 0x0f, 0x42}, // getImplementation()           — alternate proxy getter
	{0xa6, 0x19, 0x48, 0x6e}, // masterCopy()                  — Gnosis Safe / GnosisSafeProxy
}

// zeroEVMAddr is a pre-allocated zero address used by callForAddress to
// detect invalid implementation addresses without allocating on each call.
var zeroEVMAddr [20]byte

// GetContractABIResolved returns the ABI for a contract, resolving proxy
// contracts transparently.  It first calls GetContractABI on the given
// address; if the returned ABI has no entries, or if the ABI looks like a
// proxy-only ABI (contains an "implementation" function), it attempts to
// detect a proxy by trying several well-known proxy getter selectors
// (implementation(), comptrollerImplementation(), getImplementation(),
// masterCopy()).  On success it fetches the ABI from the implementation
// contract instead.
//
// Only a single level of proxy indirection is resolved; chained proxies
// (proxy → proxy → implementation) are not followed.
func (g *GrpcClient) GetContractABIResolved(contractAddress string) (*core.SmartContract_ABI, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetContractABIResolvedCtx(ctx, contractAddress)
}

// GetContractABIResolvedCtx is the context-aware version of GetContractABIResolved.
func (g *GrpcClient) GetContractABIResolvedCtx(ctx context.Context, contractAddress string) (*core.SmartContract_ABI, error) {
	ctx = g.withAPIKey(ctx)

	contractABI, err := g.GetContractABICtx(ctx, contractAddress)
	if err != nil {
		return nil, err
	}
	if len(contractABI.GetEntrys()) > 0 && !isProxyABI(contractABI) {
		return contractABI, nil
	}

	implAddr, err := g.getProxyImplementation(ctx, contractAddress)
	if err != nil || implAddr == "" {
		// Not a recognised proxy — return the original ABI.
		return contractABI, nil
	}

	implABI, err := g.GetContractABICtx(ctx, implAddr)
	if err != nil {
		return contractABI, nil
	}
	if len(implABI.GetEntrys()) == 0 {
		return contractABI, nil
	}
	return implABI, nil
}

// isProxyABI reports whether the given ABI looks like a proxy contract ABI
// rather than a real business-logic ABI.  It returns true when the ABI
// declares a function matching one of the well-known proxy getter names.
//
// This is a heuristic: a non-proxy contract with a matching function name
// would trigger proxy resolution, but the fallback logic in
// GetContractABIResolved ensures the original ABI is returned if resolution
// fails or produces no improvement.
func isProxyABI(contractABI *core.SmartContract_ABI) bool {
	for _, entry := range contractABI.GetEntrys() {
		if entry.GetType() != core.SmartContract_ABI_Entry_Function {
			continue
		}
		switch entry.GetName() {
		case "implementation", "comptrollerImplementation", "getImplementation", "masterCopy":
			return true
		}
	}
	return false
}

// getProxyImplementation tries multiple well-known getter selectors to
// discover the implementation address behind a proxy contract.  Returns the
// implementation address in Base58 format, or an empty string if no
// strategy succeeds.
func (g *GrpcClient) getProxyImplementation(ctx context.Context, contractAddress string) (string, error) {
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return "", err
	}
	contractBytes := contractDesc.Bytes()
	ownerBytes := defaultFromAddress.Bytes()

	for _, sel := range proxySelectors {
		addr := g.callForAddress(ctx, ownerBytes, contractBytes, sel[:])
		if addr != "" {
			return addr, nil
		}
	}

	return "", nil
}

// callForAddress sends a constant contract call with the given data and
// interprets the result as an ABI-encoded address.  Returns the Base58
// Tron address on success, or an empty string if the call fails or
// returns a zero/invalid address.
func (g *GrpcClient) callForAddress(ctx context.Context, ownerBytes, contractBytes, data []byte) string {
	ct := &core.TriggerSmartContract{
		OwnerAddress:    ownerBytes,
		ContractAddress: contractBytes,
		Data:            data,
	}

	tx, err := g.triggerConstantContract(ctx, ct)
	if err != nil || tx == nil {
		return ""
	}
	if res := tx.GetResult(); res == nil || res.GetCode() != 0 || !res.GetResult() {
		return ""
	}
	if len(tx.GetConstantResult()) == 0 || len(tx.GetConstantResult()[0]) < 32 {
		return ""
	}

	// The result is an ABI-encoded address: 12 bytes zero-padding followed
	// by the 20-byte EVM address.  Extract bytes [12:32] rather than
	// using the tail, so oversized responses are handled correctly.
	result := tx.GetConstantResult()[0]
	evmAddr := result[12:32]

	// Check for zero address — not a valid implementation.
	if bytes.Equal(evmAddr, zeroEVMAddr[:]) {
		return ""
	}

	tronAddr := make([]byte, 0, address.AddressLength)
	tronAddr = append(tronAddr, address.TronBytePrefix)
	tronAddr = append(tronAddr, evmAddr...)
	return address.Address(tronAddr).String()
}
