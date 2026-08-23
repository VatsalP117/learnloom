package store

import (
	"strings"
	"testing"

	"github.com/VatsalP117/learnloom/internal/domain"
)

func TestValidateOnboardingDraftAllowsIncompleteSafeProgress(t *testing.T) {
	t.Parallel()
	err := validateOnboardingDraft(2, OnboardingDraftPayload{
		Topic:        "How language models fail",
		LearnerLevel: "intermediate",
		SourceMode:   domain.SourceModeHybrid,
		Sources: []domain.SourceDefinition{
			{Name: "Incomplete row"},
			{Name: "Research", URL: "https://example.org/research", Limit: 8},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateOnboardingDraftRejectsUnsafeOrUnboundedState(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		step    int
		payload OnboardingDraftPayload
	}{
		{name: "invalid step", step: 4},
		{name: "long topic", step: 1, payload: OnboardingDraftPayload{Topic: strings.Repeat("x", 401)}},
		{name: "private source", step: 2, payload: OnboardingDraftPayload{Sources: []domain.SourceDefinition{{URL: "http://127.0.0.1/private"}}}},
		{name: "credentials", step: 2, payload: OnboardingDraftPayload{Sources: []domain.SourceDefinition{{URL: "https://user:pass@example.com"}}}},
		{name: "bare attribution ID", step: 1, payload: OnboardingDraftPayload{Attribution: &OnboardingAttribution{DossierPublicID: "30000000-0000-0000-0000-000000000000"}}},
		{name: "malformed attribution ID", step: 1, payload: OnboardingDraftPayload{Attribution: &OnboardingAttribution{DossierPublicID: "dossier-nope"}}},
		{name: "long attribution title", step: 1, payload: OnboardingDraftPayload{Attribution: &OnboardingAttribution{
			DossierPublicID: "dossier-30000000-0000-0000-0000-000000000000",
			Title:           strings.Repeat("x", 201),
		}}},
		{name: "http attribution URL", step: 1, payload: OnboardingDraftPayload{Attribution: &OnboardingAttribution{
			DossierPublicID: "dossier-30000000-0000-0000-0000-000000000000",
			CanonicalURL:    "http://example.org/d/dossier-30000000-0000-0000-0000-000000000000/a",
		}}},
		{name: "private attribution URL", step: 1, payload: OnboardingDraftPayload{Attribution: &OnboardingAttribution{
			DossierPublicID: "dossier-30000000-0000-0000-0000-000000000000",
			CanonicalURL:    "https://127.0.0.1/d/dossier-30000000-0000-0000-0000-000000000000/a",
		}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateOnboardingDraft(test.step, test.payload); err == nil {
				t.Fatal("unsafe draft was accepted")
			}
		})
	}
}

func TestValidateOnboardingDraftAcceptsBoundedSeedAttribution(t *testing.T) {
	t.Parallel()
	err := validateOnboardingDraft(1, OnboardingDraftPayload{
		Topic:        "How language models fail",
		LearnerLevel: "intermediate",
		SourceMode:   domain.SourceModeHybrid,
		Sources: []domain.SourceDefinition{{
			Name: "Evidence", URL: "https://example.org/research", Limit: 8,
		}},
		Attribution: &OnboardingAttribution{
			DossierPublicID: "dossier-30000000-0000-0000-0000-000000000000",
			Title:           "A public Dossier",
			CanonicalURL:    "https://maya.learnloom.blog/d/dossier-30000000-0000-0000-0000-000000000000/a-path",
			OwnerName:       "Maya",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
