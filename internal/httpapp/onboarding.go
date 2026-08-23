package httpapp

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/store"
	"github.com/google/uuid"
)

func (s *Server) onboardingDraft(
	response http.ResponseWriter,
	request *http.Request,
	current session,
) {
	switch request.Method {
	case http.MethodGet:
		draft, err := s.store.GetOnboardingDraft(
			request.Context(),
			current.Account.ID,
		)
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{"draft": draft})
	case http.MethodPut:
		if !s.allowAction(response, request, "onboarding-draft-save", time.Hour, 120) {
			return
		}
		var body struct {
			DraftID          string                       `json:"draftId"`
			ExpectedRevision int64                        `json:"expectedRevision"`
			Step             int                          `json:"step"`
			Payload          store.OnboardingDraftPayload `json:"payload"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		draft, err := s.store.SaveOnboardingDraft(
			request.Context(),
			current.Account.ID,
			body.DraftID,
			body.ExpectedRevision,
			body.Step,
			body.Payload,
			time.Now().UTC(),
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{"draft": draft})
	case http.MethodDelete:
		expectedRevision, err := strconv.ParseInt(
			request.URL.Query().Get("expectedRevision"), 10, 64,
		)
		if err != nil || expectedRevision < 1 {
			writeProblem(response, http.StatusBadRequest, "invalid_revision", "expectedRevision must be a positive integer")
			return
		}
		if err := s.store.DeleteOnboardingDraft(
			request.Context(),
			current.Account.ID,
			request.URL.Query().Get("draftId"),
			expectedRevision,
			request.URL.Query().Get("reason") == "abandoned",
			time.Now().UTC(),
		); err != nil {
			writeStoreError(response, err)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(response, http.MethodGet, http.MethodPut, http.MethodDelete)
	}
}

// onboardingDraftStart seeds a fresh private onboarding draft from the
// visitor's latest eligible recorded public CTA click. The query string is
// never trusted: the referral cookie fingerprint is the only attribution
// authority, and the existing draft wins over any racemate.
func (s *Server) onboardingDraftStart(
	response http.ResponseWriter,
	request *http.Request,
	current session,
) {
	if request.Method != http.MethodPost {
		methodNotAllowed(response, http.MethodPost)
		return
	}
	if !s.allowAction(response, request, "onboarding-draft-start", time.Hour, 10) {
		return
	}
	var body struct{}
	if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
		return
	}
	cookie, err := request.Cookie(publicReferralCookie)
	if err != nil || !validPublicReferralCookieValue(strings.TrimSpace(cookie.Value)) {
		writeProblem(response, http.StatusNotFound, "not_found", "No eligible starting path was found.")
		return
	}
	origin, err := s.store.GetPublicStartingPath(
		request.Context(),
		s.publicReferralFingerprint(strings.TrimSpace(cookie.Value)),
		time.Now().UTC(),
	)
	if err != nil {
		writeStoreError(response, err)
		return
	}
	draft, err := s.store.SaveOnboardingDraft(
		request.Context(),
		current.Account.ID,
		uuid.NewString(),
		0,
		1,
		onboardingSeedPayload(origin, s.cfg.RootDomain),
		time.Now().UTC(),
	)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeProblem(response, http.StatusConflict, "conflict", "An existing onboarding draft takes precedence.")
			return
		}
		writeStoreError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"draft": draft})
}

// onboardingSeedPayload builds the private, bounded onboarding draft seeded
// from a public starting path. Defaults match normal creation (intermediate,
// 12 minutes, 08:00, active, email off, AI exploration off, step 1) while the
// browser timezone is supplied later by the UI. Attributed provenance is
// display-only and never carried into newsletter creation.
func onboardingSeedPayload(origin store.PublicStartingPath, rootDomain string) store.OnboardingDraftPayload {
	payload := store.OnboardingDraftPayload{
		Name:                 truncateRunes(strings.TrimSpace(origin.NewsletterName), 80),
		Topic:                truncateRunes(strings.TrimSpace(origin.NewsletterTopic), 400),
		LearnerLevel:         "intermediate",
		LearnerGoal:          truncateRunes(strings.TrimSpace(origin.NewsletterGoal), 500),
		LessonMinutes:        12,
		ScheduleTime:         "08:00",
		Active:               true,
		EmailEnabled:         false,
		AIExplorationEnabled: false,
		SourceMode:           domain.SourceModeDiscovered,
		SourceReviewMode:     domain.SourceReviewBeforeLesson,
		Attribution: &store.OnboardingAttribution{
			DossierPublicID: origin.PublicID,
			Title:           truncateRunes(strings.TrimSpace(origin.Title), 200),
			CanonicalURL: "https://" + origin.SiteUsername + "." + rootDomain +
				"/d/" + url.PathEscape(origin.PublicID) + "/" + url.PathEscape(origin.PublicSlug),
			OwnerName: truncateRunes(strings.TrimSpace(origin.OwnerDisplayName), 80),
		},
	}
	if len(origin.Sources) == 0 {
		return payload
	}
	payload.SourceMode = domain.SourceModeHybrid
	payload.Sources = make([]domain.SourceDefinition, 0, len(origin.Sources))
	for _, source := range origin.Sources {
		payload.Sources = append(payload.Sources, domain.SourceDefinition{
			Name:  truncateRunes(strings.TrimSpace(source.Title), 120),
			URL:   source.CanonicalURL,
			Limit: 8,
		})
	}
	return payload
}

func truncateRunes(value string, maximum int) string {
	runes := []rune(value)
	if len(runes) > maximum {
		return string(runes[:maximum])
	}
	return value
}
