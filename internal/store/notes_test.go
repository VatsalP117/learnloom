package store

import (
	"strings"
	"testing"
)

func TestNormalizeLessonNote(t *testing.T) {
	t.Parallel()
	valid, err := normalizeLessonNote(LessonNoteInput{
		Kind:       " question ",
		AnchorType: " claim ",
		AnchorID:   " claim-1 ",
		Body:       " What should I verify? ",
		QuotedText: " Evidence ",
	})
	if err != nil || valid.Kind != "question" ||
		valid.AnchorID != "claim-1" ||
		valid.Body != "What should I verify?" {
		t.Fatalf("normalized=%#v err=%v", valid, err)
	}
	for _, input := range []LessonNoteInput{
		{Kind: "unknown", AnchorType: "lesson", Body: "Body"},
		{Kind: "note", AnchorType: "unknown", Body: "Body"},
		{Kind: "note", AnchorType: "lesson", Body: ""},
		{Kind: "highlight", AnchorType: "lesson", Body: "Body", QuotedText: strings.Repeat("x", 1201)},
	} {
		if _, err := normalizeLessonNote(input); err == nil {
			t.Fatalf("invalid note accepted: %#v", input)
		}
	}
}
