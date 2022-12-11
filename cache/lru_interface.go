package cache

// LRUCache is the interface for simple LRU cache.
type LRUCache[K comparable, V any] interface {
	// Add Adds a value to the cache, returns true if an eviction occurred and
	// updates the "recently used"-ness of the key.
	Add(key K, value V) bool

	// Get Returns key's value from the cache and
	// updates the "recently used"-ness of the key. #value, isFound
	Get(key K) (value V, ok bool)

	// Contains Checks if a key exists in cache without updating the recent-ness.
	Contains(key K) (ok bool)

	// Peek Returns key's value without updating the "recently used"-ness of the key.
	Peek(key K) (value V, ok bool)

	// Remove Removes a key from the cache.
	Remove(key K) bool

	// RemoveOldest Removes the oldest entry from cache.
	RemoveOldest() (K, V, bool)

	// GetOldest Returns the oldest entry from the cache. #key, value, isFound
	GetOldest() (K, V, bool)

	// Keys Returns a slice of the keys in the cache, from oldest to newest.
	Keys() []K

	// Len Returns the number of items in the cache.
	Len() int

	// Purge Clears all cache entries.
	Purge()

	// Resize Resizes cache, returning number evicted
	Resize(int) int
}
