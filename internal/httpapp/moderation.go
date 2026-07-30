package httpapp

import (
	"net/http"
	"strings"
	"time"
)

func (s *Server) issueModeration(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	issueID string,
) {
	switch request.Method {
	case http.MethodGet:
		moderation, err := s.store.GetIssueModeration(
			request.Context(),
			current.Account.ID,
			issueID,
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, moderation)
	case http.MethodPost:
		var body struct {
			State  string `json:"state"`
			Reason string `json:"reason"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		body.Reason = strings.TrimSpace(body.Reason)
		if (body.State != "clear" && body.State != "held") ||
			(body.State == "held" && body.Reason == "") ||
			len(body.Reason) > 1000 {
			writeProblem(response, http.StatusBadRequest, "invalid_moderation_state", "A hold requires a concise moderation reason.")
			return
		}
		if err := s.store.SetIssueModerationState(
			request.Context(),
			current.Account.ID,
			issueID,
			body.State,
			body.Reason,
			time.Now().UTC(),
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, body)
	default:
		methodNotAllowed(response, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) issueCorrections(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	issueID string,
) {
	if request.Method != http.MethodPost {
		methodNotAllowed(response, http.MethodPost)
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
		return
	}
	body.Body = strings.TrimSpace(body.Body)
	if len(body.Body) < 1 || len(body.Body) > 2000 {
		writeProblem(response, http.StatusBadRequest, "invalid_correction", "A correction must be between 1 and 2,000 characters.")
		return
	}
	correction, err := s.store.AddPublicCorrection(
		request.Context(),
		current.Account.ID,
		issueID,
		body.Body,
		time.Now().UTC(),
	)
	if err != nil {
		writeStoreError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, correction)
}

func (s *Server) correctionAction(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	correctionID string,
) {
	if request.Method != http.MethodDelete {
		methodNotAllowed(response, http.MethodDelete)
		return
	}
	if err := s.store.RetractPublicCorrection(
		request.Context(),
		current.Account.ID,
		correctionID,
		time.Now().UTC(),
	); err != nil {
		writeStoreError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (s *Server) reportResolution(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	reportID string,
) {
	if request.Method != http.MethodPost {
		methodNotAllowed(response, http.MethodPost)
		return
	}
	var body struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
		return
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if (body.Status != "resolved" && body.Status != "dismissed") ||
		len(body.Reason) < 1 || len(body.Reason) > 1000 {
		writeProblem(response, http.StatusBadRequest, "invalid_report_resolution", "A resolution or dismissal requires a concise reason.")
		return
	}
	if err := s.store.ResolvePublicContentReport(
		request.Context(),
		current.Account.ID,
		reportID,
		body.Status,
		body.Reason,
		time.Now().UTC(),
	); err != nil {
		writeStoreError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}
