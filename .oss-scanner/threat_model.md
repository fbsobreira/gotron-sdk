# Threat model: gotron-sdk

## What this project does and where untrusted input enters

gotron-sdk is a Go SDK and CLI (`tronctl`) for the TRON blockchain. Applications use it to build, sign and broadcast
transactions, manage keys and keystores, and talk to TRON nodes over gRPC. Downstream users include wallets, exchanges,
payment processors and bots, so a bug can mean lost or misdirected funds.

Treat these as adversarial:

- **Node responses.** Everything returned by a TRON node over gRPC (`pkg/client`, `pkg/proto`): transactions, blocks,
  accounts, contract call results, `constant_result`, `GetRawData`, events. A node can be malicious, buggy or
  man-in-the-middled. The client must not panic, index out of range, mis-sign, or accept forged data silently.
- **Keystore files and wallet imports.** JSON keystores (scrypt/PBKDF2, AES-CTR/CBC), mnemonics and private-key imports
  (`pkg/keystore`, `pkg/keys`, `pkg/mnemonic`). Files may be attacker-supplied.
- **Encoded values.** Base58/Base58Check and hex addresses, ABI-encoded data, ABI JSON definitions and event logs,
  TRC20 data (`pkg/address`, `pkg/common`, `pkg/abi`, `pkg/standards`).
- **Transaction content that gets signed.** What the caller asks to sign must match what is actually signed and
  broadcast (`pkg/txbuilder`, `pkg/txcore`, `pkg/signer`, `pkg/contract`).
- **CLI input and the release feed.** `tronctl` arguments and flags, its config files, and the release metadata and
  download used by `tronctl upgrade` (`cmd/subcommands`).

Not adversarial: the caller's own code, and the local machine's OS, filesystem permissions and memory (see out of scope).

## Components that matter most / least

Most important:

- `pkg/keystore`, `pkg/keys`, `pkg/mnemonic`, `pkg/signer`, `pkg/ledger`: key generation, storage, derivation and signing.
- `pkg/txbuilder`, `pkg/txcore`, `pkg/client` (transaction and contract paths), `pkg/abi`, `pkg/contract`: building and
  decoding transactions and contract calls. A mismatch between what the user intends and what is signed is the worst case.
- `pkg/address`, `pkg/common`: address and amount parsing. Wrong parsing sends funds to the wrong place or the wrong amount.
- `cmd/subcommands/upgrade.go`: it downloads release data and replaces the running binary.

Less important but in scope: the remaining read-only query helpers in `pkg/client`, `pkg/store` (local config), and CLI
output formatting.

Out of scope: `pkg/proto/**` (generated protobuf code), `proto/**` (vendored protocol definitions and googleapis),
`examples/`, `docs/`, test files and `testdata/`, and third-party dependencies (report those upstream unless gotron-sdk
uses them unsafely).

## How to exercise it

- Build the library with `go build ./...` and the CLI with `go build ./cmd/tronctl` (already built at `/src/bin/tronctl`).
- Unit tests: `go test ./pkg/...`. Fuzz targets exist in `pkg/abi`, `pkg/address` and `pkg/common`
  (`go test -fuzz=<Name> ./pkg/<dir>`).
- The test suite includes in-process mock gRPC servers (see `pkg/client/mock_test.go` and
  `pkg/client/untrusted_input_test.go`). Reuse that pattern for a hostile-node proof of concept.
- There is no network during the scan. Integration tests that need the Nile testnet cannot run, so reproduce issues
  with mock servers, crafted keystore files or direct function calls.
- Ledger paths can be exercised with the mock in `pkg/ledger/mock.go`.

## How you rate severity

Proposed rubric; maintainers may adjust.

- **Critical:** private key, mnemonic or passphrase disclosure; keystore decryption without the passphrase; signing or
  broadcasting a transaction that differs from what the caller asked for (wrong recipient, amount, contract call or
  permission); a remotely triggered flaw that causes loss of funds.
- **High:** a malicious node response or crafted input that causes silent mis-signing, wrong address or amount parsing,
  authentication bypass in multisig/permission handling, an unauthenticated update that runs attacker code in
  `tronctl upgrade`, or a memory-safety or code-execution issue reachable from untrusted input.
- **Medium:** remotely triggerable panics or unbounded resource use (memory, CPU, goroutines) from node responses or
  files; weak or misconfigured cryptographic parameters that remain exploitable with effort; secrets written to logs.
- **Low:** hardening gaps, information leaks with no secret material, and bugs that need unlikely local access.

A report needs a concrete path from untrusted input to the stated impact. Findings without a reproducer cap at medium.

## Report and patch expectations

- Include a minimal, runnable Go test or program that reproduces the issue offline, preferably as a `_test.go` file using
  the existing mock gRPC server pattern.
- Say which input is attacker-controlled and why the attacker can supply it.
- Propose a patch with a regression test. Match the existing style: small functions, errors returned, input validated
  before indexing or type assertions.
- One root cause per report. Group variants of the same missing check into one report.

## Anything to leave alone

- Plain `grpc.WithInsecure`/plaintext connections: the caller chooses the transport and dial options. Report only if the
  SDK weakens a transport the caller explicitly secured.
- The caller supplying a malicious private key, passphrase, or ABI to their own process. Local access to process memory
  or to the keystore passphrase is out of scope.
- Missing key zeroisation from Go memory, and memory-scraping attacks in general. Go gives no guarantee here.
- Denial of service from a caller deliberately passing huge arguments to its own SDK calls.
- Findings in third-party dependencies with no gotron-sdk-specific misuse.
- Style, lint and test-only issues. Panics reachable only from test code or `examples/`.
- Previously fixed classes of bugs: input validation before indexing/assertion, exact amount parsing, canonical ABI
  selectors, and not signing on `GetRawData` failure. Report regressions or new variants, not the original classes.
