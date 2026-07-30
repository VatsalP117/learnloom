package httpapp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestRunSourceValidationPreservesOrderAndReportsAvailability(t *testing.T) {
	t.Parallel()
	definitions := []domain.SourceDefinition{
		{Name: "Working", URL: "https://example.com/working.xml"},
		{Name: "Broken", URL: "https://example.com/broken.xml"},
	}
	results := runSourceValidation(
		context.Background(),
		func(
			_ context.Context,
			definition domain.SourceDefinition,
		) ([]domain.SourceItem, []string, error) {
			if definition.Name == "Broken" {
				return nil, []string{"Broken: feed returned 503"}, errors.New("no items")
			}
			return []domain.SourceItem{
				{Title: "First useful item"},
				{Title: "Second useful item"},
				{Title: "Third item is intentionally omitted"},
			}, nil, nil
		},
		definitions,
	)

	if len(results) != 2 ||
		results[0].Name != "Working" ||
		results[0].Status != "ready" ||
		results[0].ItemCount != 3 ||
		len(results[0].SampleTitles) != 2 {
		t.Fatalf("ready validation=%#v", results)
	}
	if results[1].Name != "Broken" ||
		results[1].Status != "unavailable" ||
		results[1].Message != "no items" {
		t.Fatalf("failed validation=%#v", results[1])
	}
}

func TestSourceValidationMessageIsBounded(t *testing.T) {
	t.Parallel()
	message := sourceValidationMessage(errors.New(strings.Repeat("x", 300)), nil)
	if len(message) != 240 {
		t.Fatalf("message length=%d", len(message))
	}
}
