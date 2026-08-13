package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/loadverify"
)

func main() {
	targets := flag.String("targets", "", "comma-separated staging GET URLs")
	rate := flag.Int("rate", 0, "forecast request rate per second")
	concurrency := flag.Int("concurrency", 0, "forecast peak concurrency")
	duration := flag.Duration("duration", 0, "test duration (10s to 24h)")
	maxErrors := flag.Float64("max-error-percent", 1, "maximum failed request percentage")
	maxP95 := flag.Duration("max-p95", 2*time.Second, "maximum aggregate p95 latency")
	minRate := flag.Float64("min-rate-percent", 90, "minimum achieved percentage of forecast request rate")
	allowProduction := flag.Bool("allow-production", false, "explicitly authorize load against learnloom.blog production hosts")
	stagingHost := flag.String("staging-host", "", "exact dedicated staging hostname required for every target")
	flag.Parse()

	report, err := loadverify.Run(context.Background(), loadverify.Config{
		Targets:           strings.Split(strings.TrimSpace(*targets), ","),
		RequestsPerSecond: *rate, Concurrency: *concurrency, Duration: *duration,
		MaxErrorPercent: *maxErrors, MaxP95: *maxP95,
		MinRatePercent:  *minRate,
		BearerToken:     os.Getenv("LOAD_VERIFY_BEARER_TOKEN"),
		AllowProduction: *allowProduction,
		StagingHost:     *stagingHost,
	}, nil)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if encodeErr := encoder.Encode(report); encodeErr != nil {
		fmt.Fprintln(os.Stderr, "encode load report:", encodeErr)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
