#!/bin/bash
set -e

HOOK_DIR=$(git rev-parse --git-path hooks)
REPO_ROOT=$(git rev-parse --show-toplevel)
HOOKS_DIR="$REPO_ROOT/.githooks"

# Make sure hooks are executable
chmod +x "$HOOKS_DIR/pre-commit"
chmod +x "$HOOKS_DIR/commit-msg"
chmod +x "$HOOKS_DIR/prepare-commit-msg"

# Create symlinks to hooks
ln -sf "$HOOKS_DIR/pre-commit" "$HOOK_DIR/pre-commit"
ln -sf "$HOOKS_DIR/commit-msg" "$HOOK_DIR/commit-msg"
ln -sf "$HOOKS_DIR/prepare-commit-msg" "$HOOK_DIR/prepare-commit-msg"

echo "✅ Git hooks installed successfully!"
echo "Pre-commit hook will run: format, lint and test on staged files"
echo "Commit-msg hook will enforce conventional commit format"
echo "Prepare-commit-msg hook will provide a commit message template"

# Check for required tools
echo "Checking required tools..."

if ! command -v goimports &> /dev/null; then
    echo "⚠️ goimports not found. Please install with:"
    echo "  go install golang.org/x/tools/cmd/goimports@latest"
fi

if ! command -v golangci-lint &> /dev/null; then
    echo "⚠️ golangci-lint not found. Please install with:"
    echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin"
fi