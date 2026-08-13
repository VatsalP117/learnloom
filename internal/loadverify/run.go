// Package loadverify runs bounded, thresholded staging load verification.
package loadverify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const maxResponseBytes = 1 << 20

type Config struct {
	Targets           []string
	RequestsPerSecond int
	Concurrency       int
	Duration          time.Duration
	MaxErrorPercent   float64
	MaxP95            time.Duration
	MinRatePercent    float64
	BearerToken       string
	AllowHTTP         bool
	AllowProduction   bool
	StagingHost       string
}

type Report struct {
	StartedAt           time.Time      `json:"startedAt"`
	CompletedAt         time.Time      `json:"completedAt"`
	DurationSeconds     float64        `json:"durationSeconds"`
	RequestsPerSecond   int            `json:"requestedRatePerSecond"`
	Concurrency         int            `json:"concurrency"`
	TotalRequests       int64          `json:"totalRequests"`
	MinimumRequests     int64          `json:"minimumRequests"`
	AchievedRatePercent float64        `json:"achievedRatePercent"`
	FailedRequests      int64          `json:"failedRequests"`
	ErrorPercent        float64        `json:"errorPercent"`
	P50Milliseconds     int64          `json:"p50Milliseconds"`
	P95Milliseconds     int64          `json:"p95Milliseconds"`
	P99Milliseconds     int64          `json:"p99Milliseconds"`
	Targets             []TargetReport `json:"targets"`
	Passed              bool           `json:"passed"`
}

type TargetReport struct {
	URL             string  `json:"url"`
	Requests        int64   `json:"requests"`
	Failures        int64   `json:"failures"`
	ErrorPercent    float64 `json:"errorPercent"`
	P95Milliseconds int64   `json:"p95Milliseconds"`
}

type observation struct {
	target   int
	duration time.Duration
	failed   bool
}

func Run(ctx context.Context, cfg Config, client *http.Client) (Report, error) {
	return run(ctx, cfg, client, 10*time.Second)
}

func run(
	ctx context.Context,
	cfg Config,
	client *http.Client,
	minimumDuration time.Duration,
) (Report, error) {
	targets, err := validate(cfg, minimumDuration)
	if err != nil {
		return Report{}, err
	}
	if client == nil {
		client = &http.Client{Timeout: cfg.MaxP95 * 3}
	}
	requestClient := *client
	configuredRedirect := requestClient.CheckRedirect
	requestClient.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) == 0 || !strings.EqualFold(request.URL.Host, via[0].URL.Host) {
			return errors.New("cross-host redirect refused")
		}
		if request.URL.Scheme != "https" && !(cfg.AllowHTTP && request.URL.Scheme == "http") {
			return errors.New("insecure redirect refused")
		}
		if configuredRedirect != nil {
			return configuredRedirect(request, via)
		}
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		return nil
	}
	started := time.Now().UTC()
	jobs := make(chan int, cfg.Concurrency)
	observations := make(chan observation, cfg.Concurrency*2)
	var workers sync.WaitGroup
	for worker := 0; worker < cfg.Concurrency; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for targetIndex := range jobs {
				startedRequest := time.Now()
				failed := execute(ctx, &requestClient, targets[targetIndex], cfg.BearerToken)
				observations <- observation{target: targetIndex, duration: time.Since(startedRequest), failed: failed}
			}
		}()
	}
	go func() {
		interval := time.Second / time.Duration(cfg.RequestsPerSecond)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		stop := time.NewTimer(cfg.Duration)
		defer stop.Stop()
		defer close(jobs)
		index := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop.C:
				return
			case <-ticker.C:
				select {
				case jobs <- index % len(targets):
					index++
				case <-ctx.Done():
					return
				case <-stop.C:
					return
				}
			}
		}
	}()
	go func() {
		workers.Wait()
		close(observations)
	}()

	allDurations := make([]time.Duration, 0, cfg.RequestsPerSecond*int(cfg.Duration.Seconds()))
	targetDurations := make([][]time.Duration, len(targets))
	targetFailures := make([]int64, len(targets))
	var total, failed atomic.Int64
	for value := range observations {
		total.Add(1)
		allDurations = append(allDurations, value.duration)
		targetDurations[value.target] = append(targetDurations[value.target], value.duration)
		if value.failed {
			failed.Add(1)
			targetFailures[value.target]++
		}
	}
	completed := time.Now().UTC()
	report := Report{
		StartedAt: started, CompletedAt: completed,
		DurationSeconds:   completed.Sub(started).Seconds(),
		RequestsPerSecond: cfg.RequestsPerSecond, Concurrency: cfg.Concurrency,
		TotalRequests: total.Load(), FailedRequests: failed.Load(),
		ErrorPercent:    percent(failed.Load(), total.Load()),
		P50Milliseconds: percentile(allDurations, 0.50).Milliseconds(),
		P95Milliseconds: percentile(allDurations, 0.95).Milliseconds(),
		P99Milliseconds: percentile(allDurations, 0.99).Milliseconds(),
	}
	expectedRequests := float64(cfg.RequestsPerSecond) * cfg.Duration.Seconds()
	report.MinimumRequests = int64(math.Floor(
		expectedRequests * cfg.MinRatePercent / 100,
	))
	report.AchievedRatePercent = 100 * float64(report.TotalRequests) / expectedRequests
	for index, target := range targets {
		requests := int64(len(targetDurations[index]))
		report.Targets = append(report.Targets, TargetReport{
			URL: target.String(), Requests: requests, Failures: targetFailures[index],
			ErrorPercent:    percent(targetFailures[index], requests),
			P95Milliseconds: percentile(targetDurations[index], 0.95).Milliseconds(),
		})
	}
	report.Passed = report.TotalRequests >= report.MinimumRequests &&
		report.ErrorPercent <= cfg.MaxErrorPercent &&
		time.Duration(report.P95Milliseconds)*time.Millisecond <= cfg.MaxP95
	if !report.Passed {
		return report, fmt.Errorf(
			"load gate failed: requests=%d minimum=%d achieved_rate_percent=%.2f error_percent=%.2f p95=%s",
			report.TotalRequests, report.MinimumRequests,
			report.AchievedRatePercent, report.ErrorPercent,
			time.Duration(report.P95Milliseconds)*time.Millisecond,
		)
	}
	return report, nil
}

func validate(cfg Config, minimumDuration time.Duration) ([]*url.URL, error) {
	if len(cfg.Targets) == 0 || len(cfg.Targets) > 20 {
		return nil, errors.New("provide between 1 and 20 target URLs")
	}
	if cfg.RequestsPerSecond < 1 || cfg.RequestsPerSecond > 1000 || cfg.Concurrency < 1 || cfg.Concurrency > 1000 {
		return nil, errors.New("request rate and concurrency must each be between 1 and 1000")
	}
	if cfg.Duration < minimumDuration || cfg.Duration > 24*time.Hour {
		return nil, fmt.Errorf("duration must be between %s and 24 hours", minimumDuration)
	}
	if cfg.MaxErrorPercent < 0 || cfg.MaxErrorPercent > 100 || cfg.MaxP95 <= 0 ||
		cfg.MinRatePercent <= 0 || cfg.MinRatePercent > 100 {
		return nil, errors.New("error and latency thresholds are invalid")
	}
	result := make([]*url.URL, 0, len(cfg.Targets))
	for _, raw := range cfg.Targets {
		parsed, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, fmt.Errorf("target %q is invalid", raw)
		}
		if parsed.Scheme != "https" && !(cfg.AllowHTTP && parsed.Scheme == "http") {
			return nil, fmt.Errorf("target %q must use HTTPS", raw)
		}
		host := strings.ToLower(parsed.Hostname())
		stagingHost := strings.ToLower(strings.TrimSpace(cfg.StagingHost))
		if stagingHost != "" && host != stagingHost {
			return nil, fmt.Errorf("target %q does not match the declared staging host %q", raw, stagingHost)
		}
		protectedProductionHost := host == "learnloom.blog" || host == "www.learnloom.blog" ||
			host == "app.learnloom.blog" ||
			(strings.HasSuffix(host, ".learnloom.blog") && strings.Count(host, ".") == 2 && host != stagingHost)
		if !cfg.AllowProduction && protectedProductionHost {
			return nil, fmt.Errorf("target %q is production; pass the explicit production override only with authorization", raw)
		}
		result = append(result, parsed)
	}
	return result, nil
}

func execute(ctx context.Context, client *http.Client, target *url.URL, token string) bool {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return true
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return true
	}
	defer response.Body.Close()
	readBytes, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, maxResponseBytes+1))
	return readErr != nil || readBytes > maxResponseBytes || response.StatusCode < 200 || response.StatusCode >= 400
}

func percentile(values []time.Duration, quantile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	copyValues := append([]time.Duration(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i] < copyValues[j] })
	index := int(float64(len(copyValues)-1) * quantile)
	return copyValues[index]
}

func percent(numerator, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return 100 * float64(numerator) / float64(denominator)
}
