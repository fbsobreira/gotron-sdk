package client_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func TestProtoParse(t *testing.T) {
	raw := &core.TransactionRaw{}

	mb, _ := hex.DecodeString("0a020cd222081e6d180d0ea1be1340c082fc94c22e5a8e01081f1289010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412540a15419df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d121541157a629d8e8d7d43218b83240afaa02e8c300b36222497a5d5b50000000000000000000000009df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d7085c1f894c22e")

	proto.Unmarshal(mb, raw)
	fmt.Printf("Raw: %+v\n", raw)
	c := raw.GetContract()[0]
	trig := &core.TriggerSmartContract{}
	// recover
	err := c.GetParameter().UnmarshalTo(trig)
	require.Nil(t, err)
	assert.Equal(t, hex.EncodeToString(trig.Data), "97a5d5b50000000000000000000000009df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d")
}

func TestProtoParseR(t *testing.T) {
	conn := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := conn.Start(grpc.WithInsecure())
	require.Nil(t, err)
	block, err := conn.GetBlockByNum(48763870)
	require.Nil(t, err)

	for _, tx := range block.Transactions {
		for _, contract := range tx.GetTransaction().GetRawData().GetContract() {
			switch contract.Type {
			case core.Transaction_Contract_TriggerSmartContract:
				tsc := core.TriggerSmartContract{}
				err := contract.Parameter.UnmarshalTo(&tsc)
				require.Nil(t, err)
				fmt.Println("Its ok.... test contract data")
			default:
				fmt.Println("handle not SC case")
			}
		}
	}
}

func TestEstimateEnergy(t *testing.T) {
	conn := client.NewGrpcClient("grpc.nile.trongrid.io:50051")
	err := conn.Start(grpc.WithInsecure())
	require.Nil(t, err)

	estimate, err := conn.EstimateEnergy(
		"TTGhREx2pDSxFX555NWz1YwGpiBVPvQA7e",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		"transfer(address,uint256)",
		`[{"address": "TE4c73WubeWPhSF1nAovQDmQytjcaLZyY9"},{"uint256": "100"}]`,
		0, "", 0,
	)
	require.Nil(t, err)
	assert.True(t, estimate.Result.Result)
	assert.Equal(t, estimate.EnergyRequired, int64(14910))
}

func TestGetAccount(t *testing.T) {
	conn := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := conn.Start(grpc.WithInsecure())
	require.Nil(t, err)

	tx, err := conn.TriggerConstantContract("",
		"TBvmoZWgmx3wqvJoDyejSXqWWogy6kCNGp",
		"statusOf(address)", `[{"address": "TQNKDtPaeSSGhtbDAykLeHEpMpfUYmSuj1"}]`)
	require.Nil(t, err)

	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)

	/*const abiJSON = `[{"outputs":[{"type":"address"}],"constant":true,"name":"kleverToken","stateMutability":"View","type":"Function"},{"name":"unpause","stateMutability":"Nonpayable","type":"Function"},{"outputs":[{"type":"bool"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"isPauser","stateMutability":"View","type":"Function"},{"inputs":[{"name":"_to","type":"address"}],"name":"reclaimTRX","stateMutability":"Nonpayable","type":"Function"},{"outputs":[{"type":"bool"}],"constant":true,"name":"paused","stateMutability":"View","type":"Function"},{"name":"renouncePauser","stateMutability":"Nonpayable","type":"Function"},{"name":"renounceOwnership","stateMutability":"Nonpayable","type":"Function"},{"inputs":[{"name":"account","type":"address"}],"name":"addPauser","stateMutability":"Nonpayable","type":"Function"},{"name":"pause","stateMutability":"Nonpayable","type":"Function"},{"inputs":[{"name":"_token","type":"address"},{"name":"_to","type":"address"}],"name":"reclaimToken","stateMutability":"Nonpayable","type":"Function"},{"outputs":[{"type":"address"}],"constant":true,"name":"owner","stateMutability":"View","type":"Function"},{"outputs":[{"type":"bool"}],"constant":true,"name":"isOwner","stateMutability":"View","type":"Function"},{"outputs":[{"type":"uint256"}],"constant":true,"name":"totalMinted","stateMutability":"View","type":"Function"},{"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","stateMutability":"Nonpayable","type":"Function"},{"inputs":[{"name":"_kleverToken","type":"address"}],"stateMutability":"Nonpayable","type":"Constructor"},{"inputs":[{"name":"account","type":"address"}],"name":"Paused","type":"Event"},{"inputs":[{"name":"account","type":"address"}],"name":"Unpaused","type":"Event"},{"inputs":[{"indexed":true,"name":"account","type":"address"}],"name":"PauserAdded","type":"Event"},{"inputs":[{"indexed":true,"name":"account","type":"address"}],"name":"PauserRemoved","type":"Event"},{"inputs":[{"indexed":true,"name":"previousOwner","type":"address"},{"indexed":true,"name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"Event"},{"inputs":[{"name":"APR","type":"uint32"}],"name":"SetAPR","type":"Event"},{"inputs":[{"indexed":true,"name":"account","type":"address"},{"name":"amount","type":"uint256"},{"name":"APR","type":"uint32"},{"name":"totalFrozen","type":"uint256"}],"name":"Freeze","type":"Event"},{"inputs":[{"indexed":true,"name":"account","type":"address"},{"name":"amount","type":"uint256"},{"name":"totalUnfrozen","type":"uint256"},{"name":"realizedInterest","type":"uint256"},{"name":"availableDate","type":"uint64"}],"name":"Unfreeze","type":"Event"},{"inputs":[{"indexed":true,"name":"account","type":"address"},{"name":"totalUnfrozen","type":"uint256"},{"name":"realizedInterest","type":"uint256"}],"name":"Withdraw","type":"Event"},{"inputs":[{"indexed":true,"name":"account","type":"address"},{"name":"totalInterest","type":"uint256"},{"name":"realizedInterest","type":"uint256"}],"name":"Claim","type":"Event"},{"inputs":[{"name":"timeInSec","type":"uint64"}],"name":"SetMinTimeToUnfreeze","type":"Event"},{"inputs":[{"name":"timeInSec","type":"uint64"}],"name":"SetUnfreezeDelay","type":"Event"},{"inputs":[{"name":"timeInSec","type":"uint64"}],"name":"SetClaimDelay","type":"Event"},{"outputs":[{"name":"total","type":"uint256"}],"constant":true,"name":"totalFrozen","stateMutability":"View","type":"Function"},{"outputs":[{"name":"value","type":"uint32"}],"constant":true,"name":"currentAPR","stateMutability":"View","type":"Function"},{"outputs":[{"name":"success","type":"bool"}],"inputs":[{"name":"value","type":"uint32"}],"name":"setAPR","stateMutability":"Nonpayable","type":"Function"},{"outputs":[{"name":"value","type":"uint64"}],"constant":true,"name":"minTimeToUnfreeze","stateMutability":"View","type":"Function"},{"outputs":[{"name":"value","type":"uint64"}],"constant":true,"name":"unfreezeDelay","stateMutability":"View","type":"Function"},{"outputs":[{"name":"value","type":"uint64"}],"constant":true,"name":"claimDelay","stateMutability":"View","type":"Function"},{"outputs":[{"name":"success","type":"bool"}],"inputs":[{"name":"timeInSec","type":"uint64"}],"name":"setMinTimeToUnfreeze","stateMutability":"Nonpayable","type":"Function"},{"outputs":[{"name":"success","type":"bool"}],"inputs":[{"name":"timeInSec","type":"uint64"}],"name":"setUnfreezeDelay","stateMutability":"Nonpayable","type":"Function"},{"outputs":[{"name":"success","type":"bool"}],"inputs":[{"name":"timeInSec","type":"uint64"}],"name":"setClaimDelay","stateMutability":"Nonpayable","type":"Function"},{"outputs":[{"name":"frozenAmount","type":"uint256"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"frozenBalanceOf","stateMutability":"View","type":"Function"},{"outputs":[{"name":"unfrozenAmount","type":"uint256"},{"name":"availableOn","type":"uint64"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"unfrozenBalanceOf","stateMutability":"View","type":"Function"},{"outputs":[{"name":"realizedAmount","type":"uint256"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"realizedInterestOf","stateMutability":"View","type":"Function"},{"outputs":[{"name":"lastClaim","type":"uint64"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"lastClaimOf","stateMutability":"View","type":"Function"},{"outputs":[{"name":"frozenDate","type":"uint64"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"frozenDateOf","stateMutability":"View","type":"Function"},{"outputs":[{"name":"pendingAmount","type":"uint256"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"pendingInterestOf","stateMutability":"View","type":"Function"},{"outputs":[{"name":"APR","type":"uint32"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"aprOf","stateMutability":"View","type":"Function"},{"outputs":[{"name":"timestamp","type":"uint64"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"canUnfreezeOn","stateMutability":"View","type":"Function"},{"outputs":[{"name":"frozen","type":"uint256"},{"name":"unfreezeAvailableOn","type":"uint64"},{"name":"frozenDate","type":"uint64"},{"name":"pendingInterest","type":"uint256"},{"name":"realizedInterest","type":"uint256"},{"name":"APR","type":"uint32"},{"name":"unfrozen","type":"uint256"},{"name":"availableOn","type":"uint64"},{"name":"lastClaim","type":"uint64"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"statusOf","stateMutability":"View","type":"Function"},{"outputs":[{"name":"success","type":"bool"}],"inputs":[{"name":"amount","type":"uint256"}],"name":"freeze","stateMutability":"Nonpayable","type":"Function"},{"outputs":[{"name":"availableDate","type":"uint64"}],"name":"unfreeze","stateMutability":"Nonpayable","type":"Function"},{"outputs":[{"name":"interestValue","type":"uint256"},{"name":"unfrozenValue","type":"uint256"}],"name":"withdraw","stateMutability":"Nonpayable","type":"Function"},{"outputs":[{"name":"interest","type":"uint256"}],"name":"claim","stateMutability":"Nonpayable","type":"Function"}]`
	a, _ := contract.JSONtoABI(abiJSON)
	*/
	a, err := conn.GetContractABI("TBvmoZWgmx3wqvJoDyejSXqWWogy6kCNGp")
	require.Nil(t, err)
	arg, err := abi.GetParser(a, "statusOf")
	require.Nil(t, err)
	fmt.Printf("\nContractABI ->>> %+v\n\n", a)
	fmt.Printf("\nResult ->>> %s\n\n", hex.EncodeToString(tx.ConstantResult[0]))
	result := map[string]interface{}{}
	err = arg.UnpackIntoMap(result, tx.ConstantResult[0])
	fmt.Printf("\nUnpack Result ->>> %+v\n\n", result)
	require.Nil(t, err)
}

func TestGetAccount2(t *testing.T) {
	conn := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := conn.Start(grpc.WithInsecure())
	require.Nil(t, err)

	tx, err := conn.GetAccountDetailed("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b")
	require.Nil(t, err)
	fmt.Printf("%v", tx)
}

func TestGetAccountMigrationContract(t *testing.T) {
	conn := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := conn.Start(grpc.WithInsecure())
	require.Nil(t, err)

	tx, err := conn.TriggerConstantContract("TX8h6Df74VpJsXF6sTDz1QJsq3Ec8dABc3",
		"TVoo62PAagTvNvZbB796YfZ7dWtqpPNxnL",
		"frozenAmount(address)", `[{"address": "TX8h6Df74VpJsXF6sTDz1QJsq3Ec8dABc3"}]`)
	require.Nil(t, err)
	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)

	a, err := conn.GetContractABI("TVoo62PAagTvNvZbB796YfZ7dWtqpPNxnL")
	require.Nil(t, err)
	arg, err := abi.GetParser(a, "frozenAmount")
	require.Nil(t, err)
	fmt.Printf("\nContractABI ->>> %+v\n\n", a)
	fmt.Printf("\nResult ->>> %s\n\n", hex.EncodeToString(tx.ConstantResult[0]))

	result := map[string]interface{}{}
	err = arg.UnpackIntoMap(result, tx.ConstantResult[0])
	fmt.Println(result["amount"].(*big.Int).Int64())
	require.Nil(t, err)
}

// TestGetEnergyPrices tests the GetEnergyPrices function
func TestGetEnergyPrices(t *testing.T) {
	conn := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := conn.Start(grpc.WithInsecure())
	require.Nil(t, err)

	prices, err := conn.GetEnergyPrices()
	require.Nil(t, err)

	// Extract the last price from the prices string
	pricesStr := prices.Prices
	require.NotEmpty(t, pricesStr)

	pricesList := strings.Split(pricesStr, ",")
	require.NotEmpty(t, pricesList)

	// Get the last price component
	lastPriceComponent := pricesList[len(pricesList)-1]
	require.NotEmpty(t, lastPriceComponent)

	// Extract the price value from the last component
	lastPriceParts := strings.Split(lastPriceComponent, ":")
	require.Len(t, lastPriceParts, 2)

	lastPriceValue := lastPriceParts[1]

	// Ensure the last price value is "420"
	require.Equal(t, "420", lastPriceValue)
}

// TestGetBandwidthPrices tests the GetBandwidthPrices function
func TestGetBandwidthPrices(t *testing.T) {
	conn := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := conn.Start(grpc.WithInsecure())
	require.Nil(t, err)

	prices, err := conn.GetBandwidthPrices()
	require.Nil(t, err)

	// Assert prices are not empty
	pricesStr := prices.Prices
	require.NotEmpty(t, pricesStr)

	// Further validation (e.g., checking format)
	pricesList := strings.Split(pricesStr, ",")
	require.NotEmpty(t, pricesList)

	// checking that each price component has a valid format
	for _, priceComponent := range pricesList {
		parts := strings.Split(priceComponent, ":")
		require.Len(t, parts, 2)
		// We could add more checks here, like validating the timestamp and price values
	}
}
