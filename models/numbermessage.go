package models

import "github.com/sasaxie/go-client-api/common/global"

type NumberMessage struct {
	Num int64
}

func GetNextMaintenanceTime() NumberMessage {
	grpcNextMaintenanceTime := global.TronClient.GetNextMaintenanceTime()

	var resultNextMaintenanceTime NumberMessage
	resultNextMaintenanceTime.Num = grpcNextMaintenanceTime.Num

	return resultNextMaintenanceTime
}

func GetTotalTransaction() NumberMessage {
	grpcTotalTransaction := global.TronClient.TotalTransaction()

	var resultTotalTransaction NumberMessage
	resultTotalTransaction.Num = grpcTotalTransaction.Num

	return resultTotalTransaction
}
