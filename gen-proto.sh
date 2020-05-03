#!/bin/bash
protoc -I=./proto/tron -I/usr/lib -I./proto/googleapis --go_out=plugins=grpc,paths=source_relative:./pkg/proto ./proto/tron/core/*.proto ./proto/tron/core/contract/*.proto 
protoc -I=./proto/tron -I/usr/lib -I./proto/googleapis --go_out=plugins=grpc,paths=source_relative:./pkg/proto ./proto/tron/api/*.proto
mkdir -p ./pkg/proto/util
protoc -I=./proto/tron -I./proto/util -I/usr/lib -I./proto/googleapis --go_out=plugins=grpc,paths=source_relative:./pkg/proto/util ./proto/util/*.proto