package store

import (
	"context"
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestNormalizeUsername(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"vatsal", "alice-2", "abc"} {
		normalized, err := normalizeUsername(value)
		if err != nil || normalized != value {
			t.Errorf("%q: normalized=%q err=%v", value, normalized, err)
		}
	}
	for _, value := range []string{"ab", "2alice", "alice-", "alice--two", "api", "UPPER SPACE"} {
		if _, err := normalizeUsername(value); err == nil {
			t.Errorf("%q should be rejected", value)
		}
	}
}

func TestUpdateSiteRejectsIndexingForPrivateSite(t *testing.T) {
	t.Parallel()
	enabled := true
	_, err := (&Store{}).UpdateSite(
		context.Background(),
		"account-id",
		domain.SitePrivate,
		nil,
		nil,
		&enabled,
	)
	if err == nil || !strings.Contains(err.Error(), "requires a public") {
		t.Fatalf("UpdateSite() error = %v", err)
	}
}
