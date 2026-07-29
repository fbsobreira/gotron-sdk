# CLAUDE.md

Guidance for Claude Code (claude.ai/code) when working in this repository.

## Project Overview

GoTRON SDK — a Go SDK and CLI (`tronctl`) for the TRON blockchain over gRPC.
Module path: `github.com/fbsobreira/gotron-sdk`. Minimum Go 1.25 (`go 1.25.0`).

As a library, the `go` directive is a hard floor for every downstream consumer — raise it only when a
dependency genuinely requires it, and never pin a patch version there. Deliberately **no `toolchain`
directive**: it would silently upgrade the CI matrix so every job ran the same Go release. Release
binaries get a patched compiler via `go-version: '>=1.26.5'` in `release.yaml` instead, and the test
and build matrices set `GOTOOLCHAIN: local` so each job really tests its named version.

Write **GoTron** or **GoTRON** — never "Gotron".

## Commands

```bash
make                    # Build to ./bin/tronctl
make run                # go run ./cmd/tronctl
make debug              # Build with debug symbols (-gcflags all=-N -l)
make install            # Copy ./bin/tronctl to ~/.local/bin
make build-windows      # Cross-compile to ./bin/tronctl.exe
make clean              # Remove ./bin

make test               # Unit tests: -race -shuffle=on, with coverage
make test-integration   # Live Nile testnet, -tags=integration (network required)
make lint               # golangci-lint
make goimports          # Format (skips *.pb.go and vendor/)
make tidy               # Verify go.mod/go.sum are tidy

make hooks              # Install .githooks (do this once per clone)
./gen-proto.sh          # Regenerate pkg/proto from proto/ definitions
```

The binary lands in `./bin/tronctl` — not the repo root.

## Architecture

CLI (`cmd/`) and SDK (`pkg/`) are cleanly separated; the CLI is a consumer of the SDK.

**CLI** — `cmd/tronctl/` (entry point, version ldflags) and `cmd/subcommands/` (Cobra commands).
`root.go` holds global flags and `PersistentPreRunE`; `runtime.go` resolves config precedence and
dials the node; `config.go` manages the YAML config at `$HOME/.config/tronctl`.

**Client** — `pkg/client/` is the central gRPC client (one file per domain: `account.go`, `trc20.go`,
`contracts.go`, `resources.go`, `bank.go`, `network.go`, …). `pkg/client/transaction/controller.go`
drives the build → sign → broadcast → confirm sequence.

**Transactions** — `pkg/txbuilder/` (fluent builder), `pkg/txcore/` (send and confirmation loop),
`pkg/txresult/` (result decoding).

**Keys and signing** — `pkg/keystore/` (Ethereum V3 JSON format, scrypt), `pkg/keys/` and
`pkg/keys/hd/` (BIP39/BIP44), `pkg/mnemonic/`, `pkg/signer/` (Signer interface: private key, keystore,
ledger), `pkg/ledger/` (Nano S APDU driver), `pkg/store/` (account-name → keystore-path mapping).

**Contracts and encoding** — `pkg/abi/` (Solidity ABI pack/unpack), `pkg/contract/`,
`pkg/standards/trc20/` and `pkg/standards/trc20enc/`.

**Support** — `pkg/address/` and `pkg/common/` (Base58Check, hex, `numeric`, `decimals`),
`pkg/account/`, `pkg/tron/`.

**Generated** — `pkg/proto/`. Never edit by hand; run `./gen-proto.sh`.

Reference docs live in `docs/` (`cli-usage.md`, `sdk-usage.md`, `api-reference.md`, `examples.md`).

## TRON domain notes

- **Addresses** are Base58Check with a `0x41` prefix byte, rendered starting with `T`. Validate before
  use; conversion helpers are in `pkg/address/` and `pkg/common/base58.go`.
- **Amounts** are in SUN: 1 TRX = 1e6 SUN (`common.AmountDecimalPoint`). TRC20 amounts use the
  token's own `decimals()`, which is *not* always 18 — always query it, never assume.
- **TRC10 vs TRC20** — TRC10 is native to the protocol (asset IDs); TRC20 is contract-based (ABI
  calls). They have entirely different APIs.
- **Energy and bandwidth** are consumed per operation; `feeLimit` caps the spend on contract calls.
- **Witnesses (SRs)** produce blocks; `sr.go` and proposal/exchange commands cover governance.
- **Transactions are built by the node**, not locally: the client calls `CreateTransaction2` /
  `TriggerContract` and signs the returned protobuf. Treat that response as untrusted input.

## Gotchas

- **`make test` excludes `/cmd` and `/pkg/proto/`.** A green `make test` therefore says nothing about
  the CLI layer — check its coverage separately and verify CLI changes by running the binary.
- **`make test-integration` hits the live Nile testnet** and needs network plus funded fixtures. It
  will not run offline.
- **Config precedence:** CLI flags > environment > config file > defaults (resolved in
  `cmd/subcommands/runtime.go`). Config is YAML at `$HOME/.config/tronctl`.
- **Check the `WithTLS` default** in `cmd/subcommands/config.go` before assuming transport security;
  `--withTLS` or `TRONCTL_TLS=true` forces TLS on.
- **Several `cmd/subcommands` flag variables are package-level globals** (e.g. `feeLimit`) shared
  across commands. Re-registering one in a new command changes the default for the others.
- **`pkg/common/hexutils.go` `LeftPadBytes` returns the input unchanged** when it already exceeds the
  target length — it does not truncate. Validate widths before padding.

## Environment variables

| Variable | Effect |
|----------|--------|
| `GOTRON_SDK_DEBUG` | `true` enables debug output (same as `--verbose`) |
| `TRONCTL_NODE` | Default node endpoint |
| `TRONCTL_TLS` | `true`/`1` enables TLS |
| `TRONCTL_SIGNER` | Default signer address |
| `TRONCTL_KS_DIR` | Keystore directory |
| `TRONGRID_APIKEY` | TronGrid API key (`TRON-PRO-API-KEY` header) |

## Go conventions

- Handle every error explicitly — no blank `_` for error returns (`files, _ := os.ReadDir(...)` is a
  bug, not a shortcut).
- Wrap gRPC errors with context. Transaction errors carry TRON-specific result codes — surface them
  rather than collapsing them into a generic message.
- Prefer generated `GetX()` proto accessors over direct field access; they are nil-safe.
- Check `len()` before indexing anything that came off the wire, including `ConstantResult[0]`.
- Table-driven tests; keep interfaces to 1-3 methods.
- Run `make goimports` and `make lint` before committing.

## Adding features

1. New CLI command → `cmd/subcommands/`, following the existing registration pattern.
2. New client method → the matching domain file in `pkg/client/`.
3. Protocol change → edit `proto/`, then run `./gen-proto.sh`.
4. **Always add tests alongside new functionality.**

## Git workflow

Install hooks once per clone with `make hooks`. They enforce the rules below automatically:
`pre-commit` runs goimports, `go mod tidy`, golangci-lint, and tests for staged packages;
`commit-msg` validates the message format.

**Commit format:** `type(scope): description` (max 100 chars).
Valid types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`,
`revert`.

**Before every commit:** `make test`, `make lint`, `make`.

### Release process

1. `git tag v0.X.Y && git push origin v0.X.Y`
2. GitHub Actions runs tests → GoReleaser builds binaries → creates the GitHub Release → pushes the
   Homebrew formula.
3. Edit the release notes to this format:

```markdown
### 🔖 Release: `v0.X.Y` — Short Title

One-sentence summary of what this release is about.

#### ✨ Features

* **Feature name** — short description (#PR)

#### 🔒 Security

* **CVE or fix title** — description (#PR)

#### 🐛 Bug Fixes

* **Fix title** — description (#PR)

#### 🔨 Refactoring

* Description (#PR)

#### 🧪 Testing

* **Coverage or test improvement** (#PR)

#### 📦 Dependencies

* Updated `package` to vX.Y.Z (#PR)

#### 📖 Documentation

* Description (#PR)

**Full Changelog**: https://github.com/fbsobreira/gotron-sdk/compare/v0.PREV...v0.X.Y
```

Rules: skip empty sections; bold name, em-dash, description; link PRs as `(#number)`; always end with
the `**Full Changelog**` compare link. Emoji headers: ✨ Features, 🔒 Security, 🐛 Bug Fixes,
🔨 Refactoring, 🧪 Testing, 📦 Dependencies, 📖 Documentation.

## Do NOT

- Commit with failing tests
- Force-push without approval
- Delete files without confirming
- Edit `pkg/proto/` by hand (use `./gen-proto.sh`)
- Store secrets in code, or accept private keys as CLI arguments (they leak to shell history and `ps`)
- Silently ignore errors

## Working style

**Quick tasks (< 30 min):** just do them, no planning overhead.

**Complex features:** keep `.local/current-task.md` updated with goal, acceptance criteria, plan, and
progress, so work survives a `/clear`. Read it first when resuming.

**Delegation:** proactively use subagents for exploration, research, and review — anything reading
more than ~3 files, or producing output the user does not need verbatim. Don't ask permission;
delegate when it makes sense. Use specialized agent types where they fit (`Explore` for codebase
search, `Plan` for architecture) and run independent subagents in parallel. Keep direct edits, short
targeted reads, and interactive iteration in the main context.

**Context:** run `/context` to check token usage. When it fills up, save progress to
`.local/current-task.md` and tell the user before they `/clear`.

**Before declaring a task complete:**

- [ ] Tests pass (`make test`)
- [ ] Linter clean (`make lint`)
- [ ] Commit message follows the convention above (when committing)
- [ ] `.local/current-task.md` updated, if it exists

## Local files

`.local/` and `.claude/` are gitignored working directories — never commit their contents.
Personal, non-shared instructions belong in `CLAUDE.local.md` (also gitignored).

## Available slash commands

`/catchup` (restore branch context) · `/project-status` (project overview)
