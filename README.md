## gotron ![](https://img.shields.io/badge/progress-80%25-yellow.svg)


Fork from [sasaxie/go-client-api](https://github.com/sasaxie/go-client-api)


## Requirements

- Go 1.13 or higher

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
go get -u github.com/fbsobreira/gotron
go run program/getnowblock.go -grpcAddress grpc.trongrid.io:50051
```
