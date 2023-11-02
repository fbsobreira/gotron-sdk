package transaction

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// "github.com/fbsobreira/gotron-sdk/pkg/client/transaction"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
)

func TestPrivateKeySign(t *testing.T) {
	conn := client.NewGrpcClient("grpc.shasta.trongrid.io:50051")

	err := conn.Start(grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to start grpc client: %v", err)
	}

	sender := "TFqcLfP3kGdMzQZ8BcPMuq6f9kwUKGxctS"
	receiver := "TBd5FeJHpQ3mh1mfrJqKWV53wCiyxrCEiG"
	contract := "TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs"
	amount := big.NewInt(100)
	feeLimit := int64(1e8)
	tx, err := conn.TRC20Send(sender, receiver, contract, amount, feeLimit)

	if err != nil {
		t.Fatalf("Failed to send TRC20 token: %v", err)
	}
	var ctrlr *Controller
	senderac, err := address.Base58ToAddress(sender)

	if err != nil {
		t.Fatalf("Failed to send TRC20 token: %v", err)
	}
	opts := func(c *Controller) {
		c.Behavior.ConfirmationWaitTime = 10
		c.Behavior.DryRun = false
		c.Behavior.SigningImpl = PrivateKey
	}

	//import private key for sender account here
	account := keystore.Account{Address: senderac, PrivateKey: "0000"}

	ctrlr = NewController(conn, nil, &account, tx.Transaction, opts)

	if err = ctrlr.ExecuteTransaction(); err != nil {

		t.Fatalf("Failed to send TRC20 token: %v", err)

	}

	addrResult := address.Address(ctrlr.Receipt.ContractAddress).String()

	result := make(map[string]interface{})
	result["txID"] = common.BytesToHexString(tx.GetTxid())
	result["blockNumber"] = ctrlr.Receipt.BlockNumber
	result["message"] = string(ctrlr.Result.Message)
	result["contractAddress"] = addrResult
	result["success"] = ctrlr.GetResultError() == nil
	result["resMessage"] = string(ctrlr.Receipt.ResMessage)
	result["receipt"] = map[string]interface{}{
		"fee":               ctrlr.Receipt.Fee,
		"energyFee":         ctrlr.Receipt.Receipt.EnergyFee,
		"energyUsage":       ctrlr.Receipt.Receipt.EnergyUsage,
		"originEnergyUsage": ctrlr.Receipt.Receipt.OriginEnergyUsage,
		"energyUsageTotal":  ctrlr.Receipt.Receipt.EnergyUsageTotal,
		"netFee":            ctrlr.Receipt.Receipt.NetFee,
		"netUsage":          ctrlr.Receipt.Receipt.NetUsage,
	}

	asJSON, _ := json.Marshal(result)
	fmt.Println(common.JSONPrettyFormat(string(asJSON)))

}
