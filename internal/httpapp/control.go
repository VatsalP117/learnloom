package httpapp

import (
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
		site, err := s.store.GetSite(request.Context(), current.Account.ID)
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{
			"csrfToken":    s.csrfToken(current.SessionID),
			"site":         s.sitePayload(site),
			"primaryEmail": current.Account.PrimaryEmail,
			"capabilities": map[string]bool{
				"sourceDiscovery": s.cfg.SourceDiscovery,
			},
		})
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
			Visibility  domain.SiteVisibility `json:"visibility"`
			DisplayName *string               `json:"displayName"`
			Description *string               `json:"description"`
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
	if request.URL.Path == "/api/workspace" {
		if request.Method != http.MethodGet {
			methodNotAllowed(response, http.MethodGet)
			return
		}
		s.workspaceSnapshot(response, request, current)
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
	if len(route) >= 3 && route[0] == "api" && route[1] == "newsletters" {
		newsletterID := route[2]
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
			s.issueAction(response, request, current, route[2], route[3])
			return
		}
	}
	if len(route) == 2 && route[0] == "issues" && request.Method == http.MethodGet {
		s.issuePreview(response, request, current, route[1])
		return
	}
	writeProblem(response, http.StatusNotFound, "not_found", "The requested route was not found.")
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

func (s *Server) workspaceSnapshot(
	response http.ResponseWriter,
	request *http.Request,
	current session,
) {
	var (
		records []store.NewsletterRecord
		issues  []domain.Issue
		next    *store.WorkspaceIssueCursor
		reviews []store.WorkspaceReview
	)
	group, context := errgroup.WithContext(request.Context())
	group.Go(func() error {
		var err error
		records, err = s.store.ListNewsletters(context, current.Account.ID)
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
		reviews, err = s.store.ListWorkspaceReviews(context, current.Account.ID, 8)
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
		"issues":          workspaceIssuePayloads(issues),
		"nextIssueCursor": encodeIssueCursor(next),
		"reviews":         reviews,
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
		"issues":          workspaceIssuePayloads(issues),
		"nextIssueCursor": encodeIssueCursor(next),
	})
}

func workspaceIssuePayloads(issues []domain.Issue) []map[string]any {
	items := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		items = append(items, map[string]any{
			"id":                 issue.ID,
			"newsletterId":       issue.NewsletterID,
			"trigger":            issue.Trigger,
			"scheduledLocalDate": issue.ScheduledLocalDate,
			"status":             issue.Status,
			"title":              issue.Title,
			"error":              issue.Error,
			"publicId":           issue.PublicID,
			"publicSlug":         issue.PublicSlug,
			"publicationState":   issue.PublicationState,
			"createdAt":          issue.CreatedAt,
			"startedAt":          issue.StartedAt,
			"completedAt":        issue.CompletedAt,
		})
	}
	return items
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
		"sourceCatalog": catalog,
	})
}

func (s *Server) newsletterDetail(
	response http.ResponseWriter,
	request *http.Request,
	current session,
	newsletterID string,
) {
	record, err := s.store.GetNewsletter(
		request.Context(),
		current.Account.ID,
		newsletterID,
	)
	if err != nil {
		writeStoreError(response, err)
		return
	}
	issues, err := s.store.ListIssues(
		request.Context(),
		current.Account.ID,
		newsletterID,
		100,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	all, err := s.store.ListNewsletters(request.Context(), current.Account.ID)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	sidebar := make([]map[string]any, 0, len(all))
	for _, item := range all {
		sidebar = append(sidebar, map[string]any{
			"id": item.ID, "name": item.Name, "active": item.Active,
		})
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
		"csrfToken":        s.csrfToken(current.SessionID),
		"resendConfigured": s.cfg.ResendConfigured,
		"newsletter":       newsletterPayload(record, current.Account.PrimaryEmail),
		"sourceSummary":    summary,
		"sourceCatalog":    catalog,
		"issues":           issues,
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
	default:
		writeProblem(response, http.StatusNotFound, "not_found", "The requested action was not found.")
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
			State domain.PublicationState `json:"state"`
		}
		if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
			return
		}
		if err := s.store.SetIssuePublication(
			request.Context(),
			current.Account.ID,
			issueID,
			body.State,
		); err != nil {
			writeStoreError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, body)
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
	etag := issueDetailETag(issue)
	if requestETagMatches(request, etag) {
		response.Header().Set("Cache-Control", "private, max-age=300, stale-while-revalidate=3600")
		response.Header().Set("ETag", etag)
		response.Header().Set("Vary", "Authorization")
		response.WriteHeader(http.StatusNotModified)
		return
	}
	artifactValue, err := s.artifacts.Get(request.Context(), issue.ArtifactKey)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	sources := make([]map[string]string, 0, len(artifactValue.Dossier.Sources))
	for _, source := range artifactValue.Dossier.Sources {
		sourceURL := source.CanonicalURL
		if sourceURL == "" {
			sourceURL = source.URL
		}
		sources = append(sources, map[string]string{
			"name": source.Source,
			"url":  sourceURL,
		})
	}
	payload := map[string]any{
		"issue":      issue,
		"newsletter": issue.Newsletter,
		"dossier":    artifactValue.Dossier,
		"sources":    sources,
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

func issueDetailETag(issue domain.Issue) string {
	checksum := issue.ArtifactSHA256
	if checksum == "" {
		checksum = issue.ArtifactKey
	}
	value := strings.Join([]string{
		checksum,
		string(issue.PublicationState),
		issue.Title,
		issue.Newsletter.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}, "\x00")
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf(`"%x"`, sum)
}

func (s *Server) decodeNewsletterInput(
	response http.ResponseWriter,
	request *http.Request,
) (store.NewsletterInput, bool) {
	var body struct {
		Name                 string                    `json:"name"`
		Topic                string                    `json:"topic"`
		LearnerLevel         string                    `json:"learnerLevel"`
		LearnerGoal          string                    `json:"learnerGoal"`
		LessonMinutes        int                       `json:"lessonMinutes"`
		SourceMode           string                    `json:"sourceMode"`
		Sources              []domain.SourceDefinition `json:"sources"`
		ScheduleTime         string                    `json:"scheduleTime"`
		TimeZone             string                    `json:"timeZone"`
		Active               *bool                     `json:"active"`
		EmailEnabled         *bool                     `json:"emailEnabled"`
		AIExplorationEnabled *bool                     `json:"aiExplorationEnabled"`
		SiteVisible          *bool                     `json:"siteVisible"`
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
		SourceMode: domain.SourceMode(mode), Sources: body.Sources,
		ScheduleHour: hour, ScheduleMinute: minute,
		TimeZone: body.TimeZone, Active: active, EmailEnabled: boolValue(body.EmailEnabled),
		AIExplorationEnabled: boolValue(body.AIExplorationEnabled),
		SiteVisible:          boolValue(body.SiteVisible),
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
		"sources":      record.Sources,
		"scheduleTime": fmt.Sprintf("%02d:%02d", record.ScheduleHour, record.ScheduleMinute),
		"timeZone":     record.TimeZone, "active": record.Active,
		"nextRunAt": record.NextRunAt, "emailEnabled": record.EmailEnabled,
		"emailRecipients":      recipients,
		"aiExplorationEnabled": record.AIExplorationEnabled,
		"publicSlug":           record.PublicSlug, "siteVisible": record.SiteVisible,
		"issueCount": record.IssueCount, "generatedCount": record.GeneratedCount,
		"sentCount": record.SentCount,
	}
}

func (s *Server) sitePayload(site *domain.PersonalSite) any {
	if site == nil {
		return nil
	}
	return map[string]any{
		"username": site.Username, "displayName": site.DisplayName,
		"description": site.Description, "visibility": site.Visibility,
		"claimedAt": site.ClaimedAt,
		"url":       "https://" + site.Username + "." + s.cfg.RootDomain,
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
