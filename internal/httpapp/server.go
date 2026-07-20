package httpapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"path"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/VatsalP117/learnloom/internal/artifact"
	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/store"
	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/google/uuid"
)

type Readiness interface {
	Ready(context.Context) error
}

type Config struct {
	RootDomain          string
	ApexOrigin          string
	AppOrigin           string
	CSRFSecret          string
	ClerkSecretKey      string
	ClerkJWTKey         string
	ClerkWebhookSecret  string
	ClerkFrontendOrigin string
	MaxRequestBodyBytes int64
	MaxNewsletters      int
	DailyAccountLimit   int
	MaxDeliveryAttempts int
	ResendConfigured    bool
	SourceDiscovery     bool
	Static              fs.FS
	Authentication      func(http.Handler) http.Handler
}

type Server struct {
	cfg           Config
	store         *store.Store
	artifacts     *artifact.Store
	logger        *slog.Logger
	authenticated http.Handler
	readiness     []Readiness
	metrics       requestMetrics
}

type requestMetrics struct {
	total       atomic.Uint64
	errors      atomic.Uint64
	rateLimited atomic.Uint64
}

type session struct {
	Account   domain.Account
	SessionID string
}

type sessionKey struct{}
type hostKey struct{}
type requestIDKey struct{}

func NewServer(
	cfg Config,
	database *store.Store,
	artifacts *artifact.Store,
	readiness []Readiness,
	logger *slog.Logger,
) (*Server, error) {
	if database == nil || artifacts == nil {
		return nil, errors.New("hosted HTTP dependencies are required")
	}
	if cfg.RootDomain == "" || cfg.AppOrigin == "" || cfg.ApexOrigin == "" ||
		cfg.CSRFSecret == "" || len(cfg.CSRFSecret) < 32 {
		return nil, errors.New("hosted HTTP configuration is invalid")
	}
	if cfg.Static == nil {
		return nil, errors.New("frontend static files are required")
	}
	if cfg.MaxRequestBodyBytes == 0 {
		cfg.MaxRequestBodyBytes = 1 << 20
	}
	if cfg.MaxNewsletters == 0 {
		cfg.MaxNewsletters = 10
	}
	if cfg.DailyAccountLimit == 0 {
		cfg.DailyAccountLimit = 5
	}
	if cfg.MaxDeliveryAttempts == 0 {
		cfg.MaxDeliveryAttempts = 6
	}
	if logger == nil {
		logger = slog.Default()
	}
	server := &Server{
		cfg: cfg, store: database, artifacts: artifacts,
		logger: logger, readiness: readiness,
	}
	auth := cfg.Authentication
	if auth == nil {
		if cfg.ClerkSecretKey == "" {
			return nil, errors.New("Clerk secret key is required")
		}
		clerk.SetKey(cfg.ClerkSecretKey)
		options := []clerkhttp.AuthorizationOption{
			clerkhttp.AuthorizedPartyMatches(cfg.AppOrigin),
			clerkhttp.AuthorizationFailureHandler(http.HandlerFunc(func(
				response http.ResponseWriter,
				_ *http.Request,
			) {
				writeProblem(response, http.StatusUnauthorized, "authentication_required", "Authentication is required.")
			})),
		}
		if cfg.ClerkJWTKey != "" {
			options = append(options, clerkhttp.JSONWebKey(cfg.ClerkJWTKey))
		}
		auth = clerkhttp.RequireHeaderAuthorization(options...)
	}
	server.authenticated = auth(http.HandlerFunc(server.handleAuthenticated))
	return server, nil
}

func (s *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	started := time.Now()
	s.metrics.total.Add(1)
	requestID := strings.TrimSpace(request.Header.Get("X-Request-ID"))
	if _, err := uuid.Parse(requestID); err != nil {
		requestID = uuid.NewString()
	}
	response.Header().Set("X-Request-ID", requestID)
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	response.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	contextWithID := context.WithValue(request.Context(), requestIDKey{}, requestID)
	request = request.WithContext(contextWithID)
	status := http.StatusOK
	writer := &statusWriter{ResponseWriter: response, status: &status}
	defer func() {
		if recovered := recover(); recovered != nil {
			s.metrics.errors.Add(1)
			s.logger.ErrorContext(
				request.Context(),
				"HTTP panic",
				"request_id", requestID,
				"panic", recovered,
				"stack", string(debug.Stack()),
			)
			if !writer.wroteHeader {
				writeProblem(writer, http.StatusInternalServerError, "internal_error", "An internal error occurred.")
			}
		}
		s.logger.InfoContext(
			request.Context(),
			"HTTP request",
			"request_id", requestID,
			"method", request.Method,
			"path", request.URL.Path,
			"host", request.Host,
			"status", status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	}()
	host, err := ClassifyHost(request.Host, s.cfg.RootDomain)
	if err != nil {
		writeProblem(writer, http.StatusMisdirectedRequest, "misdirected_request", "The request Host is not allowed.")
		return
	}
	request = request.WithContext(context.WithValue(request.Context(), hostKey{}, host))
	switch host.Kind {
	case HostWWW:
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			methodNotAllowed(writer, http.MethodGet, http.MethodHead)
			return
		}
		http.Redirect(
			writer,
			request,
			s.cfg.ApexOrigin+request.URL.RequestURI(),
			http.StatusPermanentRedirect,
		)
	case HostApex:
		s.handleApex(writer, request)
	case HostApp:
		s.handleApp(writer, request)
	case HostSite:
		s.handleReading(writer, request, host)
	default:
		writeProblem(writer, http.StatusMisdirectedRequest, "misdirected_request", "The request Host is not allowed.")
	}
}

func (s *Server) handleApex(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		methodNotAllowed(response, http.MethodGet, http.MethodHead)
		return
	}
	if strings.HasPrefix(request.URL.Path, "/assets/") {
		s.serveStatic(response, request, strings.TrimPrefix(request.URL.Path, "/"))
		return
	}
	if request.URL.Path != "/" && request.URL.Path != "/marketing" {
		writeProblem(response, http.StatusNotFound, "not_found", "Page not found.")
		return
	}
	s.serveIndex(response, request)
}

func (s *Server) handleApp(response http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/healthz":
		writeJSON(response, http.StatusOK, map[string]string{"status": "ok"})
		return
	case "/readyz":
		s.handleReady(response, request)
		return
	case "/metrics":
		s.handleMetrics(response)
		return
	case "/webhooks/clerk":
		s.handleClerkWebhook(response, request)
		return
	}
	if strings.HasPrefix(request.URL.Path, "/assets/") {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			methodNotAllowed(response, http.MethodGet, http.MethodHead)
			return
		}
		s.serveStatic(response, request, strings.TrimPrefix(request.URL.Path, "/"))
		return
	}
	if request.URL.Path == "/" ||
		strings.HasPrefix(request.URL.Path, "/sign-in") ||
		strings.HasPrefix(request.URL.Path, "/sign-up") {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			methodNotAllowed(response, http.MethodGet, http.MethodHead)
			return
		}
		s.serveIndex(response, request)
		return
	}
	if strings.HasPrefix(request.URL.Path, "/api/") ||
		strings.HasPrefix(request.URL.Path, "/issues/") {
		s.authenticated.ServeHTTP(response, request)
		return
	}
	if request.Method == http.MethodGet {
		s.serveIndex(response, request)
		return
	}
	writeProblem(response, http.StatusNotFound, "not_found", "Page not found.")
}

func (s *Server) handleAuthenticated(
	response http.ResponseWriter,
	request *http.Request,
) {
	claims, ok := clerk.SessionClaimsFromContext(request.Context())
	if !ok || claims == nil || claims.Subject == "" || claims.SessionID == "" {
		writeProblem(response, http.StatusUnauthorized, "authentication_required", "Authentication is required.")
		return
	}
	account, err := s.store.EnsureAccount(request.Context(), claims.Subject)
	if err != nil {
		if errors.Is(err, store.ErrForbidden) {
			writeProblem(response, http.StatusForbidden, "account_unavailable", "This Account is unavailable.")
			return
		}
		s.internalError(response, request, err)
		return
	}
	request = request.WithContext(context.WithValue(
		request.Context(),
		sessionKey{},
		session{Account: account, SessionID: claims.SessionID},
	))
	if request.Method != http.MethodGet && request.Method != http.MethodHead &&
		request.Method != http.MethodOptions {
		if request.Header.Get("Origin") != s.cfg.AppOrigin {
			writeProblem(response, http.StatusForbidden, "origin_rejected", "The request origin is not allowed.")
			return
		}
		if !hmac.Equal(
			[]byte(request.Header.Get("X-CSRF-Token")),
			[]byte(s.csrfToken(claims.SessionID)),
		) {
			writeProblem(response, http.StatusForbidden, "csrf_rejected", "The CSRF token is invalid.")
			return
		}
		if !strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") {
			writeProblem(response, http.StatusUnsupportedMediaType, "unsupported_media_type", "JSON is required.")
			return
		}
	}
	s.handleControl(response, request)
}

func (s *Server) csrfToken(sessionID string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.CSRFSecret))
	_, _ = mac.Write([]byte("learnloom-csrf\x00" + sessionID))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) serveIndex(response http.ResponseWriter, request *http.Request) {
	body, err := fs.ReadFile(s.cfg.Static, "index.html")
	if err != nil {
		s.internalError(response, request, fmt.Errorf("read frontend index: %w", err))
		return
	}
	s.applyAppCSP(response)
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(http.StatusOK)
	if request.Method != http.MethodHead {
		_, _ = response.Write(body)
	}
}

func (s *Server) serveStatic(
	response http.ResponseWriter,
	request *http.Request,
	name string,
) {
	clean := path.Clean(name)
	if clean != name || !strings.HasPrefix(clean, "assets/") {
		writeProblem(response, http.StatusNotFound, "not_found", "Asset not found.")
		return
	}
	body, err := fs.ReadFile(s.cfg.Static, clean)
	if err != nil {
		writeProblem(response, http.StatusNotFound, "not_found", "Asset not found.")
		return
	}
	contentType := mime.TypeByExtension(path.Ext(clean))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	response.Header().Set("Content-Type", contentType)
	response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	response.WriteHeader(http.StatusOK)
	if request.Method != http.MethodHead {
		_, _ = response.Write(body)
	}
}

func (s *Server) applyAppCSP(response http.ResponseWriter) {
	clerkOrigin := strings.TrimRight(s.cfg.ClerkFrontendOrigin, "/")
	sources := "'self'"
	if clerkOrigin != "" {
		sources += " " + clerkOrigin
	}
	response.Header().Set(
		"Content-Security-Policy",
		"default-src 'self'; script-src "+sources+
			" https://challenges.cloudflare.com; connect-src "+sources+
			"; img-src 'self' data: https://img.clerk.com; style-src 'self' 'unsafe-inline'; "+
			"font-src 'self'; frame-src https://challenges.cloudflare.com; "+
			"base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'self'",
	)
}

func (s *Server) handleReady(response http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 3*time.Second)
	defer cancel()
	for _, dependency := range s.readiness {
		if err := dependency.Ready(ctx); err != nil {
			s.logger.ErrorContext(ctx, "readiness failed", "error", err)
			writeJSON(response, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
	}
	writeJSON(response, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleMetrics(response http.ResponseWriter) {
	response.Header().Set("Content-Type", "text/plain; version=0.0.4")
	response.Header().Set("Cache-Control", "no-store")
	_, _ = fmt.Fprintf(
		response,
		"# TYPE learnloom_http_requests_total counter\nlearnloom_http_requests_total %d\n"+
			"# TYPE learnloom_http_errors_total counter\nlearnloom_http_errors_total %d\n"+
			"# TYPE learnloom_http_rate_limited_total counter\nlearnloom_http_rate_limited_total %d\n",
		s.metrics.total.Load(),
		s.metrics.errors.Load(),
		s.metrics.rateLimited.Load(),
	)
}

func (s *Server) internalError(
	response http.ResponseWriter,
	request *http.Request,
	err error,
) {
	s.metrics.errors.Add(1)
	s.logger.ErrorContext(
		request.Context(),
		"HTTP operation failed",
		"request_id", requestID(request.Context()),
		"error", err,
	)
	writeProblem(response, http.StatusInternalServerError, "internal_error", "An internal error occurred.")
}

func sessionFrom(ctx context.Context) session {
	value, _ := ctx.Value(sessionKey{}).(session)
	return value
}

func requestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}

func clientAddress(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return request.RemoteAddr
	}
	return host
}

type statusWriter struct {
	http.ResponseWriter
	status      *int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	*w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}
