SHELL := /bin/bash
version := $(shell git rev-list --count HEAD)
commit := $(shell git describe --always --long --dirty)
built_at := $(shell date +%FT%T%z)
built_by := ${USER}@cryptochain.network
BUILD_TARGET := tronctl

flags := -gcflags="all=-N -l -c 2"
ldflags := -X main.version=v${version} -X main.commit=${commit}
ldflags += -X main.builtAt=${built_at} -X main.builtBy=${built_by}
cli := ./bin/${BUILD_TARGET}
uname := $(shell uname)


.PHONY: all windows run debug install clean test lint

all:
	$(env) go build -o $(cli) -ldflags="$(ldflags)" cmd/main.go

windows:
	$(env) GOOS=windows GOARCH=amd64 go build -o $(cli).exe -ldflags="$(ldflags)" cmd/main.go

run:
	$(env) go run -ldflags="$(ldflags)" cmd/main.go

debug:
	$(env) go build $(flags) -o $(cli) -ldflags="$(ldflags)" cmd/main.go

install: all
	cp $(cli) ~/.local/bin

clean:
	@rm -f $(cli)
	@rm -rf ./bin

# Test target for CI
test:
	$(env) go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Lint target for CI
lint:
	$(env) golangci-lint run --timeout=5m