package loadverify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunPassesBoundedHealthyWorkload(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Millisecond)
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	report, err := run(context.Background(), Config{
		Targets:           []string{server.URL + "/healthz", server.URL + "/readyz"},
		RequestsPerSecond: 100, Concurrency: 4, Duration: 150 * time.Millisecond,
		MaxErrorPercent: 0, MaxP95: 100 * time.Millisecond,
		MinRatePercent: 70, AllowHTTP: true,
	}, nil, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.TotalRequests < 5 || len(report.Targets) != 2 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestRunFailsErrorAndLatencyGates(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		if calls.Add(1)%2 == 0 {
			http.Error(response, "failed", http.StatusInternalServerError)
			return
		}
		time.Sleep(20 * time.Millisecond)
		response.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	report, err := run(context.Background(), Config{
		Targets: []string{server.URL}, RequestsPerSecond: 100, Concurrency: 4,
		Duration: 150 * time.Millisecond, MaxErrorPercent: 1,
		MaxP95: 5 * time.Millisecond, MinRatePercent: 50, AllowHTTP: true,
	}, nil, 100*time.Millisecond)
	if err == nil || report.Passed || report.FailedRequests == 0 || report.P95Milliseconds < 5 {
		t.Fatalf("expected failed workload, report=%+v err=%v", report, err)
	}
}

func TestRunRefusesCrossHostRedirect(t *testing.T) {
	t.Parallel()
	destination := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(destination.Close)
	source := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Redirect(response, nil, destination.URL, http.StatusFound)
	}))
	t.Cleanup(source.Close)

	report, err := run(context.Background(), Config{
		Targets: []string{source.URL}, RequestsPerSecond: 50, Concurrency: 2,
		Duration: 120 * time.Millisecond, MaxErrorPercent: 0,
		MaxP95: 100 * time.Millisecond, MinRatePercent: 50, AllowHTTP: true,
	}, nil, 100*time.Millisecond)
	if err == nil || report.FailedRequests == 0 {
		t.Fatalf("cross-host redirect should fail: report=%+v err=%v", report, err)
	}
	var targetFailures int64
	for _, target := range report.Targets {
		targetFailures += target.Failures
	}
	if report.FailedRequests != targetFailures {
		t.Fatalf("global failures=%d target failures=%d", report.FailedRequests, targetFailures)
	}
}

func TestValidateRequiresExplicitProductionOverrideAndForecast(t *testing.T) {
	t.Parallel()
	base := Config{
		Targets:           []string{"https://app.learnloom.blog/healthz"},
		RequestsPerSecond: 1, Concurrency: 1, Duration: 10 * time.Second,
		MaxErrorPercent: 1, MaxP95: 2 * time.Second, MinRatePercent: 90,
	}
	if _, err := validate(base, 10*time.Second); err == nil || !strings.Contains(err.Error(), "production") {
		t.Fatalf("production should require override: %v", err)
	}
	base.AllowProduction = true
	if _, err := validate(base, 10*time.Second); err != nil {
		t.Fatalf("explicit production override failed: %v", err)
	}
	base.AllowProduction = false
	base.Targets = []string{"https://staging.learnloom.blog/healthz"}
	base.StagingHost = "staging.learnloom.blog"
	if _, err := validate(base, 10*time.Second); err != nil {
		t.Fatalf("declared staging host should be accepted: %v", err)
	}
	base.Targets = []string{"https://app.learnloom.blog/healthz"}
	if _, err := validate(base, 10*time.Second); err == nil || !strings.Contains(err.Error(), "declared staging host") {
		t.Fatalf("target outside declared staging host should fail: %v", err)
	}
	base.RequestsPerSecond = 0
	if _, err := validate(base, 10*time.Second); err == nil {
		t.Fatal("missing forecast rate should fail")
	}
}

func TestPercentileAndPercentEmpty(t *testing.T) {
	t.Parallel()
	if percentile(nil, 0.95) != 0 || percent(1, 0) != 0 {
		t.Fatal("empty aggregates must be zero")
	}
}
