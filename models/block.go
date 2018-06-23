package models

import (
	"github.com/sasaxie/go-client-api/common/base58"
	"github.com/sasaxie/go-client-api/common/global"
	"github.com/sasaxie/go-client-api/common/hexutil"
)

type Block struct {
	Transactions []Transaction
	BlockHeader  BlockHeader
}

type Transaction struct {
	RawData   TransactionRaw
	Signature []string
	Ret       []Result
}

type TransactionRaw struct {
	RefBlockBytes string
	RefBlockNum   int64
	RefBlockHash  string
	Expiration    int64
	Auths         []Acuthrity
	Data          string
	Contract      []Contract
	Scripts       string
	Timestamp     int64
}

type Acuthrity struct {
	Account        AccountId
	PermissionName string
}

type AccountId struct {
	Name    string
	Address string
}

type Contract struct {
	Type         string
	Parameter    interface{}
	Provider     string
	ContractName string
}

type Result struct {
	Fee int64
	Ret string
}

type BlockHeader struct {
	RawData          BlockHeaderRaw
	WitnessSignature string
}

type BlockHeaderRaw struct {
	Timestamp      int64
	TxTrieRoot     string
	ParentHash     string
	Number         int64
	WitnessId      int64
	WitnessAddress string
}

func GetNowBlock() Block {
	var nowBlock Block

	grpcNowBlock := global.TronClient.GetNowBlock()

	nowBlock.Transactions = make([]Transaction, 0)

	for _, t := range grpcNowBlock.Transactions {
		var transaction Transaction

		if t.RawData != nil {
			transaction.RawData.RefBlockBytes = hexutil.Encode(t.RawData.RefBlockBytes)
			transaction.RawData.RefBlockNum = t.RawData.RefBlockNum
			transaction.RawData.RefBlockHash = hexutil.Encode(t.RawData.RefBlockHash)
			transaction.RawData.Expiration = t.RawData.Expiration

			transaction.RawData.Auths = make([]Acuthrity, 0)
			for _, a := range t.RawData.Auths {
				var auth Acuthrity

				var accountId AccountId
				accountId.Name = string(a.Account.Name)
				accountId.Address = base58.EncodeCheck(a.Account.Address)

				auth.Account = accountId

				auth.PermissionName = string(a.PermissionName)

				transaction.RawData.Auths = append(transaction.RawData.Auths,
					auth)
			}

			transaction.RawData.Data = string(t.RawData.Data)

			transaction.RawData.Contract = make([]Contract, 0)
			for _, c := range t.RawData.Contract {
				var contract Contract
				contract.Type = c.Type.String()
				contract.Parameter = c.Parameter
				contract.Provider = string(c.Provider)
				contract.ContractName = string(c.ContractName)

				transaction.RawData.Contract = append(transaction.RawData.
					Contract, contract)
			}

			transaction.RawData.Scripts = string(t.RawData.Scripts)
			transaction.RawData.Timestamp = t.RawData.Timestamp
		}

		transaction.Signature = make([]string, 0)
		for _, s := range t.Signature {
			transaction.Signature = append(transaction.Signature, hexutil.Encode(s))
		}

		transaction.Ret = make([]Result, 0)
		for _, r := range t.Ret {
			var result Result
			result.Ret = string(r.Ret)
			result.Fee = r.Fee
			transaction.Ret = append(transaction.Ret, result)
		}

		nowBlock.Transactions = append(nowBlock.Transactions, transaction)
	}

	if grpcNowBlock.BlockHeader != nil {
		if grpcNowBlock.BlockHeader.RawData != nil {
			nowBlock.BlockHeader.RawData.Timestamp = grpcNowBlock.
				BlockHeader.RawData.Timestamp

			nowBlock.BlockHeader.RawData.TxTrieRoot = hexutil.Encode(grpcNowBlock.
				BlockHeader.RawData.TxTrieRoot)

			nowBlock.BlockHeader.RawData.ParentHash = hexutil.Encode(grpcNowBlock.
				BlockHeader.RawData.ParentHash)

			nowBlock.BlockHeader.RawData.Number = grpcNowBlock.
				BlockHeader.RawData.Number

			nowBlock.BlockHeader.RawData.WitnessId = grpcNowBlock.
				BlockHeader.RawData.WitnessId

			nowBlock.BlockHeader.RawData.WitnessAddress = base58.EncodeCheck(grpcNowBlock.
				BlockHeader.RawData.WitnessAddress)
		}

		nowBlock.BlockHeader.WitnessSignature = hexutil.Encode(grpcNowBlock.
			BlockHeader.WitnessSignature)
	}

	return nowBlock
}

func GetBlockByNum(num int64) Block {
	grpcBlock := global.TronClient.GetBlockByNum(num)

	var block Block

	if grpcBlock == nil {
		return block
	}

	block.Transactions = make([]Transaction, 0)

	for _, t := range grpcBlock.Transactions {
		var transaction Transaction

		if t.RawData != nil {
			transaction.RawData.RefBlockBytes = hexutil.Encode(t.RawData.RefBlockBytes)
			transaction.RawData.RefBlockNum = t.RawData.RefBlockNum
			transaction.RawData.RefBlockHash = hexutil.Encode(t.RawData.RefBlockHash)
			transaction.RawData.Expiration = t.RawData.Expiration

			transaction.RawData.Auths = make([]Acuthrity, 0)
			for _, a := range t.RawData.Auths {
				var auth Acuthrity

				var accountId AccountId
				accountId.Name = string(a.Account.Name)
				accountId.Address = base58.EncodeCheck(a.Account.Address)

				auth.Account = accountId

				auth.PermissionName = string(a.PermissionName)

				transaction.RawData.Auths = append(transaction.RawData.Auths,
					auth)
			}

			transaction.RawData.Data = string(t.RawData.Data)

			transaction.RawData.Contract = make([]Contract, 0)
			for _, c := range t.RawData.Contract {
				var contract Contract
				contract.Type = c.Type.String()
				contract.Parameter = c.Parameter
				contract.Provider = string(c.Provider)
				contract.ContractName = string(c.ContractName)

				transaction.RawData.Contract = append(transaction.RawData.
					Contract, contract)
			}

			transaction.RawData.Scripts = string(t.RawData.Scripts)
			transaction.RawData.Timestamp = t.RawData.Timestamp
		}

		transaction.Signature = make([]string, 0)
		for _, s := range t.Signature {
			transaction.Signature = append(transaction.Signature, hexutil.Encode(s))
		}

		transaction.Ret = make([]Result, 0)
		for _, r := range t.Ret {
			var result Result
			result.Ret = string(r.Ret)
			result.Fee = r.Fee
			transaction.Ret = append(transaction.Ret, result)
		}

		block.Transactions = append(block.Transactions, transaction)
	}

	if grpcBlock.BlockHeader != nil {
		if grpcBlock.BlockHeader.RawData != nil {
			block.BlockHeader.RawData.Timestamp = grpcBlock.
				BlockHeader.RawData.Timestamp

			block.BlockHeader.RawData.TxTrieRoot = hexutil.Encode(grpcBlock.
				BlockHeader.RawData.TxTrieRoot)

			block.BlockHeader.RawData.ParentHash = hexutil.Encode(grpcBlock.
				BlockHeader.RawData.ParentHash)

			block.BlockHeader.RawData.Number = grpcBlock.
				BlockHeader.RawData.Number

			block.BlockHeader.RawData.WitnessId = grpcBlock.
				BlockHeader.RawData.WitnessId

			block.BlockHeader.RawData.WitnessAddress = base58.EncodeCheck(grpcBlock.
				BlockHeader.RawData.WitnessAddress)
		}

		block.BlockHeader.WitnessSignature = hexutil.Encode(grpcBlock.
			BlockHeader.WitnessSignature)
	}

	return block
}
