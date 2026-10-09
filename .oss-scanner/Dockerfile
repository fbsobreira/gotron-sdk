# Build environment for the Anthropic OSS Scanner (https://red.anthropic.com/oss-scanner).
# The scanner builds this with the repository as build context, with network access, then audits the
# resulting image offline. Everything needed to build and test must therefore be fetched here.
FROM golang:1.26-bookworm

ENV DEBIAN_FRONTEND=noninteractive \
    CGO_ENABLED=1

# gcc comes with the golang image; libusb/libudev are needed by the Ledger HID transport (github.com/zondax/hid).
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates git make pkg-config libusb-1.0-0-dev libudev-dev \
    && rm -rf /var/lib/apt/lists/*

# Make the Go toolchain available in login shells too (they reset PATH).
RUN ln -s /usr/local/go/bin/* /usr/local/bin/

# scanner contract: the checkout lives inside the image, at /src
COPY . /src
WORKDIR /src

# Fetch every module now: the audit has no network access. Generated protobuf code is committed
# (pkg/proto), so the proto submodule and protoc are not required.
RUN go mod download

# Build the library and the tronctl CLI with debug info.
RUN go build -gcflags="all=-N -l" ./... \
    && go build -gcflags="all=-N -l" -o /src/bin/tronctl ./cmd/tronctl

# Pre-compile the unit tests (offline-safe) and run them, but do not let a failure abort the build.
# Integration tests need the Nile testnet and are excluded.
RUN go vet ./pkg/... ./cmd/... || echo "WARNING: go vet reported issues"
RUN go test -count=1 -timeout 300s $(go list ./... | grep -v -E '/pkg/proto/|/cmd') \
    || echo "WARNING: go test reported failures"
