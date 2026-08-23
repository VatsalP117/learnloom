package httpapp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/store"
)

func TestDecoratePublicModerationEscapesCorrectionsAndAddsReportForm(t *testing.T) {
	t.Parallel()
	document := `<!doctype html><html><head></head><body><main>Lesson</main></body></html>`
	output := decoratePublicModeration(document, store.PublicIssue{
		PublicID: "dossier-123",
	}, []store.PublicCorrection{{
		Body:      `<script>alert("no")</script>`,
		CreatedAt: time.Date(2026, time.July, 30, 0, 0, 0, 0, time.UTC),
	}})
	for _, expected := range []string{
		`Publisher corrections`,
		`&lt;script&gt;alert(&#34;no&#34;)&lt;/script&gt;`,
		`action="/report/dossier-123"`,
		`name="website"`,
		`30 Jul 2026`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("decorated document missing %q: %s", expected, output)
		}
	}
	if strings.Contains(output, `<script>alert`) {
		t.Fatal("correction HTML was not escaped")
	}
}

func TestDecoratePublicModerationUsesNeutralPalette(t *testing.T) {
	t.Parallel()
	document := `<!doctype html><html><head></head><body><main>Lesson</main></body></html>`
	output := decoratePublicModeration(document, store.PublicIssue{
		PublicID: "dossier-123",
	}, []store.PublicCorrection{{Body: "A note", CreatedAt: time.Now().UTC()}})
	for _, oldToken := range []string{
		"496b4c", "1d2c22", "344039", "69736c", "fffef9",
		"rgba(23,33,27", "border-radius:14px",
	} {
		if strings.Contains(output, oldToken) {
			t.Fatalf("moderation decoration still carries old palette token %q", oldToken)
		}
	}
	for _, expected := range []string{
		"#151719", "#2e3238", "#5c626a", "border-radius:3px", "min-height:44px",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("moderation decoration missing %q", expected)
		}
	}
}

func TestPublicReporterFingerprintIsStableAndDoesNotRevealAddress(t *testing.T) {
	t.Parallel()
	server := &Server{cfg: Config{CSRFSecret: strings.Repeat("s", 32)}}
	request := httptest.NewRequest(http.MethodPost, "https://maya.example/report/id", nil)
	request.RemoteAddr = "203.0.113.9:443"
	request.Header.Set("User-Agent", "test-agent")
	first := server.publicReporterFingerprint(request)
	second := server.publicReporterFingerprint(request)
	if first != second || len(first) != 64 {
		t.Fatalf("unexpected fingerprints %q %q", first, second)
	}
	if strings.Contains(first, "203.0.113.9") {
		t.Fatal("fingerprint exposed the client address")
	}
}

func TestReadingCSPAllowsOnlySameOriginForms(t *testing.T) {
	t.Parallel()
	response := httptest.NewRecorder()
	(&Server{}).applyReadingHeaders(response, true)
	policy := response.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "form-action 'self'") {
		t.Fatalf("reading CSP does not constrain reports to same-origin: %q", policy)
	}
}
