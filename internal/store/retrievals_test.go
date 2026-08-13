package store

import (
	"testing"
	"time"
)

func TestLessonRetrievalInputValidation(t *testing.T) {
	t.Parallel()
	if _, err := (&Store{}).RevealLessonRetrieval(
		t.Context(), "account", "issue",
		LessonRetrievalInput{PromptKey: "retrieval-1"},
		time.Time{},
	); err == nil {
		t.Fatal("empty response was accepted")
	}
	if _, err := (&Store{}).SaveLessonRetrievalDraft(
		t.Context(), "account", "issue",
		LessonRetrievalInput{PromptKey: "retrieval-1", Response: "no"},
		time.Time{},
	); err == nil {
		t.Fatal("non-attempt draft was accepted")
	}
}
