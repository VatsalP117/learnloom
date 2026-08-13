package releaseverify

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testRelease = "0123456789abcdef0123456789abcdef01234567"

func TestVerifyAcceptsMatchingHealthyRelease(t *testing.T) {
	t.Parallel()
	apex := newReleaseServer(t, func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("<h1>Give us a topic.</h1><a>Build my learning path</a>"))
	})
	app := newReleaseServer(t, func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/healthz":
			_, _ = response.Write([]byte(`{"status":"ok"}`))
		case "/readyz":
			_, _ = response.Write([]byte(`{"status":"ready"}`))
		case "/":
			response.Header().Set("X-Robots-Tag", "noindex, nofollow")
			_, _ = response.Write([]byte("application"))
		default:
			http.NotFound(response, request)
		}
	})
	www := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Location", apex.URL+"/")
		response.WriteHeader(http.StatusPermanentRedirect)
	}))
	t.Cleanup(www.Close)

	report, err := Verify(context.Background(), Config{
		ApexOrigin:      apex.URL,
		WWWOrigin:       www.URL,
		AppOrigin:       app.URL,
		ExpectedRelease: testRelease,
		AllowHTTP:       true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Release != testRelease || len(report.Checks) != 5 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestVerifyRejectsStaleMarketing(t *testing.T) {
	t.Parallel()
	apex := newReleaseServer(t, func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte("Urban Systems"))
	})
	_, err := Verify(context.Background(), Config{
		ApexOrigin:      apex.URL,
		WWWOrigin:       apex.URL,
		AppOrigin:       apex.URL,
		ExpectedRelease: testRelease,
		AllowHTTP:       true,
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "current marker") {
		t.Fatalf("expected stale marketing failure, got %v", err)
	}
}

func TestVerifyRejectsWrongReleaseHeader(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		setReleaseHeaders(response, strings.Repeat("f", 40))
		_, _ = response.Write([]byte("Give us a topic. Build my learning path"))
	}))
	t.Cleanup(server.Close)
	_, err := Verify(context.Background(), Config{
		ApexOrigin:      server.URL,
		WWWOrigin:       server.URL,
		AppOrigin:       server.URL,
		ExpectedRelease: testRelease,
		AllowHTTP:       true,
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "X-Learnloom-Release") {
		t.Fatalf("expected release mismatch, got %v", err)
	}
}

func TestVerifyRejectsMutableReleaseAndProductionHTTP(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "mutable release",
			cfg: Config{
				ApexOrigin: "https://example.com", WWWOrigin: "https://www.example.com",
				AppOrigin: "https://app.example.com", ExpectedRelease: "main",
			},
			want: "full 40- or 64-character",
		},
		{
			name: "http production origin",
			cfg: Config{
				ApexOrigin: "http://example.com", WWWOrigin: "https://www.example.com",
				AppOrigin: "https://app.example.com", ExpectedRelease: testRelease,
			},
			want: "must use HTTPS",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Verify(context.Background(), test.cfg, nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("wanted %q, got %v", test.want, err)
			}
		})
	}
}

func newReleaseServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setReleaseHeaders(response, testRelease)
		handler(response, request)
	}))
	t.Cleanup(server.Close)
	return server
}

func setReleaseHeaders(response http.ResponseWriter, release string) {
	response.Header().Set("X-Learnloom-Release", release)
	response.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	response.Header().Set("Cache-Control", "no-store")
}

func TestValidateOriginRejectsCredentialsAndPaths(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"https://user:secret@example.com",
		"https://example.com/path",
		"https://example.com?query=value",
	} {
		t.Run(fmt.Sprintf("origin_%d", len(raw)), func(t *testing.T) {
			if _, err := validateOrigin("test", raw, false); err == nil {
				t.Fatalf("origin %q should be rejected", raw)
			}
		})
	}
}
