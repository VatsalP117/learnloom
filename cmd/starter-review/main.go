package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/VatsalP117/learnloom/internal/editorial"
)

func main() {
	manifestPath := flag.String(
		"manifest",
		"docs/release-evidence/starter-path-review-v3.json",
		"path to the starter-path human review manifest",
	)
	validateOnly := flag.Bool(
		"validate-only",
		false,
		"validate packet structure without claiming the human release gate passed",
	)
	flag.Parse()

	raw, err := os.ReadFile(*manifestPath)
	if err != nil {
		fail(err)
	}
	var manifest editorial.StarterReviewManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		fail(fmt.Errorf("decode starter review manifest: %w", err))
	}
	if *validateOnly {
		if err := editorial.ValidateStarterReviewManifest(manifest); err != nil {
			fail(err)
		}
		fmt.Printf("%s: %d starter review packets are structurally valid; human release evidence is not implied\n", manifest.Version, len(manifest.Reviews))
		return
	}
	if err := editorial.ValidateStarterReviewRelease(manifest); err != nil {
		fail(err)
	}
	fmt.Printf("%s: starter-path human release gate passed\n", manifest.Version)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
