## go-client-api ![](https://img.shields.io/badge/progress-21%25-red.svg)

<strong style="color: red;">**WARNING: This is an incomplete work-in-progress.
yet**</strong>

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
go get -u github.com/sasaxie/go-client-api
go run program/getnowblock.go -grpcAddress localhost:50051
```