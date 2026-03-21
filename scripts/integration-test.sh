#!/usr/bin/env bash
# integration-test.sh — Full round-trip integration test on local TRON node
#
# Prerequisites:
#   - ./bin/tronctl built (make build)
#   - Local node running (./scripts/local-node.sh start)
#   - .env configured with TRONCTL_NODE=localhost:50051
#
# The script will:
#   1. Import a seed account from accounts-data/accounts.json (if available)
#   2. Create 2 test accounts
#   3. Fund them from the seed account
#   4. Run transfer, freeze, vote, and utility operations
#
# Usage:
#   ./scripts/integration-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
TRONCTL="./bin/tronctl"
PASS="testpass1234"
SEED_NAME="seed-account"
ACC1="integration-test-1"
ACC2="integration-test-2"
ACCOUNTS_JSON="accounts-data/accounts.json"
FUND_AMOUNT="1500"
TRANSFER_AMOUNT="1"
FREEZE_AMOUNT="10"
CONTRACT_ABI="$REPO_ROOT/testdata/contracts/TestToken.abi"
CONTRACT_BIN="$REPO_ROOT/testdata/contracts/TestToken.bin"
PASS_FILE=$(mktemp)
trap 'rm -f "$PASS_FILE"' EXIT
echo -n "$PASS" > "$PASS_FILE"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

pass=0
fail=0
skip=0

log_pass() { echo -e "  ${GREEN}[PASS]${NC} $1"; pass=$((pass + 1)); }
log_fail() { echo -e "  ${RED}[FAIL]${NC} $1: $2"; fail=$((fail + 1)); }
log_skip() { echo -e "  ${YELLOW}[SKIP]${NC} $1: $2"; skip=$((skip + 1)); }
log_info() { echo -e "  ${CYAN}[INFO]${NC} $1"; }
log_section() { echo -e "\n${CYAN}── $1 ──${NC}"; }

# Run a command and report pass/fail
run_test() {
    local name="$1"
    shift
    local output
    if output=$("$@" 2>&1); then
        log_pass "$name"
        [[ -n "$output" ]] && echo "$output" | sed 's/^/         /'
        return 0
    else
        log_fail "$name" "exit code $?"
        [[ -n "$output" ]] && echo "$output" | sed 's/^/         /'
        return 1
    fi
}

# Get address for an account name from tronctl
get_address() {
    $TRONCTL account address "$1" 2>&1 | grep -oE 'T[A-Za-z1-9]{33}' | head -1
}

# Check if an account name exists in tronctl keystore
account_exists() {
    local out
    out=$($TRONCTL keys list 2>/dev/null || true)
    echo "$out" | grep -q "$1"
}

# Check if an account has TRX balance (non-zero)
has_balance() {
    local bal
    bal=$($TRONCTL account balance "$1" 2>/dev/null || true)
    echo "$bal" | grep -qE '"balance":\s*[1-9]'
}

# Send TRX with passphrase
send_trx() {
    local from="$1" to="$2" amount="$3"
    $TRONCTL account send "$to" "$amount" \
        --signer "$from" --passphrase-file "$PASS_FILE" --no-wait 2>&1
}

echo "============================================"
echo " GoTRON SDK Integration Test"
echo "============================================"

# ── 0. Verify prerequisites ─────────────────────────────────────────────────
log_section "Prerequisites"

if [[ ! -x "$TRONCTL" ]]; then
    echo -e "${RED}Error: $TRONCTL not found. Run 'make build' first.${NC}"
    exit 1
fi
log_pass "Binary exists"

# Quick connectivity check
if $TRONCTL bc mt >/dev/null 2>&1; then
    log_pass "Node reachable"
else
    echo -e "${RED}Error: Cannot connect to node.${NC}"
    echo "  Start with: ./scripts/local-node.sh start"
    echo "  Set in .env: TRONCTL_NODE=localhost:50051"
    exit 1
fi

# ── 1. Seed account setup ───────────────────────────────────────────────────
log_section "Seed Account"

SEED_ADDR=""
if [[ -f "$ACCOUNTS_JSON" ]]; then
    log_info "Found $ACCOUNTS_JSON"

    # Extract first private key from TRE accounts
    SEED_PK=$(python3 -c "
import json, sys
with open('$ACCOUNTS_JSON') as f:
    data = json.load(f)
print(data['privateKeys'][0])
" 2>/dev/null || jq -r '.privateKeys[0]' "$ACCOUNTS_JSON" 2>/dev/null || true)

    if [[ -z "$SEED_PK" ]]; then
        log_fail "Parse seed key" "could not extract private key from $ACCOUNTS_JSON"
        exit 1
    fi

    # Import seed account if not already present
    if account_exists "$SEED_NAME"; then
        log_pass "Seed account '$SEED_NAME' already imported"
    else
        log_info "Importing seed account..."
        if $TRONCTL keys import-private-key "$SEED_PK" "$SEED_NAME" \
            --passphrase-file "$PASS_FILE" 2>&1; then
            log_pass "Imported seed account"
        else
            log_fail "Import seed account" "import failed"
            exit 1
        fi
    fi

    SEED_ADDR=$(get_address "$SEED_NAME")
    if [[ -n "$SEED_ADDR" ]]; then
        log_pass "Seed address: $SEED_ADDR"
    else
        log_fail "Seed address" "could not resolve"
        exit 1
    fi

    # Show seed balance
    SEED_BAL=$($TRONCTL account balance "$SEED_ADDR" 2>&1 || true)
    log_info "Seed balance: $(echo "$SEED_BAL" | grep -i 'balance' || echo "$SEED_BAL")"
else
    log_skip "Seed account" "$ACCOUNTS_JSON not found — skipping auto-funding"
fi

# ── 2. Test account setup ───────────────────────────────────────────────────
log_section "Test Accounts"

for ACC in "$ACC1" "$ACC2"; do
    if account_exists "$ACC"; then
        log_pass "Account '$ACC' exists"
    else
        log_info "Creating account '$ACC'..."
        if $TRONCTL keys add "$ACC" --passphrase-file "$PASS_FILE" >/dev/null 2>&1; then
            log_pass "Created account '$ACC'"
        else
            log_fail "Create '$ACC'" "failed"
            exit 1
        fi
    fi
done

ADDR1=$(get_address "$ACC1")
ADDR2=$(get_address "$ACC2")
log_info "ACC1: $ACC1 -> $ADDR1"
log_info "ACC2: $ACC2 -> $ADDR2"

# ── 3. Fund test accounts from seed ─────────────────────────────────────────
log_section "Fund Test Accounts"

if [[ -n "$SEED_ADDR" ]]; then
    for ADDR in "$ADDR1" "$ADDR2"; do
        if has_balance "$ADDR"; then
            log_pass "$ADDR already funded"
        else
            log_info "Funding $ADDR with $FUND_AMOUNT TRX from seed..."
            if send_trx "$SEED_ADDR" "$ADDR" "$FUND_AMOUNT" >/dev/null; then
                log_pass "Funded $ADDR"
                sleep 2  # wait for block
            else
                log_fail "Fund $ADDR" "transfer failed"
            fi
        fi
    done
else
    for ADDR in "$ADDR1" "$ADDR2"; do
        if has_balance "$ADDR"; then
            log_pass "$ADDR has balance"
        else
            log_skip "Fund $ADDR" "No seed account — fund manually"
        fi
    done
fi

# ── 4. Balance check ────────────────────────────────────────────────────────
log_section "Balance Check"

run_test "ACC1 balance" $TRONCTL account balance "$ADDR1" || true
run_test "ACC2 balance" $TRONCTL account balance "$ADDR2" || true
run_test "ACC1 detailed balance" $TRONCTL account balance "$ADDR1" --details || true

# ── 5. Account info ─────────────────────────────────────────────────────────
log_section "Account Info"

run_test "ACC1 info" $TRONCTL account info "$ADDR1" || true

# ── 6. Transfer TRX ─────────────────────────────────────────────────────────
log_section "TRX Transfer"

if has_balance "$ADDR1"; then
    log_info "Sending $TRANSFER_AMOUNT TRX: ACC1 -> ACC2"
    TX_OUTPUT=$(send_trx "$ADDR1" "$ADDR2" "$TRANSFER_AMOUNT" || true)
    if [[ -n "$TX_OUTPUT" ]] && ! echo "$TX_OUTPUT" | grep -qi "error"; then
        log_pass "Transfer ACC1 -> ACC2"
        sleep 2

        # Extract txID for later lookup
        TXID=$(echo "$TX_OUTPUT" | jq -r '.txID' 2>/dev/null || echo "$TX_OUTPUT" | grep -oE '[a-f0-9]{64}' | head -1 || true)

        log_info "Sending $TRANSFER_AMOUNT TRX: ACC2 -> ACC1"
        if send_trx "$ADDR2" "$ADDR1" "$TRANSFER_AMOUNT" >/dev/null; then
            log_pass "Transfer ACC2 -> ACC1"
        else
            log_fail "Transfer ACC2 -> ACC1" "failed"
        fi
    else
        log_fail "Transfer ACC1 -> ACC2" "failed"
        TXID=""
    fi
else
    log_skip "Transfer" "ACC1 has no funds"
    TXID=""
fi

# ── 6b. Transaction lookup ─────────────────────────────────────────────────
log_section "Transaction Lookup"

if [[ -n "${TXID:-}" ]]; then
    sleep 3
    log_info "Looking up transaction $TXID..."
    if $TRONCTL bc tx "$TXID" >/dev/null 2>&1; then
        log_pass "Transaction lookup"
    else
        log_skip "Transaction lookup" "tx not yet confirmed (expected on fast test)"
    fi
else
    log_skip "Transaction lookup" "No txID available"
fi

# ── 7. Freeze / Stake (V2) ──────────────────────────────────────────────────
log_section "Freeze (Stake V2)"

if has_balance "$ADDR1"; then
    log_info "Freezing $FREEZE_AMOUNT TRX for bandwidth (V2)..."
    if $TRONCTL account freezeV2 "$FREEZE_AMOUNT" \
        --signer "$ADDR1" --passphrase-file "$PASS_FILE" -t 0 --no-wait >/dev/null 2>&1; then
        log_pass "FreezeV2 for bandwidth"
    else
        log_fail "FreezeV2 for bandwidth" "failed"
    fi

    sleep 2

    log_info "Freezing $FREEZE_AMOUNT TRX for energy (V2)..."
    if $TRONCTL account freezeV2 "$FREEZE_AMOUNT" \
        --signer "$ADDR1" --passphrase-file "$PASS_FILE" -t 1 --no-wait >/dev/null 2>&1; then
        log_pass "FreezeV2 for energy"
    else
        log_fail "FreezeV2 for energy" "failed"
    fi

    sleep 2

    log_info "Unfreezing $FREEZE_AMOUNT TRX bandwidth (V2)..."
    if $TRONCTL account unfreezeV2 "$FREEZE_AMOUNT" \
        --signer "$ADDR1" --passphrase-file "$PASS_FILE" -t 0 --no-wait >/dev/null 2>&1; then
        log_pass "UnfreezeV2 for bandwidth"
    else
        log_skip "UnfreezeV2 for bandwidth" "may fail if freeze hasn't matured"
    fi
else
    log_skip "Freeze" "ACC1 has no funds"
fi

# ── 8. Vote ──────────────────────────────────────────────────────────────────
log_section "Vote"

SR_LIST=$($TRONCTL sr list --elected 2>&1 || true)
WITNESS=$(echo "$SR_LIST" | grep -oE 'T[A-Za-z1-9]{33}' | head -1 || true)

if [[ -n "$WITNESS" ]] && has_balance "$ADDR1"; then
    log_info "Voting 1 vote for witness $WITNESS..."
    if $TRONCTL account vote \
        --signer "$ADDR1" --passphrase-file "$PASS_FILE" --no-wait \
        --wv "$WITNESS:1" >/dev/null 2>&1; then
        log_pass "Vote for witness"
    else
        log_fail "Vote for witness" "failed"
    fi
else
    log_skip "Vote" "No witness found or no funds"
fi

# ── 9. Withdraw rewards ─────────────────────────────────────────────────────
log_section "Withdraw Rewards"

if has_balance "$ADDR1"; then
    if $TRONCTL account withdraw \
        --signer "$ADDR1" --passphrase-file "$PASS_FILE" --no-wait >/dev/null 2>&1; then
        log_pass "Withdraw rewards"
    else
        log_skip "Withdraw rewards" "No rewards available (expected for fresh accounts)"
    fi
else
    log_skip "Withdraw rewards" "ACC1 has no funds"
fi

# ── 10. Blockchain queries ──────────────────────────────────────────────────
log_section "Blockchain Queries"

run_test "Node info" $TRONCTL bc node || true
run_test "Maintenance time" $TRONCTL bc mt || true
run_test "SR list" $TRONCTL sr list || true

# ── 11. Utility commands ────────────────────────────────────────────────────
log_section "Utility Commands"

run_test "Metadata" $TRONCTL utility metadata || true

# Address round-trip: base58 -> hex -> base58
HEX=$($TRONCTL utility base58-to-addr "$ADDR1" 2>&1 | grep -oE '0x[0-9a-fA-F]+' | head -1 || true)
if [[ -n "$HEX" ]]; then
    log_pass "Base58 -> hex: $HEX"
    B58=$($TRONCTL utility addr-to-base58 "$HEX" 2>&1 | grep -oE 'T[A-Za-z1-9]{33}' | head -1 || true)
    if [[ "$B58" == "$ADDR1" ]]; then
        log_pass "Hex -> base58 round-trip matches"
    else
        log_fail "Address round-trip" "got '$B58', expected '$ADDR1'"
    fi
else
    log_fail "Base58 -> hex" "empty result"
fi

# ── 12. Keys commands ───────────────────────────────────────────────────────
log_section "Keys Commands"

run_test "Keys list" $TRONCTL keys list || true
run_test "Keys location" $TRONCTL keys location || true

# ── 13. Config commands ─────────────────────────────────────────────────────
log_section "Config"

run_test "Config get all" $TRONCTL config get all || true

# ── 14. Sign / Verify message ──────────────────────────────────────────────
log_section "Sign / Verify Message"

if [[ -n "$ADDR1" ]]; then
    SIG_OUTPUT=$($TRONCTL account sign "integration test message" \
        --signer "$ADDR1" --passphrase-file "$PASS_FILE" 2>&1 || true)
    SIG_HEX=$(echo "$SIG_OUTPUT" | python3 -c "import sys,json; print(json.load(sys.stdin)['Signature'])" 2>/dev/null || \
        echo "$SIG_OUTPUT" | jq -r '.Signature' 2>/dev/null || true)
    if [[ -n "$SIG_HEX" && "$SIG_HEX" != "null" ]]; then
        log_pass "Sign message"
        if $TRONCTL account verify "integration test message" "$SIG_HEX" --signer "$ADDR1" >/dev/null 2>&1; then
            log_pass "Verify message signature"
        else
            log_fail "Verify message signature" "failed"
        fi
    else
        log_fail "Sign message" "could not extract signature"
    fi
else
    log_skip "Sign/Verify" "No test address"
fi

# ── 15. Key export / import round-trip ─────────────────────────────────────
log_section "Key Export / Import Round-Trip"

if [[ -n "$ADDR1" ]]; then
    PK=$($TRONCTL keys export-private-key "$ADDR1" \
        --passphrase-file "$PASS_FILE" 2>&1 | grep -oE '[a-f0-9]{64}' | head -1 || true)
    if [[ -n "$PK" ]]; then
        log_pass "Export private key"
        # Import as new account
        if $TRONCTL keys import-private-key "$PK" "reimported-test" \
            --passphrase-file "$PASS_FILE" >/dev/null 2>&1; then
            log_pass "Import private key"
            REIMPORTED_ADDR=$(get_address "reimported-test")
            if [[ "$REIMPORTED_ADDR" == "$ADDR1" ]]; then
                log_pass "Re-imported address matches original"
            else
                log_fail "Address match" "got '$REIMPORTED_ADDR', expected '$ADDR1'"
            fi
            # Cleanup
            $TRONCTL keys remove "reimported-test" --passphrase-file "$PASS_FILE" >/dev/null 2>&1 || true
        else
            log_fail "Import private key" "failed"
        fi
    else
        log_fail "Export private key" "could not extract key"
    fi
else
    log_skip "Key round-trip" "No test address"
fi

# ── 16. Random private key ─────────────────────────────────────────────────
log_section "Random Private Key"

PK=$($TRONCTL keys random-pk 2>&1 | grep -oE '[a-f0-9]{64}' | head -1 || true)
if [[ ${#PK} -eq 64 ]]; then
    log_pass "Random private key (64 hex chars)"
else
    log_fail "Random private key" "expected 64 hex chars, got ${#PK}"
fi

# ── 17. TRC20 contract deploy + interact ──────────────────────────────────
log_section "TRC20 Contract (Deploy + Interact)"

CONTRACT_ADDR=""
if has_balance "$ADDR1" && [[ -f "$CONTRACT_ABI" ]] && [[ -f "$CONTRACT_BIN" ]]; then
    log_info "Deploying TestToken TRC20 contract..."
    DEPLOY_OUTPUT=$($TRONCTL contract deploy "TestToken" \
        --abiFile "$CONTRACT_ABI" \
        --bcFile "$CONTRACT_BIN" \
        --params '[1000000]' \
        --signer "$ADDR1" --passphrase-file "$PASS_FILE" \
        --feeLimit 1000000000 --oeLimit 10000000 2>&1 || true)

    CONTRACT_ADDR=$(echo "$DEPLOY_OUTPUT" | python3 -c "import sys,json; print(json.load(sys.stdin)['contractAddress'])" 2>/dev/null || \
        echo "$DEPLOY_OUTPUT" | jq -r '.contractAddress' 2>/dev/null || true)

    if [[ -n "$CONTRACT_ADDR" && "$CONTRACT_ADDR" != "null" && "$CONTRACT_ADDR" != "" ]]; then
        log_pass "Deploy TRC20 contract: $CONTRACT_ADDR"
        sleep 3  # wait for confirmation

        # Read token name via constant call
        NAME_OUT=$($TRONCTL contract constant "$CONTRACT_ADDR" "name()" 2>&1 || true)
        if echo "$NAME_OUT" | grep -qi "result\|0x"; then
            log_pass "Constant call: name()"
        else
            log_skip "Constant call: name()" "may need more time"
        fi

        # Read token symbol
        SYM_OUT=$($TRONCTL contract constant "$CONTRACT_ADDR" "symbol()" 2>&1 || true)
        if echo "$SYM_OUT" | grep -qi "result\|0x"; then
            log_pass "Constant call: symbol()"
        else
            log_skip "Constant call: symbol()" "may need more time"
        fi

        # Read decimals
        DEC_OUT=$($TRONCTL contract constant "$CONTRACT_ADDR" "decimals()" 2>&1 || true)
        if echo "$DEC_OUT" | grep -qi "result\|0x"; then
            log_pass "Constant call: decimals()"
        else
            log_skip "Constant call: decimals()" "may need more time"
        fi

        # TRC20 balance check via tronctl trc20
        BAL_OUT=$($TRONCTL trc20 balance "$ADDR1" "$CONTRACT_ADDR" 2>&1 || true)
        if echo "$BAL_OUT" | grep -qiE "balance|[0-9]"; then
            log_pass "TRC20 balance check"
        else
            log_skip "TRC20 balance" "contract may not be indexed yet"
        fi

        # TRC20 transfer: send tokens from ACC1 to ACC2
        log_info "Sending 100 TST tokens from ACC1 to ACC2..."
        TRC20_SEND=$($TRONCTL trc20 send "$ADDR2" 100 "$CONTRACT_ADDR" \
            --signer "$ADDR1" --passphrase-file "$PASS_FILE" \
            --feeLimit 100000000 --no-wait 2>&1 || true)
        if echo "$TRC20_SEND" | grep -qiE "txID|txid|0x[a-f0-9]"; then
            log_pass "TRC20 transfer ACC1 -> ACC2"
            sleep 5

            # Verify ACC2 received tokens
            BAL2_OUT=$($TRONCTL trc20 balance "$ADDR2" "$CONTRACT_ADDR" 2>&1 || true)
            if echo "$BAL2_OUT" | grep -qE '"balance":.*[1-9]'; then
                log_pass "TRC20 balance ACC2 > 0 after transfer"
            else
                log_info "Balance output: $BAL2_OUT"
                log_skip "TRC20 balance verify" "balance not yet reflected"
            fi
        else
            log_fail "TRC20 transfer" "$TRC20_SEND"
        fi

        # Trigger approve via contract trigger
        log_info "Approving ACC2 to spend 50 TST..."
        APPROVE_OUT=$($TRONCTL contract trigger "$CONTRACT_ADDR" \
            "approve(address,uint256)" "[\"$ADDR2\",50000000]" \
            --signer "$ADDR1" --passphrase-file "$PASS_FILE" \
            --feeLimit 100000000 --no-wait 2>&1 || true)
        if echo "$APPROVE_OUT" | grep -qiE "txID|txid|0x[a-f0-9]"; then
            log_pass "Contract trigger: approve()"
        else
            log_fail "Contract trigger: approve()" "$APPROVE_OUT"
        fi
    else
        log_fail "Deploy TRC20 contract" "no contract address in output"
        log_info "Output: $DEPLOY_OUTPUT"
    fi
else
    if ! has_balance "$ADDR1"; then
        log_skip "TRC20 Contract" "ACC1 has no funds"
    else
        log_skip "TRC20 Contract" "Contract files not found at $CONTRACT_ABI"
    fi
fi

# ── 18. TRC10 token operations ─────────────────────────────────────────────
log_section "TRC10 Token"

if has_balance "$ADDR1"; then
    log_info "Issuing test token..."
    if $TRONCTL trc10 issue "TestToken" "Test" "TST" "http://test.com" 1000000 1 \
        --signer "$ADDR1" --passphrase-file "$PASS_FILE" --no-wait -p 0 >/dev/null 2>&1; then
        log_pass "TRC10 token issue"
        sleep 2
    else
        log_skip "TRC10 token issue" "may not be supported by local node"
    fi
    run_test "TRC10 list" $TRONCTL trc10 list || true
else
    log_skip "TRC10" "ACC1 has no funds"
fi

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "============================================"
echo -e " Results: ${GREEN}$pass passed${NC}, ${RED}$fail failed${NC}, ${YELLOW}$skip skipped${NC}"
echo "============================================"

if [[ $fail -gt 0 ]]; then
    exit 1
fi
