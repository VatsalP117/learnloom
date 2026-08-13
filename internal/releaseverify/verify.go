// Package releaseverify checks a deployed release's public parity and security contract.
package releaseverify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxResponseBytes = 2 << 20

var immutableReleasePattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

type Config struct {
	ApexOrigin      string
	WWWOrigin       string
	AppOrigin       string
	ExpectedRelease string
	AllowHTTP       bool
	Timeout         time.Duration
}

type Result struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Report struct {
	Release    string    `json:"release"`
	VerifiedAt time.Time `json:"verifiedAt"`
	Checks     []Result  `json:"checks"`
}

func Verify(ctx context.Context, cfg Config, client *http.Client) (Report, error) {
	apex, err := validateOrigin("apex", cfg.ApexOrigin, cfg.AllowHTTP)
	if err != nil {
		return Report{}, err
	}
	www, err := validateOrigin("www", cfg.WWWOrigin, cfg.AllowHTTP)
	if err != nil {
		return Report{}, err
	}
	app, err := validateOrigin("app", cfg.AppOrigin, cfg.AllowHTTP)
	if err != nil {
		return Report{}, err
	}
	if !immutableReleasePattern.MatchString(cfg.ExpectedRelease) {
		return Report{}, errors.New("expected release must be a full 40- or 64-character lowercase hexadecimal commit SHA")
	}
	if client == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}

	report := Report{Release: cfg.ExpectedRelease, VerifiedAt: time.Now().UTC()}
	get := func(name string, target *url.URL) ([]byte, http.Header, error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
		if err != nil {
			return nil, nil, err
		}
		response, err := client.Do(request)
		if err != nil {
			return nil, nil, fmt.Errorf("%s request: %w", name, err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
		if err != nil {
			return nil, nil, fmt.Errorf("%s body: %w", name, err)
		}
		if len(body) > maxResponseBytes {
			return nil, nil, fmt.Errorf("%s response exceeds %d bytes", name, maxResponseBytes)
		}
		if response.StatusCode != http.StatusOK {
			return nil, nil, fmt.Errorf("%s returned HTTP %d", name, response.StatusCode)
		}
		if err := verifyReleaseHeaders(response.Header, cfg.ExpectedRelease); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", name, err)
		}
		report.Checks = append(report.Checks, Result{Name: name, URL: target.String()})
		return body, response.Header, nil
	}

	apexBody, _, err := get("apex marketing", resolve(apex, "/"))
	if err != nil {
		return Report{}, err
	}
	marketing := string(apexBody)
	for _, marker := range []string{"Give us a topic.", "Build my learning path"} {
		if !strings.Contains(marketing, marker) {
			return Report{}, fmt.Errorf("apex marketing is missing current marker %q", marker)
		}
	}
	for _, stale := range []string{"Urban Systems", "Maya's learning"} {
		if strings.Contains(marketing, stale) {
			return Report{}, fmt.Errorf("apex marketing contains stale marker %q", stale)
		}
	}

	for _, endpoint := range []struct {
		name   string
		path   string
		status string
	}{
		{name: "application health", path: "/healthz", status: "ok"},
		{name: "application readiness", path: "/readyz", status: "ready"},
	} {
		body, _, err := get(endpoint.name, resolve(app, endpoint.path))
		if err != nil {
			return Report{}, err
		}
		var payload struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(body, &payload); err != nil || payload.Status != endpoint.status {
			return Report{}, fmt.Errorf("%s returned an invalid status payload", endpoint.name)
		}
	}

	_, appHeaders, err := get("application shell", resolve(app, "/"))
	if err != nil {
		return Report{}, err
	}
	robots := strings.ToLower(appHeaders.Get("X-Robots-Tag"))
	if !strings.Contains(robots, "noindex") || !strings.Contains(robots, "nofollow") {
		return Report{}, errors.New("application shell must return X-Robots-Tag with noindex and nofollow")
	}

	redirectClient := *client
	redirectClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, resolve(www, "/").String(), nil)
	if err != nil {
		return Report{}, err
	}
	response, err := redirectClient.Do(request)
	if err != nil {
		return Report{}, fmt.Errorf("www redirect request: %w", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusMovedPermanently && response.StatusCode != http.StatusPermanentRedirect {
		return Report{}, fmt.Errorf("www redirect returned HTTP %d, want 301 or 308", response.StatusCode)
	}
	if response.Header.Get("Location") != resolve(apex, "/").String() {
		return Report{}, fmt.Errorf("www redirect location %q does not equal apex %q", response.Header.Get("Location"), resolve(apex, "/").String())
	}
	report.Checks = append(report.Checks, Result{Name: "www canonical redirect", URL: request.URL.String()})

	return report, nil
}

func validateOrigin(name, raw string, allowHTTP bool) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, fmt.Errorf("%s origin is invalid", name)
	}
	if parsed.Scheme != "https" && !(allowHTTP && parsed.Scheme == "http") {
		return nil, fmt.Errorf("%s origin must use HTTPS", name)
	}
	parsed.Path = ""
	return parsed, nil
}

func resolve(origin *url.URL, requestPath string) *url.URL {
	result := *origin
	result.Path = requestPath
	return &result
}

func verifyReleaseHeaders(header http.Header, expectedRelease string) error {
	if got := header.Get("X-Learnloom-Release"); got != expectedRelease {
		return fmt.Errorf("X-Learnloom-Release = %q, want %q", got, expectedRelease)
	}
	hsts := strings.ToLower(header.Get("Strict-Transport-Security"))
	if !strings.Contains(hsts, "includesubdomains") {
		return errors.New("HSTS is missing includeSubDomains")
	}
	maxAge := int64(0)
	for _, directive := range strings.Split(hsts, ";") {
		key, value, found := strings.Cut(strings.TrimSpace(directive), "=")
		if found && key == "max-age" {
			maxAge, _ = strconv.ParseInt(value, 10, 64)
		}
	}
	if maxAge < 31_536_000 {
		return errors.New("HSTS max-age must be at least one year")
	}
	if !strings.Contains(strings.ToLower(header.Get("Cache-Control")), "no-store") {
		return errors.New("Cache-Control must include no-store")
	}
	return nil
}
