#!/bin/bash

INCLUDES="-I=./proto/tron -I/usr/lib -I./proto/googleapis"
FLAGS="--go_out=./pkg/proto --go_opt paths=source_relative --go-grpc_out=./pkg/proto --go-grpc_opt=paths=source_relative"

protoc ${INCLUDES} ${FLAGS} ./proto/tron/core/*.proto ./proto/tron/core/contract/*.proto ./proto/tron/api/*.proto
mkdir -p ./pkg/proto/util
protoc ${INCLUDES} -I=./proto/util --go_out=./pkg/proto/util --go_opt paths=source_relative ./proto/util/*.proto