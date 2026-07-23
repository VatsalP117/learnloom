package artifact

import (
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestArtifactCacheEvictsLeastRecentlyUsedEntries(t *testing.T) {
	cache := newArtifactCache(10)
	cache.put("one", domain.DossierArtifact{Markdown: "one"}, 4)
	cache.put("two", domain.DossierArtifact{Markdown: "two"}, 4)
	if _, ok := cache.get("one"); !ok {
		t.Fatal("expected first entry")
	}
	cache.put("three", domain.DossierArtifact{Markdown: "three"}, 4)

	if _, ok := cache.get("two"); ok {
		t.Fatal("least recently used entry was not evicted")
	}
	if value, ok := cache.get("one"); !ok || value.Markdown != "one" {
		t.Fatalf("recent entry=%#v, ok=%v", value, ok)
	}
}

func TestArtifactCacheSkipsOversizedEntries(t *testing.T) {
	cache := newArtifactCache(5)
	cache.put("large", domain.DossierArtifact{Markdown: "large"}, 6)
	if _, ok := cache.get("large"); ok {
		t.Fatal("oversized entry was cached")
	}
}
