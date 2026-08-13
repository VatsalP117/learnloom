package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/source"
)

func main() {
	corpusPath := flag.String(
		"corpus",
		"internal/source/testdata/source-evaluation-v1.json",
		"path to the versioned source evaluation corpus",
	)
	validateOnly := flag.Bool(
		"validate-only",
		false,
		"validate the 50-topic seed before human labels are complete",
	)
	capture := flag.Bool("capture", false, "capture a frozen candidate set with the configured SearXNG provider")
	outputPath := flag.String("output", "", "new path for a captured corpus; required with -capture")
	searxngBaseURL := flag.String("searxng-base-url", os.Getenv("SEARXNG_BASE_URL"), "SearXNG origin for capture")
	force := flag.Bool("force", false, "replace an existing capture output explicitly")
	requireGates := flag.Bool("require-gates", false, "exit non-zero unless every source-intelligence launch threshold passes")
	adjudicate := flag.Bool("adjudicate", false, "compare two independent label sets and write a resolved corpus")
	labelSetAPath := flag.String("label-set-a", "", "first independent human-labeled corpus")
	labelSetBPath := flag.String("label-set-b", "", "second independent human-labeled corpus")
	resolutionPath := flag.String("resolution", "", "human-labeled resolved corpus with notes for disagreements")
	adjudicatorRef := flag.String("adjudicator-ref", "", "bounded non-secret adjudicator reference")
	flag.Parse()

	raw, err := os.ReadFile(*corpusPath)
	if err != nil {
		fail(err)
	}
	var corpus source.SourceEvaluationCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		fail(fmt.Errorf("decode source evaluation corpus: %w", err))
	}
	if *adjudicate {
		if strings.TrimSpace(*labelSetAPath) == "" || strings.TrimSpace(*labelSetBPath) == "" ||
			strings.TrimSpace(*resolutionPath) == "" || strings.TrimSpace(*outputPath) == "" ||
			strings.TrimSpace(*adjudicatorRef) == "" {
			fail(errors.New("-adjudicate requires -label-set-a, -label-set-b, -resolution, -output, and -adjudicator-ref"))
		}
		if !*force {
			if _, err := os.Stat(*outputPath); err == nil {
				fail(fmt.Errorf("adjudication output %s already exists; use a new path or explicit -force", *outputPath))
			} else if !os.IsNotExist(err) {
				fail(err)
			}
		}
		labelSetA, rawA := readCorpus(*labelSetAPath)
		labelSetB, rawB := readCorpus(*labelSetBPath)
		resolution, _ := readCorpus(*resolutionPath)
		adjudicated, err := source.AdjudicateSourceCorpora(
			labelSetA, labelSetB, resolution,
			fmt.Sprintf("%x", sha256.Sum256(rawA)),
			fmt.Sprintf("%x", sha256.Sum256(rawB)),
			*adjudicatorRef,
		)
		if err != nil {
			fail(err)
		}
		output, err := json.MarshalIndent(adjudicated, "", "  ")
		if err != nil {
			fail(err)
		}
		if err := os.WriteFile(*outputPath, append(output, '\n'), 0o600); err != nil {
			fail(err)
		}
		fmt.Printf(
			"adjudicated %d topics to %s; agreement=%.3f disagreements=%d resolved=%d\n",
			len(adjudicated.Topics), *outputPath,
			adjudicated.Adjudication.AgreementRate,
			adjudicated.Adjudication.Disagreements,
			adjudicated.Adjudication.Resolved,
		)
		return
	}
	if *capture {
		if strings.TrimSpace(*outputPath) == "" || strings.TrimSpace(*searxngBaseURL) == "" {
			fail(errors.New("-capture requires -output and -searxng-base-url (or SEARXNG_BASE_URL)"))
		}
		if !*force {
			if _, err := os.Stat(*outputPath); err == nil {
				fail(fmt.Errorf("capture output %s already exists; use a new path or explicit -force", *outputPath))
			} else if !os.IsNotExist(err) {
				fail(err)
			}
		}
		searcher, err := source.NewSearXNG(source.SearXNGConfig{
			BaseURL: *searxngBaseURL, Timeout: 10 * time.Second,
		})
		if err != nil {
			fail(err)
		}
		capturedAt := time.Now().UTC()
		corpus.CapturedAt = &capturedAt
		corpus.RankingVersion = "source-rank-v2"
		corpus.LabelStatus = "awaiting_human_labels"
		for index := range corpus.Topics {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			candidates, err := source.CaptureSourceEvaluationTopic(
				ctx, searcher, corpus.Topics[index], 5, 30, 10,
			)
			cancel()
			if err != nil {
				fail(err)
			}
			corpus.Topics[index].Candidates = candidates
			fmt.Fprintf(os.Stderr, "captured %s (%d candidates)\n", corpus.Topics[index].ID, len(candidates))
		}
		corpus.CandidateSnapshotHash = source.SourceEvaluationSnapshotHash(corpus)
		output, err := json.MarshalIndent(corpus, "", "  ")
		if err != nil {
			fail(err)
		}
		output = append(output, '\n')
		if err := os.WriteFile(*outputPath, output, 0o600); err != nil {
			fail(err)
		}
		fmt.Printf("captured %d topics to %s; human labels remain required\n", len(corpus.Topics), *outputPath)
		return
	}
	if *validateOnly {
		if err := source.ValidateSourceCorpusSeed(corpus); err != nil {
			fail(err)
		}
		fmt.Printf("%s: %d representative topics are structurally valid; human labels remain %s\n", corpus.Version, len(corpus.Topics), corpus.LabelStatus)
		return
	}
	metrics, err := source.EvaluateSourceCorpus(corpus)
	if err != nil {
		fail(err)
	}
	output, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(output))
	if *requireGates {
		if err := source.ValidateSourceReleaseGates(metrics); err != nil {
			fail(err)
		}
	}
}

func readCorpus(path string) (source.SourceEvaluationCorpus, []byte) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fail(err)
	}
	var corpus source.SourceEvaluationCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		fail(fmt.Errorf("decode source evaluation corpus %s: %w", path, err))
	}
	return corpus, raw
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
