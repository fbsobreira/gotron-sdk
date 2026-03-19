// Example: Transaction Builder + Contract Call Builder + TRC20
//
// This file demonstrates the new builder API introduced in #243.
// Use it as a reference or for recording SDK autocomplete demos.
//
// Usage:
//
//	go run ./examples/builder
package main

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/contract"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/fbsobreira/gotron-sdk/pkg/standards/trc20"
	"github.com/fbsobreira/gotron-sdk/pkg/txbuilder"
)

const (
	// Nile testnet
	grpcEndpoint    = "grpc.nile.trongrid.io:50051"
	usdtContract    = "TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj" // Nile USDT
	sampleAddress   = "TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"
	receiverAddress = "TYMwiDu22V6XG3yk6w9cTVBz48okKLRczh"
)

func main() {
	// --- Setup: connect to TRON node ---
	conn := connect()
	defer conn.Stop()

	// Create a signer from a private key
	s := newSigner()

	ctx := context.Background()

	// =====================================================
	// 1. TRANSACTION BUILDER — native TRON operations
	// =====================================================
	tx := txbuilder.New(conn)

	// Transfer TRX
	fmt.Println("=== Transfer TRX ===")
	transferTx := tx.Transfer(sampleAddress, receiverAddress, 1_000_000)
	printTx("Transfer 1 TRX", transferTx, ctx)

	// Transfer with memo
	fmt.Println("\n=== Transfer with memo ===")
	memoTx := tx.Transfer(sampleAddress, receiverAddress, 500_000,
		txbuilder.WithMemo("payment for coffee"))
	printTx("Transfer 0.5 TRX with memo", memoTx, ctx)

	// Freeze TRX for energy
	fmt.Println("\n=== Freeze for Energy ===")
	freezeTx := tx.FreezeV2(sampleAddress, 10_000_000, core.ResourceCode_ENERGY)
	printTx("Freeze 10 TRX for energy", freezeTx, ctx)

	// Delegate resources with lock
	fmt.Println("\n=== Delegate Energy with Lock ===")
	delegateTx := tx.DelegateResource(sampleAddress, receiverAddress, core.ResourceCode_ENERGY, 5_000_000).
		Lock(86400)
	printDelegateTx("Delegate 5 TRX energy, locked 86400 blocks", delegateTx, ctx)

	// Vote for witnesses
	fmt.Println("\n=== Vote for Witnesses ===")
	voteTx := tx.VoteWitness(sampleAddress).
		Vote("TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH", 100).
		Vote("TGj1Ej1qRzL9feLTLO73w2dbMardMoLJv2", 200)
	printVoteTx("Vote for 2 witnesses", voteTx, ctx)

	// Multi-sig: shared builder defaults
	fmt.Println("\n=== Multi-sig Builder ===")
	multiSigTx := txbuilder.New(conn, txbuilder.WithPermissionID(2))
	msTx := multiSigTx.Transfer(sampleAddress, receiverAddress, 100_000)
	printTx("Multi-sig transfer", msTx, ctx)

	// =====================================================
	// 2. CONTRACT CALL BUILDER — smart contract interactions
	// =====================================================
	fmt.Println("\n=== Contract Call: balanceOf ===")
	call := contract.New(conn, usdtContract).
		Method("balanceOf(address)").
		Params(fmt.Sprintf(`[{"address": "%s"}]`, sampleAddress))

	result, err := call.Call(ctx)
	if err != nil {
		fmt.Printf("  Call error: %v\n", err)
	} else {
		fmt.Printf("  Raw result: %x\n", result.RawResults[0])
	}

	// Contract call with pre-packed data
	fmt.Println("\n=== Contract Call: WithData ===")
	dataResult, err := contract.New(conn, usdtContract).
		WithData([]byte{0x06, 0xfd, 0xde, 0x03}). // name() selector
		Call(ctx)
	if err != nil {
		fmt.Printf("  WithData call error: %v\n", err)
	} else {
		fmt.Printf("  name() result: %x\n", dataResult.RawResults[0])
	}

	// Build a state-changing contract call
	fmt.Println("\n=== Contract Call: Build transfer ===")
	buildCall := contract.New(conn, usdtContract).
		Method("transfer(address,uint256)").
		From(sampleAddress).
		Params(fmt.Sprintf(`[{"address": "%s"},{"uint256": "1000000"}]`, receiverAddress)).
		Apply(contract.WithFeeLimit(100_000_000))

	builtTx, err := buildCall.Build(ctx)
	if err != nil {
		fmt.Printf("  Build error: %v\n", err)
	} else {
		fmt.Printf("  Built tx hash: %x\n", builtTx.GetTxid())
	}

	// =====================================================
	// 3. TRC20 — typed token wrapper
	// =====================================================
	token := trc20.New(conn, usdtContract)

	fmt.Println("\n=== TRC20: Token Info ===")
	info, err := token.Info(ctx)
	if err != nil {
		fmt.Printf("  Info error: %v\n", err)
	} else {
		fmt.Printf("  Name:         %s\n", info.Name)
		fmt.Printf("  Symbol:       %s\n", info.Symbol)
		fmt.Printf("  Decimals:     %d\n", info.Decimals)
		fmt.Printf("  Total Supply: %s\n", info.TotalSupply)
	}

	fmt.Println("\n=== TRC20: Balance ===")
	balance, err := token.BalanceOf(ctx, sampleAddress)
	if err != nil {
		fmt.Printf("  BalanceOf error: %v\n", err)
	} else {
		fmt.Printf("  Raw:     %s\n", balance.Raw)
		fmt.Printf("  Display: %s %s\n", balance.Display, balance.Symbol)
	}

	fmt.Println("\n=== TRC20: Transfer (build only) ===")
	trc20Tx := token.Transfer(sampleAddress, receiverAddress, big.NewInt(1_000_000),
		contract.WithFeeLimit(100_000_000))

	trc20Built, err := trc20Tx.Build(ctx)
	if err != nil {
		fmt.Printf("  Build error: %v\n", err)
	} else {
		fmt.Printf("  Built tx hash: %x\n", trc20Built.GetTxid())
	}

	// =====================================================
	// 4. FULL FLOW: Build → Sign → Send
	// =====================================================
	fmt.Println("\n=== Full flow: Transfer → Sign → Send ===")
	fmt.Printf("  Signer address: %s\n", s.Address())
	fmt.Println("  (skipping broadcast in demo mode)")

	_ = s // signer ready for: tx.Transfer(...).Send(ctx, s)
}

func connect() *client.GrpcClient {
	conn := client.NewGrpcClient(grpcEndpoint)
	if err := conn.Start(grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		fmt.Fprintf(os.Stderr, "connect error: %v\n", err)
		os.Exit(1)
	}
	return conn
}

func newSigner() signer.Signer {
	key, err := keys.GenerateKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "key error: %v\n", err)
		os.Exit(1)
	}
	s, err := signer.NewPrivateKeySignerFromBTCEC(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "signer error: %v\n", err)
		os.Exit(1)
	}
	return s
}

func printTx(label string, tx *txbuilder.Tx, ctx context.Context) {
	ext, err := tx.Build(ctx)
	if err != nil {
		fmt.Printf("  %s — error: %v\n", label, err)
		return
	}
	fmt.Printf("  %s — tx hash: %x\n", label, ext.GetTxid())
}

func printVoteTx(label string, tx *txbuilder.VoteTx, ctx context.Context) {
	ext, err := tx.Build(ctx)
	if err != nil {
		fmt.Printf("  %s — error: %v\n", label, err)
		return
	}
	fmt.Printf("  %s — tx hash: %x\n", label, ext.GetTxid())
}

func printDelegateTx(label string, tx *txbuilder.DelegateTx, ctx context.Context) {
	ext, err := tx.Build(ctx)
	if err != nil {
		fmt.Printf("  %s — error: %v\n", label, err)
		return
	}
	fmt.Printf("  %s — tx hash: %x\n", label, ext.GetTxid())
}
