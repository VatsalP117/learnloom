package httpapp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	svix "github.com/svix/svix-webhooks/go"
)

type clerkWebhookEvent struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	Data      struct {
		ID                    string `json:"id"`
		PrimaryEmailAddressID string `json:"primary_email_address_id"`
		Banned                bool   `json:"banned"`
		Locked                bool   `json:"locked"`
		EmailAddresses        []struct {
			ID           string `json:"id"`
			EmailAddress string `json:"email_address"`
			Verification *struct {
				Status string `json:"status"`
			} `json:"verification"`
		} `json:"email_addresses"`
	} `json:"data"`
}

func (s *Server) handleClerkWebhook(
	response http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		methodNotAllowed(response, http.MethodPost)
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, s.cfg.MaxRequestBodyBytes)
	body, err := io.ReadAll(request.Body)
	if err != nil {
		var maximum *http.MaxBytesError
		if errors.As(err, &maximum) {
			writeProblem(response, http.StatusRequestEntityTooLarge, "request_too_large", "The webhook is too large.")
		} else {
			writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The webhook body is invalid.")
		}
		return
	}
	verifier, err := svix.NewWebhook(s.cfg.ClerkWebhookSecret)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	if err := verifier.Verify(body, request.Header); err != nil {
		writeProblem(response, http.StatusBadRequest, "invalid_webhook_signature", "The webhook signature is invalid.")
		return
	}
	eventID := strings.TrimSpace(request.Header.Get("svix-id"))
	if eventID == "" {
		writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The webhook event ID is missing.")
		return
	}
	var event clerkWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The webhook payload is invalid.")
		return
	}
	if event.Data.ID == "" || event.Type == "" || event.Timestamp < 1 {
		writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The webhook payload is incomplete.")
		return
	}
	now := time.Now().UTC()
	fresh, err := s.store.BeginWebhook(request.Context(), eventID, event.Type, now)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	if !fresh {
		response.WriteHeader(http.StatusNoContent)
		return
	}
	processErr := s.processClerkEvent(request, event)
	if completionErr := s.store.CompleteWebhook(
		request.Context(),
		eventID,
		processErr,
		now,
	); completionErr != nil {
		s.logger.ErrorContext(
			request.Context(),
			"complete Clerk webhook",
			"event_id", eventID,
			"error", completionErr,
		)
	}
	if processErr != nil {
		s.internalError(response, request, processErr)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (s *Server) processClerkEvent(
	request *http.Request,
	event clerkWebhookEvent,
) error {
	switch event.Type {
	case "user.created", "user.updated":
		status := domain.AccountActive
		if event.Data.Banned || event.Data.Locked {
			status = domain.AccountSuspended
		}
		_, err := s.store.SyncAccountIdentity(
			request.Context(),
			event.Data.ID,
			verifiedPrimaryEmail(event),
			status,
			event.Timestamp,
		)
		return err
	case "user.deleted":
		_, err := s.store.SyncAccountIdentity(
			request.Context(),
			event.Data.ID,
			"",
			domain.AccountDeleted,
			event.Timestamp,
		)
		return err
	default:
		// Clerk may add events independently. Verified but irrelevant events
		// remain idempotently recorded without mutating Learnloom state.
		return nil
	}
}

func verifiedPrimaryEmail(event clerkWebhookEvent) string {
	for _, address := range event.Data.EmailAddresses {
		if address.ID == event.Data.PrimaryEmailAddressID &&
			address.Verification != nil &&
			address.Verification.Status == "verified" {
			return strings.TrimSpace(address.EmailAddress)
		}
	}
	return ""
}
