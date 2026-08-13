package store

import (
	"context"
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestReplaceProvidedSourceRejectsNonPublicLiteralBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()
	_, err := (&Store{}).ReplaceProvidedSource(
		context.Background(),
		"account",
		"newsletter",
		"source",
		domain.SourceDefinition{Name: "Internal", URL: "http://127.0.0.1/private", Limit: 8},
	)
	if err == nil || !strings.Contains(err.Error(), "public host") {
		t.Fatalf("private replacement URL was accepted: %v", err)
	}
}

func TestSetSourcePreferenceRejectsUnknownValueBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()
	err := (&Store{}).SetSourcePreference(
		context.Background(),
		"account",
		"newsletter",
		"source",
		domain.SourcePreference("trust_me"),
	)
	if err == nil {
		t.Fatal("unknown source preference was accepted")
	}
}
