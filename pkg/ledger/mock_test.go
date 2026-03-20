package ledger

import (
	"context"
	"errors"
	"testing"
)

func TestMockDevice_GetAddress(t *testing.T) {
	t.Run("returns configured address", func(t *testing.T) {
		dev := &MockDevice{Address: "TADDR123"}
		addr, err := dev.GetAddress()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if addr != "TADDR123" {
			t.Fatalf("got %q, want %q", addr, "TADDR123")
		}
	})

	t.Run("returns configured error", func(t *testing.T) {
		dev := &MockDevice{Err: errors.New("device error")}
		_, err := dev.GetAddress()
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("calls GetAddressFn when set", func(t *testing.T) {
		called := false
		dev := &MockDevice{
			GetAddressFn: func() (string, error) {
				called = true
				return "CUSTOM", nil
			},
		}
		addr, err := dev.GetAddress()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Fatal("GetAddressFn was not called")
		}
		if addr != "CUSTOM" {
			t.Fatalf("got %q, want %q", addr, "CUSTOM")
		}
	})
}

func TestMockDevice_SignTransaction(t *testing.T) {
	t.Run("returns configured signature", func(t *testing.T) {
		sig := []byte{1, 2, 3}
		dev := &MockDevice{Signature: sig}
		got, err := dev.SignTransaction(context.Background(), []byte{0xff})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != len(sig) {
			t.Fatalf("got len %d, want %d", len(got), len(sig))
		}
	})

	t.Run("calls SignTransactionFn when set", func(t *testing.T) {
		dev := &MockDevice{
			SignTransactionFn: func(_ context.Context, tx []byte) ([]byte, error) {
				return append([]byte{0xAA}, tx...), nil
			},
		}
		got, err := dev.SignTransaction(context.Background(), []byte{0xBB})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got[0] != 0xAA || got[1] != 0xBB {
			t.Fatalf("unexpected result: %x", got)
		}
	})
}

func TestMockDevice_Close(t *testing.T) {
	t.Run("returns nil by default", func(t *testing.T) {
		dev := &MockDevice{}
		if err := dev.Close(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("calls CloseFn when set", func(t *testing.T) {
		want := errors.New("close error")
		dev := &MockDevice{
			CloseFn: func() error { return want },
		}
		if err := dev.Close(); !errors.Is(err, want) {
			t.Fatalf("got %v, want %v", err, want)
		}
	})
}

// Verify MockDevice implements Device at compile time.
var _ Device = (*MockDevice)(nil)
