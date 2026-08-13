package httpapp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/artifact"
	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/store"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

func (s *Server) handleControl(
	response http.ResponseWriter,
	request *http.Request,
) {
	route := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	current := sessionFrom(request.Context())
	if request.URL.Path == "/api/me" {
		if request.Method != http.MethodGet {
			methodNotAllowed(response, http.MethodGet)
			return
		}
		var (
			site          *domain.PersonalSite
			notifications store.NotificationPreferences
		)
		group, context := errgroup.WithContext(request.Context())
		group.Go(func() error {
			var err error
			site, err = s.store.GetSite(context, current.Account.ID)
			return err
		})
		group.Go(func() error {
			var err error
			notifications, err = s.store.GetNotificationPreferences(
				context,
				current.Account.ID,
			)
			return err
		})
		if err := group.Wait(); err != nil {
			s.internalError(response, request, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{
			"csrfToken":     s.csrfToken(current.SessionID),
			"site":          s.sitePayload(site),
			"primaryEmail":  current.Account.PrimaryEmail,
			"notifications": notifications,
			"capabilities": map[string]bool{
				"sourceDiscovery": s.cfg.SourceDiscovery,
			},
		})
		return
	}
	if request.URL.Path == "/api/me/notifications" {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, http.MethodPost)
			return
		}
		var body struct {
			WeeklyRecap     bool   `json:"weeklyRecap"`
			ReentryReminder bool   `json:"reentryReminder"`
			TimeZone        string `json:"timeZone"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		preferences, err := s.store.UpdateNotificationPreferences(
			request.Context(),
			current.Account.ID,
			body.WeeklyRecap,
			body.ReentryReminder,
			body.TimeZone,
			time.Now().UTC(),
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{
			"notifications": preferences,
		})
		return
	}
	if request.URL.Path == "/api/me/public-analytics" {
		if request.Method != http.MethodGet {
			methodNotAllowed(response, http.MethodGet)
			return
		}
		periodDays := 30
		if raw := request.URL.Query().Get("days"); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || !validPublicAnalyticsPeriod(value) {
				writeProblem(response, http.StatusBadRequest, "invalid_period", "The analytics period must be 7, 30, or 90 days.")
				return
			}
			periodDays = value
		}
		analytics, err := s.store.GetPublicGrowthAnalytics(
			request.Context(), current.Account.ID, periodDays, time.Now().UTC(),
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{"analytics": analytics})
		return
	}
	if request.URL.Path == "/api/me/billing" {
		if request.Method != http.MethodGet {
			methodNotAllowed(response, http.MethodGet)
			return
		}
		entitlement, err := s.store.GetBillingEntitlement(
			request.Context(), current.Account.ID, time.Now().UTC(),
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		if entitlement.PlanID == "free" {
			paywallID := "paywall:" + current.Account.ID + ":" + entitlement.PeriodStart.UTC().Format("2006-01-02")
			if err := s.store.RecordBillingLifecycleEvent(
				request.Context(), current.Account.ID, "paywall_exposed", paywallID, time.Now().UTC(),
			); err != nil {
				s.internalError(response, request, err)
				return
			}
		}
		writeJSON(response, http.StatusOK, map[string]any{
			"billing": entitlement, "commerceAvailable": s.paddleConfigured(),
		})
		return
	}
	if request.URL.Path == "/api/me/billing/checkout" {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, http.MethodPost)
			return
		}
		if !s.paddleConfigured() {
			writeProblem(response, http.StatusServiceUnavailable, "billing_unavailable", "Billing is not available yet.")
			return
		}
		entitlement, err := s.store.GetBillingEntitlement(request.Context(), current.Account.ID, time.Now().UTC())
		if err != nil {
			writeStoreError(response, err)
			return
		}
		if entitlement.PlanID == "pro" && entitlement.SubscriptionStatus != "canceled" &&
			entitlement.SubscriptionStatus != "refunded" {
			writeProblem(response, http.StatusConflict, "billing_already_active", "Your Pro subscription is already active.")
			return
		}
		now := time.Now().UTC()
		pendingID, err := s.store.GetPendingBillingCheckout(request.Context(), current.Account.ID, now)
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		if pendingID != "" {
			checkoutURL, err := s.paddleCheckoutURL(pendingID)
			if err != nil {
				s.internalError(response, request, err)
				return
			}
			writeJSON(response, http.StatusOK, map[string]string{"url": checkoutURL})
			return
		}
		transactionID, checkoutURL, err := s.createPaddleCheckout(request.Context(), current.Account.ID)
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		selectedID, err := s.store.RecordPendingBillingCheckout(
			request.Context(), current.Account.ID, transactionID, now,
		)
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		if selectedID != transactionID {
			checkoutURL, err = s.paddleCheckoutURL(selectedID)
			if err != nil {
				s.internalError(response, request, err)
				return
			}
			transactionID = selectedID
		}
		if err := s.store.RecordBillingLifecycleEvent(request.Context(), current.Account.ID,
			"checkout_started", transactionID, now); err != nil {
			s.internalError(response, request, err)
			return
		}
		writeJSON(response, http.StatusCreated, map[string]string{"url": checkoutURL})
		return
	}
	if request.URL.Path == "/api/me/billing/portal" {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, http.MethodPost)
			return
		}
		if !s.paddleConfigured() {
			writeProblem(response, http.StatusServiceUnavailable, "billing_unavailable", "Billing is not available yet.")
			return
		}
		customerID, err := s.store.GetBillingProviderCustomerID(request.Context(), current.Account.ID)
		if errors.Is(err, store.ErrNotFound) {
			writeProblem(response, http.StatusConflict, "billing_customer_missing", "Start a subscription before opening billing management.")
			return
		}
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		portalURL, err := s.createPaddlePortal(request.Context(), customerID)
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		writeJSON(response, http.StatusCreated, map[string]string{"url": portalURL})
		return
	}
	if request.URL.Path == "/api/me/billing/feedback" {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, http.MethodPost)
			return
		}
		var body struct {
			Context    string `json:"context"`
			ReasonCode string `json:"reasonCode"`
			Note       string `json:"note"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		if err := s.store.RecordBillingFeedback(
			request.Context(), current.Account.ID, body.Context,
			body.ReasonCode, body.Note, time.Now().UTC(),
		); err != nil {
			if errors.Is(err, store.ErrConflict) {
				writeProblem(response, http.StatusConflict, "billing_feedback_state_mismatch", "This feedback does not match the current plan state.")
				return
			}
			writeStoreError(response, err)
			return
		}
		response.WriteHeader(http.StatusNoContent)
		return
	}
	if len(route) == 3 && route[0] == "api" && route[1] == "usernames" {
		if request.Method != http.MethodGet {
			methodNotAllowed(response, http.MethodGet)
			return
		}
		if !s.allowAction(response, request, "username-check", time.Minute, 30) {
			return
		}
		available, err := s.store.UsernameAvailable(request.Context(), route[2])
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{
			"username": strings.ToLower(route[2]), "available": available,
		})
		return
	}
	if request.URL.Path == "/api/me/site/claim" {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, http.MethodPost)
			return
		}
		if !s.allowAction(response, request, "username-claim", time.Hour, 5) {
			return
		}
		var body struct {
			Username    string `json:"username"`
			DisplayName string `json:"displayName"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		site, err := s.store.ClaimSite(
			request.Context(),
			current.Account.ID,
			body.Username,
			body.DisplayName,
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusCreated, map[string]any{"site": s.sitePayload(&site)})
		return
	}
	if request.URL.Path == "/api/me/site/settings" {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, http.MethodPost)
			return
		}
		var body struct {
			Visibility     domain.SiteVisibility `json:"visibility"`
			DisplayName    *string               `json:"displayName"`
			Description    *string               `json:"description"`
			SearchIndexing *bool                 `json:"searchIndexing"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		site, err := s.store.UpdateSite(
			request.Context(),
			current.Account.ID,
			body.Visibility,
			body.DisplayName,
			body.Description,
			body.SearchIndexing,
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{"site": s.sitePayload(&site)})
		return
	}
	if request.URL.Path == "/api/newsletters" {
		switch request.Method {
		case http.MethodGet:
			s.listNewsletters(response, request, current)
		case http.MethodPost:
			s.createNewsletter(response, request, current)
		default:
			methodNotAllowed(response, http.MethodGet, http.MethodPost)
		}
		return
	}
	if request.URL.Path == "/api/onboarding/draft" {
		s.onboardingDraft(response, request, current)
		return
	}
	if request.URL.Path == "/api/sources/validate" {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, http.MethodPost)
			return
		}
		s.validateSources(response, request, current)
		return
	}
	if request.URL.Path == "/api/source-portfolio/preview" {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, http.MethodPost)
			return
		}
		s.previewSourcePortfolio(response, request, current)
		return
	}
	if request.URL.Path == "/api/workspace" {
		if request.Method != http.MethodGet {
			methodNotAllowed(response, http.MethodGet)
			return
		}
		s.workspaceSnapshot(response, request, current)
		return
	}
	if request.URL.Path == "/api/library" {
		if request.Method != http.MethodGet {
			methodNotAllowed(response, http.MethodGet)
			return
		}
		s.libraryLessons(response, request, current)
		return
	}
	if request.URL.Path == "/api/performance/vitals" {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, http.MethodPost)
			return
		}
		s.recordWebVital(response, request)
		return
	}
	if request.URL.Path == "/api/issues" {
		if request.Method != http.MethodGet {
			methodNotAllowed(response, http.MethodGet)
			return
		}
		s.listWorkspaceIssues(response, request, current)
		return
	}
	if len(route) == 4 && route[0] == "api" && route[1] == "reviews" &&
		route[3] == "assess" {
		if request.Method != http.MethodPost {
			methodNotAllowed(response, http.MethodPost)
			return
		}
		var body struct {
			Assessment     store.ReviewAssessment `json:"assessment"`
			IdempotencyKey string                 `json:"idempotencyKey"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		review, err := s.store.AssessReview(
			request.Context(),
			current.Account.ID,
			route[2],
			body.IdempotencyKey,
			body.Assessment,
			time.Now().UTC(),
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, review)
		return
	}
	if len(route) >= 3 && route[0] == "api" && route[1] == "newsletters" {
		newsletterID := route[2]
		if len(route) == 6 && route[3] == "sources" {
			s.sourceAction(
				response,
				request,
				current,
				newsletterID,
				route[4],
				route[5],
			)
			return
		}
		if len(route) == 3 {
			switch request.Method {
			case http.MethodGet:
				s.newsletterDetail(response, request, current, newsletterID)
			case http.MethodPut:
				s.updateNewsletter(response, request, current, newsletterID)
			default:
				methodNotAllowed(response, http.MethodGet, http.MethodPut)
			}
			return
		}
		if len(route) == 4 {
			s.newsletterAction(
				response,
				request,
				current,
				newsletterID,
				route[3],
			)
			return
		}
	}
	if len(route) >= 3 && route[0] == "api" && route[1] == "issues" {
		if len(route) == 3 {
			if request.Method != http.MethodGet {
				methodNotAllowed(response, http.MethodGet)
				return
			}
			s.issueDetail(response, request, current, route[2])
			return
		}
		if len(route) == 4 {
			if route[3] == "notes" {
				s.issueNotes(response, request, current, route[2])
				return
			}
			if route[3] == "corrections" {
				s.issueCorrections(response, request, current, route[2])
				return
			}
			if route[3] == "moderation" {
				s.issueModeration(response, request, current, route[2])
				return
			}
			if route[3] == "export" {
				s.issueExport(response, request, current, route[2])
				return
			}
			s.issueAction(response, request, current, route[2], route[3])
			return
		}
		if len(route) == 5 && route[3] == "retrievals" {
			s.lessonRetrieval(response, request, current, route[2], route[4])
			return
		}
	}
	if len(route) == 3 && route[0] == "api" && route[1] == "notes" {
		if request.Method != http.MethodDelete {
			methodNotAllowed(response, http.MethodDelete)
			return
		}
		if err := s.store.DeleteLessonNote(
			request.Context(),
			current.Account.ID,
			route[2],
		); err != nil {
			writeStoreError(response, err)
			return
		}
		response.WriteHeader(http.StatusNoContent)
		return
	}
	if len(route) == 3 && route[0] == "api" && route[1] == "corrections" {
		s.correctionAction(response, request, current, route[2])
		return
	}
	if len(route) == 4 && route[0] == "api" && route[1] == "reports" &&
		route[3] == "resolve" {
		s.reportResolution(response, request, current, route[2])
		return
	}
	if len(route) == 2 && route[0] == "issues" && request.Method == http.MethodGet {
		s.issuePreview(response, request, current, route[1])
		return
	}
	writeProblem(response, http.StatusNotFound, "not_found", "The requested route was not found.")
}

func (s *Server) lessonRetrieval(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	issueID, promptKey string,
) {
	if request.Method != http.MethodPost && request.Method != http.MethodPut {
		methodNotAllowed(response, http.MethodPost, http.MethodPut)
		return
	}
	var body struct {
		Response string `json:"response"`
		Skipped  bool   `json:"skipped"`
	}
	if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
		return
	}
	input := store.LessonRetrievalInput{
		PromptKey: promptKey,
		Response:  body.Response,
		Skipped:   body.Skipped,
	}
	var result store.LessonRetrievalResponse
	var err error
	if request.Method == http.MethodPut {
		result, err = s.store.SaveLessonRetrievalDraft(
			request.Context(), current.Account.ID, issueID, input, time.Now().UTC(),
		)
	} else {
		result, err = s.store.RevealLessonRetrieval(
			request.Context(), current.Account.ID, issueID, input, time.Now().UTC(),
		)
	}
	if err != nil {
		writeStoreError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (s *Server) issueExport(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	issueID string,
) {
	if request.Method != http.MethodGet {
		methodNotAllowed(response, http.MethodGet)
		return
	}
	issue, err := s.store.GetIssue(request.Context(), current.Account.ID, issueID)
	if err != nil {
		writeStoreError(response, err)
		return
	}
	if issue.Status != domain.IssueGenerated || issue.ArtifactKey == "" {
		writeProblem(response, http.StatusConflict, "issue_not_generated", "The Issue has no generated Dossier.")
		return
	}
	artifactValue, err := s.artifacts.Get(request.Context(), issue.ArtifactKey)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	notes, err := s.store.ListLessonNotes(
		request.Context(),
		current.Account.ID,
		issueID,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	response.Header().Set("Cache-Control", "private, no-store")
	format := request.URL.Query().Get("format")
	if format == "json" {
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		response.Header().Set("Content-Disposition", `attachment; filename="learnloom-lesson.json"`)
		writeJSON(response, http.StatusOK, map[string]any{
			"version": 1,
			"issue": map[string]any{
				"id": issue.ID, "title": issue.Title, "createdAt": issue.CreatedAt,
			},
			"dossier": artifactValue.Dossier,
			"notes":   notes,
		})
		return
	}
	if format != "" && format != "markdown" {
		writeProblem(response, http.StatusBadRequest, "invalid_export_format", "Export format must be markdown or json.")
		return
	}
	export := renderLessonExport(artifactValue.Markdown, notes)
	response.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	response.Header().Set("Content-Disposition", `attachment; filename="learnloom-lesson.md"`)
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write([]byte(export))
}

func renderLessonExport(markdown string, notes []store.LessonNote) string {
	var export strings.Builder
	export.WriteString(markdown)
	if len(notes) == 0 {
		return export.String()
	}
	export.WriteString("\n\n---\n\n## Your notes\n")
	for _, note := range notes {
		title := "Note"
		if note.Kind != "" {
			title = strings.ToUpper(note.Kind[:1]) + note.Kind[1:]
		}
		export.WriteString("\n### " + title + "\n\n")
		if note.QuotedText != "" {
			for _, line := range strings.Split(note.QuotedText, "\n") {
				export.WriteString("> " + line + "\n")
			}
			export.WriteString("\n")
		}
		export.WriteString(note.Body + "\n")
	}
	return export.String()
}

func (s *Server) issueNotes(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	issueID string,
) {
	switch request.Method {
	case http.MethodGet:
		notes, err := s.store.ListLessonNotes(
			request.Context(),
			current.Account.ID,
			issueID,
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{"notes": notes})
	case http.MethodPost:
		var body struct {
			Kind       string `json:"kind"`
			AnchorType string `json:"anchorType"`
			AnchorID   string `json:"anchorId"`
			Body       string `json:"body"`
			QuotedText string `json:"quotedText"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		note, err := s.store.CreateLessonNote(
			request.Context(),
			current.Account.ID,
			issueID,
			store.LessonNoteInput{
				Kind:       body.Kind,
				AnchorType: body.AnchorType,
				AnchorID:   body.AnchorID,
				Body:       body.Body,
				QuotedText: body.QuotedText,
			},
			time.Now().UTC(),
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusCreated, note)
	default:
		methodNotAllowed(response, http.MethodGet, http.MethodPost)
	}
}

type sourceValidationResult struct {
	Name         string   `json:"name"`
	URL          string   `json:"url"`
	Status       string   `json:"status"`
	ItemCount    int      `json:"itemCount"`
	SampleTitles []string `json:"sampleTitles,omitempty"`
	Message      string   `json:"message,omitempty"`
}

func (s *Server) validateSources(
	response http.ResponseWriter,
	request *http.Request,
	current session,
) {
	validator := s.sourceValidator()
	if validator == nil {
		writeProblem(
			response,
			http.StatusServiceUnavailable,
			"source_validation_unavailable",
			"Source validation is temporarily unavailable.",
		)
		return
	}
	if !s.allowAction(response, request, "source-validation", time.Hour, 20) {
		return
	}
	var body struct {
		Sources           []domain.SourceDefinition `json:"sources"`
		OnboardingDraftID string                    `json:"onboardingDraftId"`
	}
	if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
		return
	}
	if len(body.Sources) < 1 || len(body.Sources) > 12 {
		writeProblem(
			response,
			http.StatusBadRequest,
			"invalid_sources",
			"Validate from one to twelve sources at a time.",
		)
		return
	}
	results := runSourceValidation(request.Context(), validator, body.Sources)
	allReady := true
	for _, result := range results {
		if result.Status != "ready" {
			allReady = false
			break
		}
	}
	if allReady && body.OnboardingDraftID != "" {
		if err := s.store.RecordOnboardingPreviewReached(
			request.Context(), current.Account.ID, body.OnboardingDraftID, time.Now().UTC(),
		); err != nil {
			writeStoreError(response, err)
			return
		}
	}
	writeJSON(response, http.StatusOK, map[string]any{"sources": results})
}

func runSourceValidation(
	ctx context.Context,
	validator SourceValidator,
	definitions []domain.SourceDefinition,
) []sourceValidationResult {
	results := make([]sourceValidationResult, len(definitions))
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(3)
	for index, definition := range definitions {
		index, definition := index, definition
		group.Go(func() error {
			result := sourceValidationResult{
				Name: definition.Name,
				URL:  definition.URL,
			}
			items, warnings, err := validator(ctx, definition)
			if err != nil {
				result.Status = "unavailable"
				result.Message = sourceValidationMessage(err, warnings)
				results[index] = result
				return nil
			}
			result.Status = "ready"
			result.ItemCount = len(items)
			for _, item := range items {
				if len(result.SampleTitles) == 2 {
					break
				}
				if title := strings.TrimSpace(item.Title); title != "" {
					result.SampleTitles = append(result.SampleTitles, title)
				}
			}
			if len(warnings) > 0 {
				result.Message = sourceValidationMessage(nil, warnings)
			}
			results[index] = result
			return nil
		})
	}
	_ = group.Wait()
	return results
}

func sourceValidationMessage(err error, warnings []string) string {
	message := ""
	if err != nil {
		message = err.Error()
	} else if len(warnings) > 0 {
		message = warnings[0]
	}
	message = strings.TrimSpace(message)
	if len(message) > 240 {
		message = message[:237] + "..."
	}
	if message == "" {
		return "No recent items were found."
	}
	return message
}

func (s *Server) recordWebVital(
	response http.ResponseWriter,
	request *http.Request,
) {
	var body struct {
		Name           string  `json:"name"`
		Value          float64 `json:"value"`
		Rating         string  `json:"rating"`
		NavigationType string  `json:"navigationType"`
		Page           string  `json:"page"`
	}
	if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
		return
	}
	if !validWebVital(body.Name, body.Value, body.Rating, body.NavigationType, body.Page) {
		writeProblem(response, http.StatusBadRequest, "invalid_metric", "The browser performance metric is invalid.")
		return
	}
	s.logger.InfoContext(
		request.Context(),
		"Browser performance metric",
		"metric", body.Name,
		"value", body.Value,
		"rating", body.Rating,
		"navigation_type", body.NavigationType,
		"page", body.Page,
	)
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(http.StatusNoContent)
}

func validWebVital(name string, value float64, rating, navigationType, page string) bool {
	if name != "CLS" && name != "INP" && name != "LCP" {
		return false
	}
	if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}
	if rating != "good" && rating != "needs-improvement" && rating != "poor" {
		return false
	}
	return len(navigationType) <= 40 &&
		strings.HasPrefix(page, "/") &&
		len(page) <= 80
}

func validPublicAnalyticsPeriod(value int) bool {
	return value == 7 || value == 30 || value == 90
}

func (s *Server) workspaceSnapshot(
	response http.ResponseWriter,
	request *http.Request,
	current session,
) {
	var (
		records    []store.NewsletterRecord
		issues     []domain.Issue
		next       *store.WorkspaceIssueCursor
		reviews    []store.WorkspaceReview
		progress   []store.LessonProgress
		retention  store.RetentionState
		todayFocus store.TodayFocus
	)
	group, context := errgroup.WithContext(request.Context())
	group.Go(func() error {
		var err error
		records, err = s.store.ListNewsletters(context, current.Account.ID)
		return err
	})
	group.Go(func() error {
		var err error
		retention, err = s.store.GetRetentionState(
			context,
			current.Account.ID,
			time.Now().UTC(),
		)
		return err
	})
	group.Go(func() error {
		var err error
		issues, next, err = s.store.ListWorkspaceIssuesPage(
			context,
			current.Account.ID,
			24,
			nil,
		)
		return err
	})
	group.Go(func() error {
		var err error
		reviews, err = s.store.ListWorkspaceReviews(
			context,
			current.Account.ID,
			8,
			time.Now().UTC(),
		)
		return err
	})
	group.Go(func() error {
		var err error
		progress, err = s.store.ListRecentLessonProgress(context, current.Account.ID, 24)
		return err
	})
	group.Go(func() error {
		var err error
		todayFocus, err = s.store.RefreshTodayFocus(
			context, current.Account.ID, time.Now().UTC(),
		)
		return err
	})
	if err := group.Wait(); err != nil {
		s.internalError(response, request, err)
		return
	}
	items := make([]map[string]any, 0, len(records))
	active, generated := 0, 0
	for _, record := range records {
		items = append(items, newsletterPayload(record, current.Account.PrimaryEmail))
		if record.Active {
			active++
		}
		generated += record.GeneratedCount
	}
	payload := map[string]any{
		"summary": map[string]int{
			"newsletters": len(records), "active": active, "generated": generated,
		},
		"newsletters":     items,
		"issues":          issuePayloads(issues),
		"nextIssueCursor": encodeIssueCursor(next),
		"reviews":         reviews,
		"lessonProgress":  progress,
		"retention":       retention,
		"todayFocus":      todayFocus,
	}
	writePrivateCacheableJSON(
		response,
		request,
		http.StatusOK,
		payload,
		"private, max-age=0, must-revalidate",
	)
}

func (s *Server) listWorkspaceIssues(
	response http.ResponseWriter,
	request *http.Request,
	current session,
) {
	limit := 40
	if raw := request.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			writeProblem(response, http.StatusBadRequest, "invalid_limit", "The Issue page limit must be between 1 and 100.")
			return
		}
		limit = value
	}
	cursor, err := decodeIssueCursor(request.URL.Query().Get("cursor"))
	if err != nil {
		writeProblem(response, http.StatusBadRequest, "invalid_cursor", "The Issue page cursor is invalid.")
		return
	}
	issues, next, err := s.store.ListWorkspaceIssuesPage(
		request.Context(),
		current.Account.ID,
		limit,
		cursor,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"issues":          issuePayloads(issues),
		"nextIssueCursor": encodeIssueCursor(next),
	})
}

func (s *Server) libraryLessons(
	response http.ResponseWriter,
	request *http.Request,
	current session,
) {
	limit := 24
	if raw := request.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			writeProblem(response, http.StatusBadRequest, "invalid_limit", "The Library page limit must be between 1 and 100.")
			return
		}
		limit = value
	}
	search := strings.TrimSpace(request.URL.Query().Get("q"))
	if len(search) > 120 {
		writeProblem(response, http.StatusBadRequest, "invalid_query", "The Library search must be 120 characters or fewer.")
		return
	}
	filter := store.LibraryFilter(request.URL.Query().Get("filter"))
	if filter == "" {
		filter = store.LibraryAll
	}
	switch filter {
	case store.LibraryAll, store.LibraryUnread, store.LibraryInProgress, store.LibraryCompleted:
	default:
		writeProblem(response, http.StatusBadRequest, "invalid_filter", "The Library filter is invalid.")
		return
	}
	cursor, err := decodeIssueCursor(request.URL.Query().Get("cursor"))
	if err != nil {
		writeProblem(response, http.StatusBadRequest, "invalid_cursor", "The Library cursor is invalid.")
		return
	}
	lessons, next, err := s.store.ListLibraryLessonsPage(
		request.Context(),
		current.Account.ID,
		search,
		filter,
		limit,
		cursor,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"lessons":    lessons,
		"nextCursor": encodeIssueCursor(next),
	})
}

func issuePayloads(issues []domain.Issue) []map[string]any {
	items := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		item := map[string]any{
			"id":                     issue.ID,
			"newsletterId":           issue.NewsletterID,
			"trigger":                issue.Trigger,
			"scheduledLocalDate":     issue.ScheduledLocalDate,
			"status":                 issue.Status,
			"title":                  issue.Title,
			"publicId":               issue.PublicID,
			"publicSlug":             issue.PublicSlug,
			"publicationState":       issue.PublicationState,
			"publicationUpdatedAt":   issue.PublicationUpdatedAt,
			"firstPublishReviewedAt": issue.FirstPublishReviewedAt,
			"publishedAt":            issue.PublishedAt,
			"createdAt":              issue.CreatedAt,
			"startedAt":              issue.StartedAt,
			"completedAt":            issue.CompletedAt,
		}
		if (issue.Status == domain.IssueFailed || issue.Status == domain.IssueDeferred) && issue.Error != "" {
			item["error"] = issue.Error
			item["failureCode"] = issue.FailureCode
			item["failureCategory"] = issue.FailureCategory
			item["failureStage"] = issue.FailureStage
			item["failureRetryable"] = issue.FailureRetryable
			item["incidentId"] = issue.IncidentID
		}
		if issue.Delivery != nil {
			item["delivery"] = map[string]any{
				"status":        issue.Delivery.Status,
				"attemptCount":  issue.Delivery.AttemptCount,
				"completedAt":   issue.Delivery.CompletedAt,
				"nextAttemptAt": issue.Delivery.NextAttempt,
			}
		}
		items = append(items, item)
	}
	return items
}

func sourceCatalogPayloads(items []domain.SourceCatalogItem) []map[string]any {
	payloads := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payload := map[string]any{
			"id":               item.ID,
			"displayName":      item.DisplayName,
			"canonicalUrl":     item.CanonicalURL,
			"origin":           item.Origin,
			"scope":            item.Scope,
			"kind":             item.Kind,
			"state":            item.State,
			"health":           item.Health,
			"discoveryReason":  item.DiscoveryReason,
			"role":             item.Role,
			"rankingVersion":   item.RankingVersion,
			"preference":       item.Preference,
			"lastCheckedAt":    item.LastCheckedAt,
			"lastSuccessfulAt": item.LastSuccessfulAt,
		}
		if item.Error != "" {
			payload["error"] = "This source could not be refreshed."
		}
		payloads = append(payloads, payload)
	}
	return payloads
}

type issueCursorToken struct {
	CreatedAt time.Time `json:"createdAt"`
	IssueID   string    `json:"issueId"`
}

func encodeIssueCursor(cursor *store.WorkspaceIssueCursor) string {
	if cursor == nil {
		return ""
	}
	raw, _ := json.Marshal(issueCursorToken{
		CreatedAt: cursor.CreatedAt,
		IssueID:   cursor.IssueID,
	})
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeIssueCursor(raw string) (*store.WorkspaceIssueCursor, error) {
	if raw == "" {
		return nil, nil
	}
	if len(raw) > 512 {
		return nil, errors.New("Issue cursor is too long")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	var token issueCursorToken
	if err := json.Unmarshal(decoded, &token); err != nil {
		return nil, err
	}
	if token.CreatedAt.IsZero() {
		return nil, errors.New("Issue cursor timestamp is missing")
	}
	if _, err := uuid.Parse(token.IssueID); err != nil {
		return nil, errors.New("Issue cursor ID is invalid")
	}
	return &store.WorkspaceIssueCursor{
		CreatedAt: token.CreatedAt,
		IssueID:   token.IssueID,
	}, nil
}

func (s *Server) listNewsletters(
	response http.ResponseWriter,
	request *http.Request,
	current session,
) {
	records, err := s.store.ListNewsletters(request.Context(), current.Account.ID)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	items := make([]map[string]any, 0, len(records))
	active, generated := 0, 0
	for _, record := range records {
		items = append(items, newsletterPayload(record, current.Account.PrimaryEmail))
		if record.Active {
			active++
		}
		generated += record.GeneratedCount
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"summary": map[string]int{
			"newsletters": len(records), "active": active, "generated": generated,
		},
		"newsletters": items,
	})
}

func (s *Server) createNewsletter(
	response http.ResponseWriter,
	request *http.Request,
	current session,
) {
	if !s.allowAction(response, request, "newsletter-create", time.Hour, 20) {
		return
	}
	input, ok := s.decodeNewsletterInput(response, request)
	if !ok {
		return
	}
	result, err := s.store.CreateNewsletter(
		request.Context(),
		current.Account.ID,
		input,
		s.cfg.MaxNewsletters,
		s.cfg.DailyAccountLimit,
	)
	if err != nil {
		writeStoreError(response, err)
		return
	}
	summary, _ := s.store.GetSourceSummary(request.Context(), result.Newsletter.ID)
	writeJSON(response, http.StatusCreated, map[string]any{
		"newsletter":    newsletterPayload(result.Newsletter, current.Account.PrimaryEmail),
		"issue":         result.FirstIssue,
		"sourceSummary": summary,
	})
}

func (s *Server) updateNewsletter(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	newsletterID string,
) {
	input, ok := s.decodeNewsletterInput(response, request)
	if !ok {
		return
	}
	if input.SourceReviewMode == "" {
		currentRecord, err := s.store.GetNewsletter(
			request.Context(),
			current.Account.ID,
			newsletterID,
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		input.SourceReviewMode = currentRecord.SourceReviewMode
	}
	record, err := s.store.UpdateNewsletter(
		request.Context(),
		current.Account.ID,
		newsletterID,
		input,
	)
	if err != nil {
		writeStoreError(response, err)
		return
	}
	summary, _ := s.store.GetSourceSummary(request.Context(), newsletterID)
	catalog, err := s.store.ListSourceCatalog(
		request.Context(),
		current.Account.ID,
		newsletterID,
		50,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"newsletter":    newsletterPayload(record, current.Account.PrimaryEmail),
		"sourceSummary": summary,
		"sourceCatalog": sourceCatalogPayloads(catalog),
	})
}

func (s *Server) newsletterDetail(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	newsletterID string,
) {
	var (
		record     store.NewsletterRecord
		issues     []domain.Issue
		progress   []store.LessonProgress
		all        []store.NewsletterRecord
		summary    domain.SourceSummary
		catalog    []domain.SourceCatalogItem
		curriculum store.CurriculumProjection
		site       *domain.PersonalSite
	)
	group, context := errgroup.WithContext(request.Context())
	group.Go(func() error {
		var err error
		record, err = s.store.GetNewsletter(context, current.Account.ID, newsletterID)
		return err
	})
	group.Go(func() error {
		var err error
		site, err = s.store.GetSite(context, current.Account.ID)
		return err
	})
	group.Go(func() error {
		var err error
		issues, err = s.store.ListIssues(context, current.Account.ID, newsletterID, 100)
		return err
	})
	group.Go(func() error {
		var err error
		progress, err = s.store.ListLessonProgress(context, current.Account.ID)
		return err
	})
	group.Go(func() error {
		var err error
		all, err = s.store.ListNewsletters(context, current.Account.ID)
		return err
	})
	group.Go(func() error {
		summary, _ = s.store.GetSourceSummary(context, newsletterID)
		return nil
	})
	group.Go(func() error {
		var err error
		catalog, err = s.store.ListSourceCatalog(
			context,
			current.Account.ID,
			newsletterID,
			50,
		)
		return err
	})
	group.Go(func() error {
		var err error
		curriculum, err = s.store.GetCurriculum(
			context,
			current.Account.ID,
			newsletterID,
		)
		return err
	})
	if err := group.Wait(); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeStoreError(response, err)
			return
		}
		s.internalError(response, request, err)
		return
	}
	sidebar := make([]map[string]any, 0, len(all))
	for _, item := range all {
		sidebar = append(sidebar, map[string]any{
			"id": item.ID, "name": item.Name, "active": item.Active,
		})
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"csrfToken":        s.csrfToken(current.SessionID),
		"resendConfigured": s.cfg.ResendConfigured,
		"newsletter":       newsletterPayload(record, current.Account.PrimaryEmail),
		"sourceSummary":    summary,
		"sourceCatalog":    sourceCatalogPayloads(catalog),
		"curriculum":       curriculum,
		"site":             s.sitePayload(site),
		"issues":           issuePayloads(issues),
		"lessonProgress":   progress,
		"newsletters":      sidebar,
	})
}

func (s *Server) newsletterAction(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	newsletterID, action string,
) {
	if request.Method != http.MethodPost {
		methodNotAllowed(response, http.MethodPost)
		return
	}
	switch action {
	case "run":
		if !s.allowAction(response, request, "manual-generation", time.Hour, 10) {
			return
		}
		issue, err := s.store.EnqueueManualIssue(
			request.Context(),
			current.Account.ID,
			newsletterID,
			s.cfg.DailyAccountLimit,
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusAccepted, map[string]any{"issue": issue})
	case "active":
		var body struct {
			Active bool `json:"active"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		if err := s.store.SetNewsletterActive(
			request.Context(),
			current.Account.ID,
			newsletterID,
			body.Active,
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]bool{"active": body.Active})
	case "rhythm":
		var body struct {
			Mode                domain.RhythmMode `json:"mode"`
			SelectedWeekdays    []int             `json:"selectedWeekdays"`
			AutoThrottleEnabled *bool             `json:"autoThrottleEnabled"`
			UnopenedLessonLimit int               `json:"unopenedLessonLimit"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		autoThrottle := true
		if body.AutoThrottleEnabled != nil {
			autoThrottle = *body.AutoThrottleEnabled
		}
		if body.UnopenedLessonLimit == 0 {
			body.UnopenedLessonLimit = 3
		}
		record, err := s.store.SetNewsletterRhythm(
			request.Context(),
			current.Account.ID,
			newsletterID,
			store.RhythmInput{
				Mode:                body.Mode,
				SelectedWeekdays:    body.SelectedWeekdays,
				AutoThrottleEnabled: autoThrottle,
				UnopenedLessonLimit: body.UnopenedLessonLimit,
			},
			time.Now().UTC(),
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{
			"newsletter": newsletterPayload(record, current.Account.PrimaryEmail),
		})
	case "reset-backlog":
		result, err := s.store.ResetNewsletterBacklog(
			request.Context(),
			current.Account.ID,
			newsletterID,
			time.Now().UTC(),
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{
			"dismissedCount": result.DismissedCount,
			"newsletter":     newsletterPayload(result.Newsletter, current.Account.PrimaryEmail),
		})
	case "delivery":
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		if body.Enabled && current.Account.PrimaryEmail == "" {
			writeProblem(response, http.StatusConflict, "verified_email_required", "A verified primary email is required.")
			return
		}
		if err := s.store.SetNewsletterEmail(
			request.Context(),
			current.Account.ID,
			newsletterID,
			body.Enabled,
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]bool{"enabled": body.Enabled})
	case "content":
		var body struct {
			AIExplorationEnabled bool `json:"aiExplorationEnabled"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		if err := s.store.SetNewsletterContent(
			request.Context(),
			current.Account.ID,
			newsletterID,
			body.AIExplorationEnabled,
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, body)
	case "site":
		var body struct {
			Visible bool `json:"visible"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		if err := s.store.SetNewsletterSiteVisible(
			request.Context(),
			current.Account.ID,
			newsletterID,
			body.Visible,
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, body)
	case "publication-default":
		var body struct {
			State             domain.PublicationState `json:"state"`
			AudienceConfirmed bool                    `json:"audienceConfirmed"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		if err := s.store.SetNewsletterPublicationDefault(
			request.Context(), current.Account.ID, newsletterID,
			body.State, body.AudienceConfirmed, time.Now().UTC(),
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, body)
	case "bulk-publication":
		var body struct {
			IssueIDs          []string                `json:"issueIds"`
			State             domain.PublicationState `json:"state"`
			AudienceConfirmed bool                    `json:"audienceConfirmed"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		updated, err := s.store.BulkSetIssuePublication(
			request.Context(), current.Account.ID, newsletterID, body.IssueIDs,
			store.PublicationChange{
				State: body.State, AudienceConfirmed: body.AudienceConfirmed,
				Now: time.Now().UTC(),
			},
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{"updated": updated, "state": body.State})
	case "source-mode":
		var body struct {
			Mode domain.SourceMode `json:"mode"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		if err := s.store.SetNewsletterSourceMode(
			request.Context(),
			current.Account.ID,
			newsletterID,
			body.Mode,
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, body)
	case "source-review-mode":
		var body struct {
			Mode domain.SourceReviewMode `json:"mode"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		if body.Mode != domain.SourceReviewAuto && body.Mode != domain.SourceReviewBeforeLesson {
			writeProblem(response, http.StatusBadRequest, "invalid_source_review_mode", "Source review mode must be auto or review.")
			return
		}
		if err := s.store.SetNewsletterSourceReviewMode(
			request.Context(),
			current.Account.ID,
			newsletterID,
			body.Mode,
			time.Now().UTC(),
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, body)
	case "approve-source-portfolio":
		if err := s.store.ApproveSourcePortfolio(
			request.Context(),
			current.Account.ID,
			newsletterID,
			time.Now().UTC(),
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusAccepted, map[string]string{"status": "queued"})
	case "preparation-wait-exited":
		if err := s.store.RecordPreparationWaitExited(
			request.Context(),
			current.Account.ID,
			newsletterID,
			time.Now().UTC(),
		); err != nil {
			s.internalError(response, request, err)
			return
		}
		writeJSON(response, http.StatusAccepted, map[string]string{"status": "recorded"})
	default:
		writeProblem(response, http.StatusNotFound, "not_found", "The requested action was not found.")
	}
}

func (s *Server) sourceAction(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	newsletterID, sourceID, action string,
) {
	if request.Method != http.MethodPost {
		methodNotAllowed(response, http.MethodPost)
		return
	}
	switch action {
	case "preference":
		var body struct {
			Preference domain.SourcePreference `json:"preference"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		if err := s.store.SetSourcePreference(
			request.Context(),
			current.Account.ID,
			newsletterID,
			sourceID,
			body.Preference,
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, body)
	case "replace":
		var body domain.SourceDefinition
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		replacementID, err := s.store.ReplaceProvidedSource(
			request.Context(),
			current.Account.ID,
			newsletterID,
			sourceID,
			body,
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]string{"sourceId": replacementID})
	default:
		writeProblem(response, http.StatusNotFound, "not_found", "The requested source action was not found.")
	}
}

func (s *Server) issueAction(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	issueID, action string,
) {
	if request.Method != http.MethodPost {
		methodNotAllowed(response, http.MethodPost)
		return
	}
	switch action {
	case "retry-generation":
		if err := s.store.RetryIssue(
			request.Context(),
			current.Account.ID,
			issueID,
			time.Now().UTC(),
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusAccepted, map[string]string{"status": "queued"})
	case "publication":
		var body struct {
			State             domain.PublicationState `json:"state"`
			AudienceConfirmed bool                    `json:"audienceConfirmed"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		issue, err := s.store.SetIssuePublication(
			request.Context(),
			current.Account.ID,
			issueID,
			store.PublicationChange{
				State: body.State, AudienceConfirmed: body.AudienceConfirmed,
				Now: time.Now().UTC(),
			},
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{
			"publicationState":       issue.PublicationState,
			"publicationUpdatedAt":   issue.PublicationUpdatedAt,
			"firstPublishReviewedAt": issue.FirstPublishReviewedAt,
			"publishedAt":            issue.PublishedAt,
		})
	case "retry-delivery":
		if err := s.store.RetryDelivery(
			request.Context(),
			current.Account.ID,
			issueID,
			s.cfg.MaxDeliveryAttempts,
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusAccepted, map[string]string{"status": "pending"})
	case "complete":
		progress, err := s.store.CompleteLesson(
			request.Context(),
			current.Account.ID,
			issueID,
			time.Now().UTC(),
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, progress)
	case "opened":
		if err := s.store.RecordOwnedLessonEvent(
			request.Context(),
			current.Account.ID,
			issueID,
			store.ProductEventLessonOpened,
			time.Now().UTC(),
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusAccepted, map[string]string{"status": "recorded"})
	case "feedback":
		var body struct {
			Difficulty       *string `json:"difficulty"`
			Relevance        *string `json:"relevance"`
			RecallConfidence *string `json:"recallConfidence"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		feedback, err := s.store.SaveLessonFeedback(
			request.Context(),
			current.Account.ID,
			issueID,
			store.LessonFeedbackInput{
				Difficulty:       body.Difficulty,
				Relevance:        body.Relevance,
				RecallConfidence: body.RecallConfidence,
			},
			time.Now().UTC(),
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, feedback)
	case "progress":
		var body struct {
			Progress int `json:"progress"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		if body.Progress < 1 || body.Progress > 99 {
			writeProblem(response, http.StatusBadRequest, "invalid_progress", "Lesson progress must be between 1 and 99.")
			return
		}
		progress, err := s.store.SaveLessonProgress(
			request.Context(),
			current.Account.ID,
			issueID,
			body.Progress,
			time.Now().UTC(),
		)
		if err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, progress)
	default:
		writeProblem(response, http.StatusNotFound, "not_found", "The requested action was not found.")
	}
}

func (s *Server) issuePreview(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	issueID string,
) {
	issue, err := s.store.GetIssue(request.Context(), current.Account.ID, issueID)
	if err != nil {
		writeStoreError(response, err)
		return
	}
	if issue.Status != domain.IssueGenerated || issue.ArtifactKey == "" {
		writeProblem(response, http.StatusConflict, "issue_not_generated", "The Issue has no generated Dossier.")
		return
	}
	artifactValue, err := s.artifacts.Get(request.Context(), issue.ArtifactKey)
	if err != nil {
		if errors.Is(err, artifact.ErrNotFound) {
			writeProblem(
				response,
				http.StatusGone,
				"artifact_unavailable",
				"This lesson file is unavailable. Please prepare the lesson again.",
			)
			return
		}
		s.internalError(response, request, err)
		return
	}
	response.Header().Set(
		"Content-Security-Policy",
		"default-src 'none'; style-src 'unsafe-inline'; img-src https: data:; "+
			"font-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'",
	)
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Cache-Control", "private, no-store")
	_, _ = response.Write([]byte(artifactValue.HTML))
}

func (s *Server) issueDetail(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	issueID string,
) {
	issue, err := s.store.GetIssue(request.Context(), current.Account.ID, issueID)
	if err != nil {
		writeStoreError(response, err)
		return
	}
	if issue.Status != domain.IssueGenerated || issue.ArtifactKey == "" {
		writeProblem(response, http.StatusConflict, "issue_not_generated", "The Issue has no generated Dossier.")
		return
	}
	feedback, err := s.store.GetLessonFeedback(
		request.Context(),
		current.Account.ID,
		issueID,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	notes, err := s.store.ListLessonNotes(
		request.Context(),
		current.Account.ID,
		issueID,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	progress, err := s.store.GetLessonProgress(
		request.Context(), current.Account.ID, issueID,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	retrievals, err := s.store.ListLessonRetrievalResponses(
		request.Context(), current.Account.ID, issueID,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	navigation, err := s.store.GetLessonNavigation(
		request.Context(), current.Account.ID, issueID,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	site, err := s.store.GetSite(request.Context(), current.Account.ID)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	etag := issueDetailETag(issue, feedback, notes, progress, retrievals, navigation)
	if requestETagMatches(request, etag) {
		response.Header().Set("Cache-Control", "private, max-age=300, stale-while-revalidate=3600")
		response.Header().Set("ETag", etag)
		response.Header().Set("Vary", "Authorization")
		response.WriteHeader(http.StatusNotModified)
		return
	}
	artifactValue, err := s.artifacts.Get(request.Context(), issue.ArtifactKey)
	if err != nil {
		if errors.Is(err, artifact.ErrNotFound) {
			writeProblem(
				response,
				http.StatusGone,
				"artifact_unavailable",
				"This lesson file is unavailable. Please prepare the lesson again.",
			)
			return
		}
		s.internalError(response, request, err)
		return
	}
	sources := make([]map[string]any, 0, len(artifactValue.Dossier.Sources))
	for _, source := range artifactValue.Dossier.Sources {
		sourceURL := source.CanonicalURL
		if sourceURL == "" {
			sourceURL = source.URL
		}
		sourceID := source.SourceID
		if sourceID == "" {
			sourceID = fmt.Sprintf("S%d", len(sources)+1)
		}
		sources = append(sources, map[string]any{
			"id":            sourceID,
			"name":          source.Source,
			"url":           sourceURL,
			"summary":       source.Summary,
			"author":        source.Author,
			"publishedAt":   source.PublishedAt,
			"contentSource": source.ContentSource,
		})
	}
	payload := map[string]any{
		"issue":          issue,
		"newsletter":     issue.Newsletter,
		"dossier":        artifactValue.Dossier,
		"sources":        sources,
		"feedback":       feedback,
		"notes":          notes,
		"lessonProgress": progress,
		"retrievals":     retrievals,
		"navigation":     navigation,
		"site":           s.sitePayload(site),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writePrivateJSONWithETag(
		response,
		request,
		http.StatusOK,
		append(body, '\n'),
		etag,
		"private, max-age=300, stale-while-revalidate=3600",
	)
}

func issueDetailETag(
	issue domain.Issue,
	feedback *store.LessonFeedback,
	notes []store.LessonNote,
	progress *store.LessonProgress,
	retrievals []store.LessonRetrievalResponse,
	navigation store.LessonNavigation,
) string {
	checksum := issue.ArtifactSHA256
	if checksum == "" {
		checksum = issue.ArtifactKey
	}
	feedbackVersion := ""
	if feedback != nil {
		feedbackVersion = feedback.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	notesVersion := ""
	if len(notes) > 0 {
		notesVersion = notes[0].UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	progressVersion := ""
	if progress != nil {
		progressVersion = progress.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	retrievalVersion := ""
	if len(retrievals) > 0 {
		retrievalVersion = retrievals[len(retrievals)-1].UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	navigationVersion := ""
	if navigation.Previous != nil {
		navigationVersion += navigation.Previous.IssueID
	}
	if navigation.Next != nil {
		navigationVersion += "\x00" + navigation.Next.IssueID
	}
	if navigation.NextReviewAt != nil {
		navigationVersion += "\x00" + navigation.NextReviewAt.UTC().Format(time.RFC3339Nano)
	}
	value := strings.Join([]string{
		checksum,
		string(issue.PublicationState),
		issue.Title,
		issue.Newsletter.UpdatedAt.UTC().Format(time.RFC3339Nano),
		feedbackVersion,
		notesVersion,
		progressVersion,
		retrievalVersion,
		navigationVersion,
	}, "\x00")
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf(`"%x"`, sum)
}

func (s *Server) decodeNewsletterInput(
	response http.ResponseWriter,
	request *http.Request,
) (store.NewsletterInput, bool) {
	var body struct {
		Name                    string                    `json:"name"`
		Topic                   string                    `json:"topic"`
		LearnerLevel            string                    `json:"learnerLevel"`
		LearnerGoal             string                    `json:"learnerGoal"`
		LessonMinutes           int                       `json:"lessonMinutes"`
		SourceMode              string                    `json:"sourceMode"`
		SourceReviewMode        domain.SourceReviewMode   `json:"sourceReviewMode"`
		Sources                 []domain.SourceDefinition `json:"sources"`
		ScheduleTime            string                    `json:"scheduleTime"`
		TimeZone                string                    `json:"timeZone"`
		Active                  *bool                     `json:"active"`
		EmailEnabled            *bool                     `json:"emailEnabled"`
		AIExplorationEnabled    *bool                     `json:"aiExplorationEnabled"`
		SiteVisible             *bool                     `json:"siteVisible"`
		TemplateID              string                    `json:"templateId"`
		TemplateVersion         int                       `json:"templateVersion"`
		OnboardingDraftID       string                    `json:"onboardingDraftId"`
		OnboardingDraftRevision int64                     `json:"onboardingDraftRevision"`
	}
	if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
		return store.NewsletterInput{}, false
	}
	scheduleTime := body.ScheduleTime
	if scheduleTime == "" {
		scheduleTime = "08:00"
	}
	hour, minute, err := parseScheduleTime(scheduleTime)
	if err != nil {
		writeProblem(response, http.StatusBadRequest, "invalid_schedule", err.Error())
		return store.NewsletterInput{}, false
	}
	mode := body.SourceMode
	if mode == "" && len(body.Sources) > 0 {
		mode = "provided"
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	return store.NewsletterInput{
		Name: body.Name, Topic: body.Topic, LearnerLevel: body.LearnerLevel,
		LearnerGoal: body.LearnerGoal, LessonMinutes: body.LessonMinutes,
		SourceMode: domain.SourceMode(mode), SourceReviewMode: body.SourceReviewMode,
		Sources:      body.Sources,
		ScheduleHour: hour, ScheduleMinute: minute,
		TimeZone: body.TimeZone, Active: active, EmailEnabled: boolValue(body.EmailEnabled),
		AIExplorationEnabled:    boolValue(body.AIExplorationEnabled),
		SiteVisible:             boolValue(body.SiteVisible),
		TemplateID:              body.TemplateID,
		TemplateVersion:         body.TemplateVersion,
		OnboardingDraftID:       body.OnboardingDraftID,
		OnboardingDraftRevision: body.OnboardingDraftRevision,
	}, true
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func newsletterPayload(
	record store.NewsletterRecord,
	primaryEmail string,
) map[string]any {
	recipients := []string{}
	if record.EmailEnabled && primaryEmail != "" {
		recipients = append(recipients, primaryEmail)
	}
	return map[string]any{
		"id": record.ID, "name": record.Name, "topic": record.Topic,
		"learnerLevel": record.LearnerLevel, "learnerGoal": record.LearnerGoal,
		"lessonMinutes": record.LessonMinutes, "sourceMode": record.SourceMode,
		"sourceReviewMode":                   record.SourceReviewMode,
		"sourceApprovedAt":                   record.SourceApprovedAt,
		"sources":                            record.Sources,
		"scheduleTime":                       fmt.Sprintf("%02d:%02d", record.ScheduleHour, record.ScheduleMinute),
		"rhythmMode":                         record.RhythmMode,
		"selectedWeekdays":                   record.SelectedWeekdays,
		"effectiveRhythmMode":                record.EffectiveRhythmMode,
		"autoThrottleEnabled":                record.AutoThrottleEnabled,
		"unopenedLessonLimit":                record.UnopenedLessonLimit,
		"rhythmReason":                       record.RhythmReason,
		"rhythmThrottledAt":                  record.RhythmThrottledAt,
		"lessonPublicationDefault":           record.LessonPublicationDefault,
		"lessonPublicationDefaultReviewedAt": record.LessonPublicationDefaultReviewedAt,
		"timeZone":                           record.TimeZone, "active": record.Active,
		"nextRunAt": record.NextRunAt, "emailEnabled": record.EmailEnabled,
		"emailRecipients":      recipients,
		"aiExplorationEnabled": record.AIExplorationEnabled,
		"publicSlug":           record.PublicSlug, "siteVisible": record.SiteVisible,
		"issueCount": record.IssueCount, "generatedCount": record.GeneratedCount,
		"sentCount":               record.SentCount,
		"capabilityCount":         record.CapabilityCount,
		"recalledCapabilityCount": record.RecalledCapabilityCount,
		"currentGapCount":         record.CurrentGapCount,
	}
}

func (s *Server) sitePayload(site *domain.PersonalSite) any {
	if site == nil {
		return nil
	}
	return map[string]any{
		"username": site.Username, "displayName": site.DisplayName,
		"description": site.Description, "visibility": site.Visibility,
		"searchIndexing": site.SearchIndexing,
		"claimedAt":      site.ClaimedAt,
		"url":            "https://" + site.Username + "." + s.cfg.RootDomain,
	}
}

func (s *Server) allowAction(
	response http.ResponseWriter,
	request *http.Request,
	action string,
	window time.Duration,
	limit int,
) bool {
	current := sessionFrom(request.Context())
	key := current.Account.ID + ":" + clientAddress(request)
	allowed, err := s.store.AllowRequest(
		request.Context(),
		key,
		action,
		window,
		limit,
		time.Now().UTC(),
	)
	if err != nil {
		s.internalError(response, request, err)
		return false
	}
	if !allowed {
		s.metrics.rateLimited.Add(1)
		response.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
		writeProblem(response, http.StatusTooManyRequests, "rate_limited", "Too many requests.")
		return false
	}
	return true
}

func parseScheduleTime(value string) (int, int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, errors.New("scheduleTime must use HH:MM")
	}
	hour, hourErr := strconv.Atoi(parts[0])
	minute, minuteErr := strconv.Atoi(parts[1])
	if hourErr != nil || minuteErr != nil || hour < 0 || hour > 23 ||
		minute < 0 || minute > 59 {
		return 0, 0, errors.New("scheduleTime must be a valid 24-hour time")
	}
	return hour, minute, nil
}
