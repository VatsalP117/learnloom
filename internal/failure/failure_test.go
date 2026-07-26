package failure

import (
	"errors"
	"strings"
	"testing"
)

func TestDescribeKeepsInternalDetailOutOfPublicMessage(t *testing.T) {
	secret := errors.New("provider rejected secret-token")
	detail := Describe(New(
		"model_contract_unsatisfied",
		CategoryContentQuality,
		"editor",
		true,
		PublicInternal,
		secret,
	))
	if detail.Code != "model_contract_unsatisfied" ||
		detail.Category != CategoryContentQuality ||
		detail.Stage != "editor" ||
		!detail.Retryable ||
		detail.IncidentID == "" {
		t.Fatalf("unexpected detail: %#v", detail)
	}
	if !strings.Contains(detail.Internal, "secret-token") ||
		strings.Contains(detail.PublicMessage, "secret-token") {
		t.Fatalf("failure detail leaked: %#v", detail)
	}
}
