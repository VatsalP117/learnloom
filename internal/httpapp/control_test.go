package httpapp

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/failure"
	"github.com/VatsalP117/learnloom/internal/store"
)

func TestDecodeNewsletterInputSupportsTopicOnlyDefaults(t *testing.T) {
	server := &Server{cfg: Config{MaxRequestBodyBytes: 1 << 20}}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/newsletters",
		strings.NewReader(`{
			"topic":"LLM inference",
			"sourceMode":"discovered",
			"templateId":"ai-systems-evidence",
			"templateVersion":2,
			"timeZone":"Asia/Kolkata"
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	input, ok := server.decodeNewsletterInput(response, request)
	if !ok {
		t.Fatalf("decode failed: status=%d body=%s", response.Code, response.Body.String())
	}
	if input.SourceMode != domain.SourceModeDiscovered ||
		input.ScheduleHour != 8 || input.ScheduleMinute != 0 ||
		input.TemplateID != "ai-systems-evidence" || input.TemplateVersion != 2 ||
		!input.Active || input.SiteVisible || len(input.Sources) != 0 {
		t.Fatalf("input=%#v", input)
	}
}

func TestRenderLessonExportKeepsNotesSeparateFromArtifact(t *testing.T) {
	t.Parallel()
	export := renderLessonExport("# Frozen lesson\n", []store.LessonNote{{
		Kind:       "question",
		QuotedText: "Claim line one\nClaim line two",
		Body:       "What evidence would change this?",
	}})
	for _, expected := range []string{
		"# Frozen lesson",
		"## Your notes",
		"### Question",
		"> Claim line one",
		"> Claim line two",
		"What evidence would change this?",
	} {
		if !strings.Contains(export, expected) {
			t.Fatalf("export missing %q: %s", expected, export)
		}
	}
}

func TestSitePayloadIncludesSearchIndexingPreference(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{RootDomain: "learnloom.blog"}}
	payload := server.sitePayload(&domain.PersonalSite{
		Username:       "maya",
		Visibility:     domain.SitePublic,
		SearchIndexing: true,
	})
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"searchIndexing":true`) {
		t.Fatalf("site payload = %s", encoded)
	}
}

func TestPublicAnalyticsPeriodValidation(t *testing.T) {
	t.Parallel()
	for _, value := range []int{7, 30, 90} {
		if !validPublicAnalyticsPeriod(value) {
			t.Errorf("period %d should be valid", value)
		}
	}
	for _, value := range []int{0, 1, 31, 365} {
		if validPublicAnalyticsPeriod(value) {
			t.Errorf("period %d should be invalid", value)
		}
	}
}

func TestIssueCursorRoundTrip(t *testing.T) {
	t.Parallel()
	cursor := &store.WorkspaceIssueCursor{
		CreatedAt: time.Date(2026, 7, 24, 3, 15, 45, 123, time.UTC),
		IssueID:   "40cd6201-3df1-4a69-aa23-c609b0920923",
	}
	encoded := encodeIssueCursor(cursor)
	decoded, err := decodeIssueCursor(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.CreatedAt.Equal(cursor.CreatedAt) || decoded.IssueID != cursor.IssueID {
		t.Fatalf("decoded=%#v, want %#v", decoded, cursor)
	}
}

func TestIssueCursorRejectsMalformedValues(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"not-base64",
		base64.RawURLEncoding.EncodeToString([]byte(`{"createdAt":"2026-07-24T03:00:00Z","issueId":"not-a-uuid"}`)),
	} {
		if _, err := decodeIssueCursor(raw); err == nil {
			t.Fatalf("decodeIssueCursor(%q) succeeded", raw)
		}
	}
}

func TestValidWebVital(t *testing.T) {
	t.Parallel()
	if !validWebVital("LCP", 1234.5, "good", "navigate", "/library") {
		t.Fatal("valid LCP metric was rejected")
	}
	for _, test := range []struct {
		name   string
		value  float64
		rating string
		page   string
	}{
		{name: "FID", value: 10, rating: "good", page: "/"},
		{name: "CLS", value: -1, rating: "good", page: "/"},
		{name: "INP", value: 10, rating: "unknown", page: "/"},
		{name: "LCP", value: 10, rating: "good", page: "library"},
	} {
		if validWebVital(test.name, test.value, test.rating, "navigate", test.page) {
			t.Fatalf("invalid metric was accepted: %#v", test)
		}
	}
}

func TestClerkSessionTokenSupportsAPIsAndPageNavigations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		authorization string
		cookie        string
		want          string
	}{
		{
			name:          "bearer token for API request",
			authorization: "Bearer api-token",
			cookie:        "cookie-token",
			want:          "api-token",
		},
		{
			name:   "session cookie for page navigation",
			cookie: "cookie-token",
			want:   "cookie-token",
		},
		{
			name:          "empty bearer falls back to session cookie",
			authorization: "Bearer ",
			cookie:        "cookie-token",
			want:          "cookie-token",
		},
		{
			name:          "unrelated authorization scheme is ignored",
			authorization: "Basic credentials",
			want:          "",
		},
		{name: "anonymous request", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/issues/issue-1", nil)
			if test.authorization != "" {
				request.Header.Set("Authorization", test.authorization)
			}
			if test.cookie != "" {
				request.AddCookie(&http.Cookie{Name: "__session", Value: test.cookie})
			}
			if got := clerkSessionToken(request); got != test.want {
				t.Fatalf("clerkSessionToken() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDecodeNewsletterInputKeepsBackwardCompatibleProvidedMode(t *testing.T) {
	server := &Server{cfg: Config{MaxRequestBodyBytes: 1 << 20}}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/newsletters",
		strings.NewReader(`{
			"topic":"LLM inference",
			"timeZone":"UTC",
			"scheduleTime":"09:30",
			"active":false,
			"sources":[{"name":"Docs","url":"https://example.com/docs","limit":8}]
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	input, ok := server.decodeNewsletterInput(response, request)
	if !ok {
		t.Fatalf("decode failed: status=%d body=%s", response.Code, response.Body.String())
	}
	if input.SourceMode != domain.SourceModeProvided ||
		input.ScheduleHour != 9 || input.ScheduleMinute != 30 || input.Active {
		t.Fatalf("input=%#v", input)
	}
}

func TestIssuePayloadsNeverExposeInternalFailureDetails(t *testing.T) {
	t.Parallel()
	issues := []domain.Issue{{
		ID: "issue-1", Status: domain.IssueFailed,
		Error:            "We couldn’t prepare this lesson. We’ve been notified, and you can retry now.",
		FailureCode:      "model_contract_unsatisfied",
		FailureCategory:  "content_quality",
		FailureStage:     "editor",
		FailureRetryable: true,
		IncidentID:       "incident-1",
		Delivery: &domain.DeliveryReceipt{
			Status: domain.DeliveryFailed,
			Error:  "secret provider delivery diagnostic",
		},
	}}
	payload, err := json.Marshal(issuePayloads(issues))
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	if strings.Contains(body, "secret provider delivery diagnostic") ||
		!strings.Contains(body, "model_contract_unsatisfied") ||
		!strings.Contains(body, "We couldn’t prepare this lesson") {
		t.Fatalf("unsafe Issue payload: %s", body)
	}
}

func TestIssuePayloadsExplainEvidenceDeferralsWithoutInternalDetail(t *testing.T) {
	t.Parallel()
	issues := []domain.Issue{{
		ID: "issue-deferred", Status: domain.IssueDeferred,
		Error:           failure.PublicNoEvidence,
		FailureCode:     "no_worthwhile_evidence",
		FailureCategory: string(failure.CategoryInsufficientEvidence),
		FailureStage:    "source_intelligence",
		IncidentID:      "incident-deferred",
	}}
	payload, err := json.Marshal(issuePayloads(issues))
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	for _, expected := range []string{
		`"status":"deferred"`,
		`"failureCode":"no_worthwhile_evidence"`,
		failure.PublicNoEvidence,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("deferred payload missing %q: %s", expected, body)
		}
	}
}

func TestSourceCatalogPayloadExplainsRoleWithoutNumericTrustClaim(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(sourceCatalogPayloads([]domain.SourceCatalogItem{{
		ID: "source-1", DisplayName: "Official reference",
		CanonicalURL: "https://docs.example.com/reference",
		Origin:       domain.SourceOriginDiscovered, Scope: domain.SourceScopeExact,
		State: domain.SourceStateActive, Health: "healthy",
		DiscoveryReason: "Primary reference for maintained documentation",
		Role:            domain.SourceRoleOfficialPrimary, RankingVersion: "source-rank-v2",
	}}))
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	for _, expected := range []string{
		`"role":"official_primary"`,
		`"rankingVersion":"source-rank-v2"`,
		"Primary reference for maintained documentation",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("source payload missing %q: %s", expected, body)
		}
	}
	if strings.Contains(body, "rankScore") || strings.Contains(body, "authorityScore") {
		t.Fatalf("source payload exposed a misleading numeric trust score: %s", body)
	}
}
