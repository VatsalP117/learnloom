package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/VatsalP117/learnloom/internal/releaseverify"
)

func main() {
	apex := flag.String("apex-origin", "", "HTTPS apex origin")
	www := flag.String("www-origin", "", "HTTPS www origin")
	app := flag.String("app-origin", "", "HTTPS application origin")
	release := flag.String("expected-release", "", "full immutable git commit SHA")
	timeout := flag.Duration("timeout", 15*time.Second, "per-request timeout")
	flag.Parse()

	report, err := releaseverify.Verify(context.Background(), releaseverify.Config{
		ApexOrigin:      *apex,
		WWWOrigin:       *www,
		AppOrigin:       *app,
		ExpectedRelease: *release,
		Timeout:         *timeout,
	}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "release verification failed:", err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, "encode report:", err)
		os.Exit(1)
	}
}
