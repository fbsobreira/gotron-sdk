package models

import "github.com/sasaxie/go-client-api/common/global"

type AccountNetMessage struct {
	FreeNetUsed    int64
	FreeNetLimit   int64
	NetUsed        int64
	NetLimit       int64
	AssetNetUsed   map[string]int64
	AssetNetLimit  map[string]int64
	TotalNetLimit  int64
	TotalNetWeight int64
}

func GetAccountNet(address string) AccountNetMessage {
	grpcAccountNet := global.TronClient.GetAccountNet(address)

	var resultAccountNet AccountNetMessage

	if grpcAccountNet == nil {
		return resultAccountNet
	}

	resultAccountNet.FreeNetUsed = grpcAccountNet.FreeNetUsed
	resultAccountNet.FreeNetLimit = grpcAccountNet.FreeNetLimit
	resultAccountNet.NetUsed = grpcAccountNet.NetUsed
	resultAccountNet.NetLimit = grpcAccountNet.NetLimit
	resultAccountNet.AssetNetUsed = grpcAccountNet.AssetNetUsed
	resultAccountNet.AssetNetLimit = grpcAccountNet.AssetNetLimit
	resultAccountNet.TotalNetLimit = grpcAccountNet.TotalNetLimit
	resultAccountNet.TotalNetWeight = grpcAccountNet.TotalNetWeight

	return resultAccountNet
}
