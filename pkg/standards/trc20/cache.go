package trc20

import (
	"container/list"
	"sync"
)

// Bitmask flags tracking which metadata fields have been populated.
const (
	metaName     uint8 = 1 << iota // 1
	metaSymbol                     // 2
	metaDecimals                   // 4
)

// tokenMeta holds the immutable metadata for a single TRC20 contract.
type tokenMeta struct {
	name      string
	symbol    string
	decimals  uint8
	populated uint8 // bitmask of metaName | metaSymbol | metaDecimals
}

// cacheEntry is stored in the LRU list elements.
type cacheEntry struct {
	addr string // contract address (map key)
	meta tokenMeta
}

// MetadataCache is a thread-safe LRU cache for immutable TRC20 token metadata
// (name, symbol, decimals). It is safe for concurrent use by multiple
// goroutines and multiple Token instances.
//
// Create with NewMetadataCache and pass to Token via WithCache.
type MetadataCache struct {
	mu       sync.Mutex
	maxSize  int
	items    map[string]*list.Element
	eviction *list.List // front = most recently used
}

// NewMetadataCache creates a cache that holds metadata for up to maxSize
// contract addresses. When full, the least recently used entry is evicted.
func NewMetadataCache(maxSize int) *MetadataCache {
	if maxSize < 1 {
		maxSize = 1
	}
	return &MetadataCache{
		maxSize:  maxSize,
		items:    make(map[string]*list.Element, maxSize),
		eviction: list.New(),
	}
}

// Len returns the number of entries currently in the cache.
func (c *MetadataCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

// Evict removes the entry for the given contract address.
// Returns true if an entry was removed.
func (c *MetadataCache) Evict(contractAddress string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[contractAddress]; ok {
		c.removeElement(el)
		return true
	}
	return false
}

// Clear removes all entries from the cache.
func (c *MetadataCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element, c.maxSize)
	c.eviction.Init()
}

// getOrCreate returns the entry for addr, creating it if needed.
// Moves the entry to the front of the LRU list.
// Caller must hold c.mu.
func (c *MetadataCache) getOrCreate(addr string) *cacheEntry {
	if el, ok := c.items[addr]; ok {
		c.eviction.MoveToFront(el)
		return el.Value.(*cacheEntry)
	}
	// Create new entry, evict if at capacity.
	entry := &cacheEntry{addr: addr}
	el := c.eviction.PushFront(entry)
	c.items[addr] = el
	if c.eviction.Len() > c.maxSize {
		c.removeOldest()
	}
	return entry
}

// removeOldest evicts the least recently used entry.
// Caller must hold c.mu.
func (c *MetadataCache) removeOldest() {
	el := c.eviction.Back()
	if el != nil {
		c.removeElement(el)
	}
}

// removeElement removes an element from both the list and map.
// Caller must hold c.mu.
func (c *MetadataCache) removeElement(el *list.Element) {
	c.eviction.Remove(el)
	entry := el.Value.(*cacheEntry)
	delete(c.items, entry.addr)
}

// --- Package-private get/put methods called by Token ---

func (c *MetadataCache) getName(addr string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[addr]
	if !ok {
		return "", false
	}
	c.eviction.MoveToFront(el)
	entry := el.Value.(*cacheEntry)
	if entry.meta.populated&metaName == 0 {
		return "", false
	}
	return entry.meta.name, true
}

func (c *MetadataCache) getSymbol(addr string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[addr]
	if !ok {
		return "", false
	}
	c.eviction.MoveToFront(el)
	entry := el.Value.(*cacheEntry)
	if entry.meta.populated&metaSymbol == 0 {
		return "", false
	}
	return entry.meta.symbol, true
}

func (c *MetadataCache) getDecimals(addr string) (uint8, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[addr]
	if !ok {
		return 0, false
	}
	c.eviction.MoveToFront(el)
	entry := el.Value.(*cacheEntry)
	if entry.meta.populated&metaDecimals == 0 {
		return 0, false
	}
	return entry.meta.decimals, true
}

func (c *MetadataCache) putName(addr, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.getOrCreate(addr)
	entry.meta.name = name
	entry.meta.populated |= metaName
}

func (c *MetadataCache) putSymbol(addr, symbol string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.getOrCreate(addr)
	entry.meta.symbol = symbol
	entry.meta.populated |= metaSymbol
}

func (c *MetadataCache) putDecimals(addr string, decimals uint8) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.getOrCreate(addr)
	entry.meta.decimals = decimals
	entry.meta.populated |= metaDecimals
}
