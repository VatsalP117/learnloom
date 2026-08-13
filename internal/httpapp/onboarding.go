package httpapp

import (
	"net/http"
	"strconv"
	"time"

	"github.com/VatsalP117/learnloom/internal/store"
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
