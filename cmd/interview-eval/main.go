package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/VatsalP117/learnloom/internal/research"
)

func main() {
	corpusPath := flag.String("corpus", "", "path to the frozen privacy-safe interview corpus")
	requireGate := flag.Bool("require-gate", false, "exit non-zero unless the Phase 1 evidence threshold passes")
	flag.Parse()
	if *corpusPath == "" {
		fail(fmt.Errorf("-corpus is required"))
	}
	raw, err := os.ReadFile(*corpusPath)
	if err != nil {
		fail(err)
	}
	var corpus research.InterviewCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		fail(fmt.Errorf("decode interview corpus: %w", err))
	}
	report, err := research.EvaluateInterviews(corpus)
	if err != nil {
		fail(err)
	}
	output, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(output))
	if *requireGate {
		if err := research.ValidateInterviewExitGate(report); err != nil {
			fail(err)
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
