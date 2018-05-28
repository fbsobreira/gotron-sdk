## go-client-api

> The Golang java-tron gRPC client

## Requirements

- Go 1.6 or higher

## Installation

First you need to install ProtocolBuffers 3.0.0-beta-3 or later.

```sh
mkdir tmp
cd tmp
git clone https://github.com/google/protobuf
cd protobuf
./autogen.sh
./configure
make
make check
sudo make install
```

Then, `go get -u` as usual the following packages:

```sh
go get -u github.com/golang/protobuf/protoc-gen-go
```

Update protocol:

```sh
git submodule update --remote
```

Example:

```sh
go get -u github.com/tronprotocol/go-client-api
go run program/client.go
```

## TODO

- [x] GetAccount
- [ ] CreateTransaction
- [ ] BroadcastTransaction
- [ ] UpdateAccount
- [ ] CreateAccount
- [ ] VoteWitnessAccount
- [ ] CreateAssetIssue
- [x] ListWitnesses
- [ ] UpdateWitness
- [ ] CreateWitness
- [ ] TransferAsset
- [ ] ParticipateAssetIssue
- [x] ListNodes
- [ ] GetAssetIssueList
- [ ] GetAssetIssueByAccount
- [ ] GetAssetIssueByName
- [x] GetNowBlock
- [ ] GetBlockByNum
- [ ] TotalTransaction