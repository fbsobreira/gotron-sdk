package client

import (
	"crypto/sha256"
	"fmt"
	"strconv"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// UpdateEnergyLimitContract update contract enery limit
func (g *GrpcClient) UpdateEnergyLimitContract(from, contractAddress string, value int64) (*api.TransactionExtention, error) {
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

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.UpdateEnergyLimit(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}

	return tx, err
}

// UpdateSettingContract change contract owner consumption ratio
func (g *GrpcClient) UpdateSettingContract(from, contractAddress string, value int64) (*api.TransactionExtention, error) {
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

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.UpdateSetting(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}

	return tx, err
}

// TriggerConstantContract and return tx result
func (g *GrpcClient) TriggerConstantContract(from, contractAddress, method, jsonString string) (*api.TransactionExtention, error) {
	var err error
	fromDesc := address.HexToAddress("410000000000000000000000000000000000000000")
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

	param, err := abi.LoadFromJSON(jsonString)
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

	return g.triggerConstantContract(ct)
}

// triggerConstantContract and return tx result
func (g *GrpcClient) triggerConstantContract(ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.TriggerConstantContract(ctx, ct)
}

// TriggerContract and return tx result
func (g *GrpcClient) TriggerContract(from, contractAddress, method, jsonString string,
	feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	param, err := abi.LoadFromJSON(jsonString)
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

	return g.triggerContract(ct, feeLimit)
}

// triggerContract and return tx result
func (g *GrpcClient) triggerContract(ct *core.TriggerSmartContract, feeLimit int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.getContext()
	defer cancel()

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
		g.UpdateHash(tx)
	}
	return tx, err
}

// EstimateEnergy returns enery required
func (g *GrpcClient) EstimateEnergy(from, contractAddress, method, jsonString string,
	tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error) {
	fromDesc, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}

	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	param, err := abi.LoadFromJSON(jsonString)
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

	return g.estimateEnergy(ct)
}

// triggerContract and return tx result
func (g *GrpcClient) estimateEnergy(ct *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.EstimateEnergy(ctx, ct)
	if err != nil {
		return nil, err
	}

	if tx.Result.Code > 0 {
		return nil, fmt.Errorf("%s", string(tx.Result.Message))
	}

	return tx, err
}

// DeployContract and return tx result
func (g *GrpcClient) DeployContract(from, contractName string,
	abi *core.SmartContract_ABI, codeStr string,
	feeLimit, curPercent, oeLimit int64,
) (*api.TransactionExtention, error) {

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

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.DeployContract(ctx, ct)
	if err != nil {
		return nil, err
	}
	if feeLimit > 0 {
		tx.Transaction.RawData.FeeLimit = feeLimit
		// update hash
		g.UpdateHash(tx)
	}
	return tx, err
}

// UpdateHash after local changes
func (g *GrpcClient) UpdateHash(tx *api.TransactionExtention) error {
	rawData, err := proto.Marshal(tx.Transaction.GetRawData())
	if err != nil {
		return err
	}

	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)
	tx.Txid = hash
	return nil
}

// GetContractABI return smartContract
func (g *GrpcClient) GetContractABI(contractAddress string) (*core.SmartContract_ABI, error) {
	var err error
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	sm, err := g.Client.GetContract(ctx, GetMessageBytes(contractDesc))
	if err != nil {
		return nil, err
	}
	if sm == nil {
		return nil, fmt.Errorf("invalid contract abi")
	}

	return sm.Abi, nil
}
