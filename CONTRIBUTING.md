# Contributing to GoTRON SDK

Thank you for your interest in contributing to GoTRON SDK! This guide will help you get started.

## Development Setup

### Prerequisites

- Go 1.24 or higher
- Make
- Protocol Buffers compiler (only if regenerating protos)

### Clone and Build

```bash
git clone https://github.com/fbsobreira/gotron-sdk.git
cd gotron-sdk
make
```

### Run Tests

```bash
# Unit tests
make test

# Integration tests (requires network access)
make test-integration
```

### Linting and Formatting

```bash
# Run linter
make lint

# Run goimports
make goimports

# Tidy modules
make tidy
```

## How to Submit a Pull Request

1. **Fork** the repository on GitHub.
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/gotron-sdk.git
   ```
3. **Create a branch** for your change:
   ```bash
   git checkout -b feature/my-change
   ```
4. **Make your changes** and ensure they pass all checks:
   ```bash
   make goimports
   make lint
   make test
   ```
5. **Commit** your changes (see commit format below).
6. **Push** to your fork and open a Pull Request against `master`.

## Code Style

- Format all code with `gofmt` and `goimports` (`make goimports`).
- Follow standard Go conventions and idioms.
- Run `make lint` before submitting and fix any warnings.
- Keep changes focused — one logical change per PR.

## Commit Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <description>

[optional body]
```

**Types:** `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`, `perf`

**Examples:**
```text
feat(client): add GetAccountResource method
fix(address): handle empty base58 input
docs: update SDK usage examples
test(client): add mock tests for transfer
```

## Finding Work

Look for issues labeled [`good first issue`](https://github.com/fbsobreira/gotron-sdk/labels/good%20first%20issue) — these are great starting points for new contributors.

## Reporting Bugs

Open an issue on the [Issue Tracker](https://github.com/fbsobreira/gotron-sdk/issues) with:

- A clear description of the problem
- Steps to reproduce
- Expected vs. actual behavior
- Go version and OS

## License

By contributing, you agree that your contributions will be licensed under the [LGPL-3.0 License](LICENSE).
