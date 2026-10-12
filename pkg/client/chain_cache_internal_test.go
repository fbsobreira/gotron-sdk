package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPQCapsCache_KeepsLaterProbe checks that when two overlapping probes
// finish out of order, the one that started later stays cached.
func TestPQCapsCache_KeepsLaterProbe(t *testing.T) {
	var c pqCapsCache
	_, ok, gen := c.lookup()
	require.False(t, ok)

	older := time.Now()
	newer := older.Add(time.Millisecond)
	c.store(PQCapabilities{MLDSA44: true}, newer, gen)
	c.store(PQCapabilities{FNDSA512: true}, older, gen)

	got, ok, _ := c.lookup()
	require.True(t, ok)
	assert.Equal(t, PQCapabilities{MLDSA44: true}, got, "the probe that started later must win")
}

// TestPQCapsCache_StaleGenerationNotStored checks that a probe that started
// before an invalidation does not store its result.
func TestPQCapsCache_StaleGenerationNotStored(t *testing.T) {
	var c pqCapsCache
	_, _, gen := c.lookup()

	c.mu.Lock()
	c.invalidateLocked()
	c.mu.Unlock()
	c.store(PQCapabilities{MLDSA44: true}, time.Now(), gen)

	_, ok, _ := c.lookup()
	assert.False(t, ok)
}
