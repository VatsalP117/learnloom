package httpapp

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/store"
)

func TestOnboardingSeedPayloadBuildsPrivateBoundedHybridDraft(t *testing.T) {
	t.Parallel()
	origin := store.PublicStartingPath{
		PublicID:         "dossier-30000000-0000-0000-0000-000000000000",
		PublicSlug:       "a-public-dossier",
		Title:            "A public Dossier",
		NewsletterName:   "Systems thinking",
		NewsletterTopic:  "How systems fail",
		NewsletterGoal:   "Explain failure modes clearly",
		OwnerDisplayName: "Maya",
		SiteUsername:     "maya",
		Sources: []store.PublicStartingSource{
			{Title: "Source one", CanonicalURL: "https://example.org/one"},
			{Title: "Source two", CanonicalURL: "https://example.org/two"},
		},
	}
	payload := onboardingSeedPayload(origin, "learnloom.blog")
	if payload.Name != "Systems thinking" || payload.Topic != "How systems fail" ||
		payload.LearnerGoal != "Explain failure modes clearly" ||
		payload.LearnerLevel != "intermediate" || payload.LessonMinutes != 12 ||
		payload.ScheduleTime != "08:00" || !payload.Active ||
		payload.EmailEnabled || payload.AIExplorationEnabled ||
		payload.SourceMode != domain.SourceModeHybrid ||
		payload.SourceReviewMode != domain.SourceReviewBeforeLesson ||
		payload.TimeZone != "" {
		t.Fatalf("unexpected seeded payload %#v", payload)
	}
	if len(payload.Sources) != 2 ||
		payload.Sources[0].Name != "Source one" ||
		payload.Sources[0].URL != "https://example.org/one" ||
		payload.Sources[0].Limit != 8 {
		t.Fatalf("unexpected seeded sources %#v", payload.Sources)
	}
	if payload.Attribution == nil ||
		payload.Attribution.DossierPublicID != origin.PublicID ||
		payload.Attribution.Title != "A public Dossier" ||
		payload.Attribution.OwnerName != "Maya" ||
		payload.Attribution.CanonicalURL !=
			"https://maya.learnloom.blog/d/dossier-30000000-0000-0000-0000-000000000000/a-public-dossier" {
		t.Fatalf("unexpected seeded attribution %#v", payload.Attribution)
	}
}

func TestOnboardingSeedPayloadFallsBackToDiscoveredModeWithoutSources(t *testing.T) {
	t.Parallel()
	payload := onboardingSeedPayload(store.PublicStartingPath{
		PublicID:       "dossier-30000000-0000-0000-0000-000000000000",
		PublicSlug:     "a",
		NewsletterName: "Systems",
		SiteUsername:   "maya",
	}, "learnloom.blog")
	if payload.SourceMode != domain.SourceModeDiscovered || len(payload.Sources) != 0 {
		t.Fatalf("expected discovered mode without evidence, got %#v", payload)
	}
}

func TestOnboardingSeedPayloadBoundsTransferredText(t *testing.T) {
	t.Parallel()
	origin := store.PublicStartingPath{
		PublicID:         "dossier-30000000-0000-0000-0000-000000000000",
		PublicSlug:       "a",
		Title:            strings.Repeat("t", 300),
		NewsletterName:   strings.Repeat("n", 100),
		NewsletterTopic:  strings.Repeat("p", 500),
		NewsletterGoal:   strings.Repeat("g", 700),
		OwnerDisplayName: strings.Repeat("o", 100),
		SiteUsername:     "maya",
		Sources: []store.PublicStartingSource{
			{Title: strings.Repeat("s", 200), CanonicalURL: "https://example.org/one"},
		},
	}
	payload := onboardingSeedPayload(origin, "learnloom.blog")
	if len([]rune(payload.Name)) != 80 || len([]rune(payload.Topic)) != 400 ||
		len([]rune(payload.LearnerGoal)) != 500 ||
		len([]rune(payload.Attribution.Title)) != 200 ||
		len([]rune(payload.Attribution.OwnerName)) != 80 ||
		len([]rune(payload.Sources[0].Name)) != 120 {
		t.Fatalf("seeded text was not bounded %#v", payload)
	}
}

func TestOnboardingDraftStartAcceptsOnlyPost(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{MaxRequestBodyBytes: 1 << 10}}
	request := httptest.NewRequest(http.MethodGet, "/api/onboarding/draft/start", nil)
	response := httptest.NewRecorder()
	server.onboardingDraftStart(response, request, session{})
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d want 405", response.Code)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("Allow=%q want POST", allow)
	}
}

func TestPublicReferralCookieValueValidation(t *testing.T) {
	t.Parallel()
	valid := hex.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	for _, test := range []struct {
		name  string
		value string
		ok    bool
	}{
		{name: "exact 32-byte hex", value: valid, ok: true},
		{name: "surrounding whitespace", value: "  " + valid + " ", ok: true},
		{name: "too short", value: "abcd"},
		{name: "non-hex", value: strings.Repeat("z", 64)},
		{name: "empty", value: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := validPublicReferralCookieValue(test.value); got != test.ok {
				t.Fatalf("validPublicReferralCookieValue(%q)=%v want %v", test.value, got, test.ok)
			}
		})
	}
}
