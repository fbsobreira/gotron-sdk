package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/fbsobreira/gotron/common/crypto"
	"github.com/fbsobreira/gotron/service"
)

func main() {
	grpcAddress := flag.String("grpcAddress", "",
		"gRPC address: <IP:port> example: -grpcAddress localhost:50051")

	ownerPrivateKey := flag.String("ownerPrivateKey", "",
		"ownerPrivateKey: <account private key>")

	name := flag.String("name", "",
		"name: <new asset issue name>")

	description := flag.String("description", "",
		"description: <new asset issue description>")

	abbr := flag.String("abbr", "",
		"description: <new asset issue symbol>")

	urlStr := flag.String("url", "",
		"url: <new asset issue url>")

	totalSupply := flag.Int64("totalSupply", 0,
		"totalSupply: <new asset issue total supply>")

	startTime := flag.Int64("startTime", 0,
		"startTime: <new asset issue start time(ms)>")

	endTime := flag.Int64("endTime", 0,
		"endTime: <new asset issue end time(ms)>")

	freeAssetNetLimit := flag.Int64("freeAssetNetLimit", 0,
		"freeAssetNetLimit: <new asset issue free asset net limit>")

	publicFreeAssetNetLimit := flag.Int64("publicFreeAssetNetLimit", 0,
		"publicFreeAssetNetLimit: <new asset issue public free asset net"+
			" limit>")

	trxNum := flag.Int("trxNum", 0,
		"trxNum: <new asset issue free asset trx num>")

	icoNum := flag.Int("icoNum", 0,
		"icoNum: <new asset issue free asset ico num>")

	frozenSupply := flag.String("frozenSupply", "",
		"frozenSupply: <days:amount,days:amount,...>")

	flag.Parse()

	if (strings.EqualFold("", *ownerPrivateKey) && len(*ownerPrivateKey) == 0) ||
		(strings.EqualFold("", *name) && len(*name) == 0) ||
		(strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) ||
		(strings.EqualFold("", *description) && len(*description) == 0) ||
		(strings.EqualFold("", *abbr) && len(*abbr) > 4) ||
		(strings.EqualFold("", *abbr) && len(*abbr) == 0) ||
		(strings.EqualFold("", *urlStr) && len(*urlStr) == 0) ||
		(*totalSupply <= 0) ||
		(*startTime <= 0) ||
		(*endTime <= 0) ||
		(*freeAssetNetLimit < 0) ||
		(*publicFreeAssetNetLimit < 0) ||
		(*trxNum <= 0) ||
		(*icoNum <= 0) ||
		(strings.EqualFold("", *frozenSupply) && len(*frozenSupply) == 0) {
		log.Fatalln("./create-asset-issue " +
			"-grpcAddress localhost:50051 " +
			"-ownerPrivateKey <your private key> " +
			"-name <new asset issue name> " +
			"-description <new asset issue description> " +
			"-abbr <new asset issue symbol> " +
			"-url <new asset issue url> " +
			"-totalSupply <new asset issue total supply> " +
			"-startTime <start time> " +
			"-endTime <end time> " +
			"-freeAssetNetLimit <new asset issue free asset net limit> " +
			"-publicFreeAssetNetLimit <new asset issue public free asset net" +
			" limit> " +
			"-trxNum <new asset issue free asset trx num> " +
			"-icoNum <new asset issue free asset ico num> " +
			"-frozenSupply <amount:days,amount:days,...>")
	}

	frozenSupplySlice := strings.Split(*frozenSupply, ",")

	frozenSupplyMap := make(map[string]string)
	for _, value := range frozenSupplySlice {
		frozenSupplyKeyValue := strings.Split(value, ":")
		frozenSupplyMap[frozenSupplyKeyValue[0]] = frozenSupplyKeyValue[1]
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	key, err := crypto.GetPrivateKeyByHexString(*ownerPrivateKey)

	if err != nil {
		log.Fatalf("get private key by hex string error: %v", err)
	}

	result := client.CreateAssetIssue(key, *name, *description, *abbr, *urlStr,
		*totalSupply, *startTime, *endTime, *freeAssetNetLimit,
		*publicFreeAssetNetLimit, int32(*trxNum), int32(*icoNum), 0,
		frozenSupplyMap)

	fmt.Printf("result: %v\n", result)
}
