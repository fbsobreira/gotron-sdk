package models

import "github.com/fbsobreira/gotron/common/global"

type NumberMessage struct {
	Num int64
}

func GetNextMaintenanceTime() (NumberMessage, error) {
	var resultNextMaintenanceTime NumberMessage

	grpcNextMaintenanceTime, err := global.TronClient.GetNextMaintenanceTime()
	if err != nil {
		return resultNextMaintenanceTime, err
	}
	resultNextMaintenanceTime.Num = grpcNextMaintenanceTime.Num
	return resultNextMaintenanceTime, nil
}

func GetTotalTransaction() NumberMessage {
	grpcTotalTransaction := global.TronClient.TotalTransaction()

	var resultTotalTransaction NumberMessage
	resultTotalTransaction.Num = grpcTotalTransaction.Num

	return resultTotalTransaction
}
