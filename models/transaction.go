package models

type Transaction struct {
	RawData   *TransactionRaw
	Signature []string
	Ret       []*Result
}

type TransactionRaw struct {
	RefBlockBytes string
	RefBlockNum   int64
	RefBlockHash  string
	Expiration    int64
	Auths         []*Acuthrity
	Data          string
	Contract      []*Contract
	Scripts       string
	Timestamp     int64
}

type Acuthrity struct {
	Account        *AccountId
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
