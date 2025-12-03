module github.com/fbsobreira/gotron-sdk

go 1.24.0

// Retract all v2 versions - these were tagged incorrectly without /v2 in module path
retract (
	v2.3.0+incompatible
	v2.2.2+incompatible
	v2.2.0+incompatible
	v2.1.1+incompatible
	v2.1.0+incompatible
	v2.0.0+incompatible
)

require (
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de
	github.com/btcsuite/btcd/btcec/v2 v2.3.4
	github.com/deckarep/golang-set v1.8.0
	github.com/ethereum/go-ethereum v1.15.6
	github.com/fatih/color v1.18.0
	github.com/fatih/structs v1.1.0
	github.com/fbsobreira/go-bip39 v1.2.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pborman/uuid v1.2.1
	github.com/pkg/errors v0.9.1
	github.com/rjeczalik/notify v0.9.3
	github.com/shengdoushi/base58 v1.0.0
	github.com/spf13/cobra v1.9.1
	github.com/stretchr/testify v1.10.0
	github.com/zondax/hid v0.9.2
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.45.0
	golang.org/x/term v0.37.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250227231956-55c901821b1e
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.6
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250227231956-55c901821b1e // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
