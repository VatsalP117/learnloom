package artifact

import (
	"container/list"
	"sync"

	"github.com/VatsalP117/learnloom/internal/domain"
)

type cacheEntry struct {
	key      string
	artifact domain.DossierArtifact
	bytes    int64
}

type artifactCache struct {
	mu       sync.Mutex
	maxBytes int64
	bytes    int64
	entries  map[string]*list.Element
	order    *list.List
}

func newArtifactCache(maxBytes int64) *artifactCache {
	if maxBytes <= 0 {
		return nil
	}
	return &artifactCache{
		maxBytes: maxBytes,
		entries:  make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (cache *artifactCache) get(key string) (domain.DossierArtifact, bool) {
	if cache == nil {
		return domain.DossierArtifact{}, false
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	element, exists := cache.entries[key]
	if !exists {
		return domain.DossierArtifact{}, false
	}
	cache.order.MoveToFront(element)
	return element.Value.(cacheEntry).artifact, true
}

func (cache *artifactCache) put(key string, artifact domain.DossierArtifact, bytes int64) {
	if cache == nil || bytes <= 0 || bytes > cache.maxBytes {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if existing, exists := cache.entries[key]; exists {
		cache.bytes -= existing.Value.(cacheEntry).bytes
		cache.order.Remove(existing)
		delete(cache.entries, key)
	}
	element := cache.order.PushFront(cacheEntry{
		key: key, artifact: artifact, bytes: bytes,
	})
	cache.entries[key] = element
	cache.bytes += bytes
	for cache.bytes > cache.maxBytes {
		oldest := cache.order.Back()
		if oldest == nil {
			break
		}
		entry := oldest.Value.(cacheEntry)
		cache.bytes -= entry.bytes
		delete(cache.entries, entry.key)
		cache.order.Remove(oldest)
	}
}

func (cache *artifactCache) remove(key string) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	element, exists := cache.entries[key]
	if !exists {
		return
	}
	cache.bytes -= element.Value.(cacheEntry).bytes
	delete(cache.entries, key)
	cache.order.Remove(element)
}
