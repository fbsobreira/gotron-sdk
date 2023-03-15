package common_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/stretchr/testify/assert"
)

func Test_DecodeBase58(t *testing.T) {

	invalidAddresses := []string{
		"TronEnergyioE1Z3ukeRv38sYkv5Jn55bL",
		"TronEnergyioNijNo8g3LF2ABKUAae6D2Z",
		"TronEnergyio3ZMcXA5hSjrTxaioKGgqyr",
	}

	validAddresses := []string{
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		"TVj7RNVHy6thbM7BWdSe9G6gXwKhjhdNZS",
		"THPvaUhoh2Qn2y9THCZML3H815hhFhn5YC",
	}

	for _, addr := range invalidAddresses {
		_, err := common.DecodeCheck(addr)
		assert.NotNil(t, err)
	}

	for _, addr := range validAddresses {
		_, err := common.DecodeCheck(addr)
		assert.Nil(t, err)
	}

}
