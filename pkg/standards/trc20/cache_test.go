package trc20

import (
	"context"
	"math/big"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- MetadataCache unit tests ---

func TestNewMetadataCache(t *testing.T) {
	c := NewMetadataCache(100)
	assert.Equal(t, 0, c.Len())
}

func TestNewMetadataCache_MinSize(t *testing.T) {
	c := NewMetadataCache(0)
	assert.NotNil(t, c)
	// Should clamp to 1
	c.putName("A", "TokenA")
	c.putName("B", "TokenB")
	assert.Equal(t, 1, c.Len()) // A evicted
}

func TestCache_PutAndGet(t *testing.T) {
	c := NewMetadataCache(10)

	c.putName("TUSDT", "Tether USD")
	c.putSymbol("TUSDT", "USDT")
	c.putDecimals("TUSDT", 6)

	name, ok := c.getName("TUSDT")
	assert.True(t, ok)
	assert.Equal(t, "Tether USD", name)

	symbol, ok := c.getSymbol("TUSDT")
	assert.True(t, ok)
	assert.Equal(t, "USDT", symbol)

	decimals, ok := c.getDecimals("TUSDT")
	assert.True(t, ok)
	assert.Equal(t, uint8(6), decimals)
}

func TestCache_MissReturnsNotOk(t *testing.T) {
	c := NewMetadataCache(10)

	_, ok := c.getName("TUSDT")
	assert.False(t, ok)

	_, ok = c.getSymbol("TUSDT")
	assert.False(t, ok)

	_, ok = c.getDecimals("TUSDT")
	assert.False(t, ok)
}

func TestCache_PartialPopulation(t *testing.T) {
	c := NewMetadataCache(10)

	// Only populate decimals
	c.putDecimals("TUSDT", 6)

	// Decimals should hit
	d, ok := c.getDecimals("TUSDT")
	assert.True(t, ok)
	assert.Equal(t, uint8(6), d)

	// Name and symbol should miss (not populated)
	_, ok = c.getName("TUSDT")
	assert.False(t, ok)
	_, ok = c.getSymbol("TUSDT")
	assert.False(t, ok)

	// Now populate symbol
	c.putSymbol("TUSDT", "USDT")
	symbol, ok := c.getSymbol("TUSDT")
	assert.True(t, ok)
	assert.Equal(t, "USDT", symbol)

	// Name still missing
	_, ok = c.getName("TUSDT")
	assert.False(t, ok)
}

func TestCache_LRUEviction(t *testing.T) {
	c := NewMetadataCache(3)

	c.putName("A", "TokenA")
	c.putName("B", "TokenB")
	c.putName("C", "TokenC")
	assert.Equal(t, 3, c.Len())

	// Adding D should evict A (least recently used)
	c.putName("D", "TokenD")
	assert.Equal(t, 3, c.Len())

	_, ok := c.getName("A")
	assert.False(t, ok, "A should have been evicted")

	_, ok = c.getName("B")
	assert.True(t, ok)
	_, ok = c.getName("D")
	assert.True(t, ok)
}

func TestCache_LRUAccessUpdatesRecency(t *testing.T) {
	c := NewMetadataCache(3)

	c.putName("A", "TokenA")
	c.putName("B", "TokenB")
	c.putName("C", "TokenC")

	// Access A to move it to front
	c.getName("A")

	// Adding D should evict B (now least recently used), not A
	c.putName("D", "TokenD")

	_, ok := c.getName("A")
	assert.True(t, ok, "A should still be cached after access")

	_, ok = c.getName("B")
	assert.False(t, ok, "B should have been evicted")
}

func TestCache_Evict(t *testing.T) {
	c := NewMetadataCache(10)

	c.putName("TUSDT", "Tether USD")
	c.putDecimals("TUSDT", 6)

	removed := c.Evict("TUSDT")
	assert.True(t, removed)
	assert.Equal(t, 0, c.Len())

	_, ok := c.getName("TUSDT")
	assert.False(t, ok)

	// Evicting non-existent returns false
	removed = c.Evict("nonexistent")
	assert.False(t, removed)
}

func TestCache_Clear(t *testing.T) {
	c := NewMetadataCache(10)

	c.putName("A", "TokenA")
	c.putName("B", "TokenB")
	c.putName("C", "TokenC")
	assert.Equal(t, 3, c.Len())

	c.Clear()
	assert.Equal(t, 0, c.Len())

	_, ok := c.getName("A")
	assert.False(t, ok)
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := NewMetadataCache(100)
	var wg sync.WaitGroup

	// 10 goroutines writing different keys
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			addr := string(rune('A' + id))
			for range 100 {
				c.putName(addr, "Token"+addr)
				c.putSymbol(addr, addr)
				c.putDecimals(addr, uint8(id))
				c.getName(addr)
				c.getSymbol(addr)
				c.getDecimals(addr)
			}
		}(i)
	}
	wg.Wait()

	// Should not panic or corrupt state
	assert.LessOrEqual(t, c.Len(), 100)
}

// --- Token integration tests with cache ---

func TestToken_CacheReducesRPCCalls(t *testing.T) {
	// Only one result in the queue — if second Decimals() call hits RPC,
	// it will get nil results and fail. Cache makes it succeed.
	mc := &mockClient{
		results: [][][]byte{
			{abiEncodeUint256(big.NewInt(6))},
			// No more results — second RPC call would fail
		},
	}

	cache := NewMetadataCache(10)
	token := New(mc, "TContract", WithCache(cache))

	// First call — hits RPC, populates cache
	d1, err := token.Decimals(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint8(6), d1)

	// Second call — must use cache (no RPC results left)
	d2, err := token.Decimals(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint8(6), d2)

	// Verify cache has the entry
	cached, ok := cache.getDecimals("TContract")
	assert.True(t, ok)
	assert.Equal(t, uint8(6), cached)
}

func TestToken_NoCacheBackwardCompatible(t *testing.T) {
	mc := &mockClient{
		constantResult: [][]byte{abiEncodeString("Tether USD")},
	}

	// No WithCache — existing behavior
	token := New(mc, "TContract")

	name, err := token.Name(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Tether USD", name)
}

func TestToken_CacheSharedAcrossInstances(t *testing.T) {
	mc := &mockClient{
		constantResult: [][]byte{abiEncodeString("USDT")},
	}

	cache := NewMetadataCache(10)

	// Two Token instances for the same contract, sharing a cache
	t1 := New(mc, "TContract", WithCache(cache))
	t2 := New(mc, "TContract", WithCache(cache))

	// First instance populates cache
	sym1, err := t1.Symbol(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "USDT", sym1)

	// Second instance should get cache hit
	sym2, err := t2.Symbol(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "USDT", sym2)
}

func TestToken_EvictForcesRefetch(t *testing.T) {
	mc := &mockClient{
		results: [][][]byte{
			{abiEncodeString("OldName")},
			{abiEncodeString("NewName")}, // returned after eviction
		},
	}

	cache := NewMetadataCache(10)
	token := New(mc, "TContract", WithCache(cache))

	// First call populates cache
	name, err := token.Name(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "OldName", name)

	// Evict forces next call to hit RPC
	cache.Evict("TContract")

	// Second call hits RPC again (cache evicted), gets "NewName"
	name, err = token.Name(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "NewName", name)
}

func TestToken_BalanceOfUsesCachedDecimals(t *testing.T) {
	mc := &mockClient{
		results: [][][]byte{
			// First call: balanceOf
			{abiEncodeUint256(big.NewInt(1_500_000))},
			// Second call: decimals
			{abiEncodeUint256(big.NewInt(6))},
			// Third call: symbol
			{abiEncodeString("USDT")},
			// Fourth call: balanceOf again
			{abiEncodeUint256(big.NewInt(2_000_000))},
			// decimals and symbol should come from cache — no more RPC results needed
		},
	}

	cache := NewMetadataCache(10)
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1", WithCache(cache))

	// First BalanceOf — fetches balance, decimals, symbol from RPC
	bal1, err := token.BalanceOf(context.Background(), "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)
	assert.Equal(t, "1.5", bal1.Display)
	assert.Equal(t, "USDT", bal1.Symbol)

	// Second BalanceOf — fetches balance from RPC, decimals and symbol from cache
	bal2, err := token.BalanceOf(context.Background(), "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)
	assert.Equal(t, "2", bal2.Display)
	assert.Equal(t, "USDT", bal2.Symbol)
}
