package dossier

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/failure"
)

func (g *Generator) runStage(
	ctx context.Context,
	stage, input string,
	onStage StageObserver,
) (result string, resultErr error) {
	started := time.Now()
	var usage ModelUsage
	defer func() {
		if onStage != nil {
			onStage(stage, time.Since(started), usage, resultErr)
		}
	}()
	completion, err := g.model.Complete(ctx, CompletionRequest{
		Stage:       stage,
		Instruction: stageInstructions()[stage],
		Input:       input,
	})
	if err != nil {
		return "", stageExecutionFailure(stage, err)
	}
	usage = completion.Usage
	output := completion.Output
	if strings.TrimSpace(output) == "" {
		return "", fmt.Errorf("%s stage returned empty output", stage)
	}
	return strings.TrimSpace(output), nil
}

func runStructured[T any](
	ctx context.Context,
	model Completer,
	stage, instruction, input string,
	onStage StageObserver,
	validate func(T) error,
) (result T, resultErr error) {
	started := time.Now()
	var usage ModelUsage
	defer func() {
		if onStage != nil {
			onStage(stage, time.Since(started), usage, resultErr)
		}
	}()
	var zero T
	var repairReason string
	for attempt := 0; attempt < 2; attempt++ {
		stageInput := input
		if repairReason != "" {
			stageInput += "\n\n# Contract repair\n\nYour previous response was rejected: " +
				repairReason +
				"\nReturn a corrected response in the exact requested format."
		}
		completion, err := model.Complete(ctx, CompletionRequest{
			Stage: stage, Instruction: instruction, Input: stageInput, Structured: true,
		})
		if err != nil {
			return zero, stageExecutionFailure(stage, err)
		}
		usage = combineModelUsage(usage, completion.Usage)
		output := completion.Output
		var value T
		if err := decodeStructured(output, &value); err == nil {
			if err := validate(value); err == nil {
				return value, nil
			} else {
				repairReason = safeRepairReason(err)
				continue
			}
		} else {
			repairReason = safeRepairReason(err)
		}
	}
	return zero, failure.New(
		"model_contract_unsatisfied",
		failure.CategoryContentQuality,
		stage,
		true,
		failure.PublicInternal,
		fmt.Errorf("could not satisfy its output contract: %s", repairReason),
	)
}

func combineModelUsage(left, right ModelUsage) ModelUsage {
	return ModelUsage{
		InputTokens:           left.InputTokens + right.InputTokens,
		OutputTokens:          left.OutputTokens + right.OutputTokens,
		Retries:               left.Retries + right.Retries,
		EstimatedCostMicroUSD: left.EstimatedCostMicroUSD + right.EstimatedCostMicroUSD,
	}
}

func stageExecutionFailure(stage string, err error) error {
	category := failure.CategoryProvider
	code := "model_provider_failure"
	if errors.Is(err, ErrOutputTruncated) {
		category = failure.CategoryContentQuality
		code = "model_output_truncated"
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		category = failure.CategoryInfrastructure
		code = "generation_interrupted"
	}
	return failure.New(code, category, stage, true, failure.PublicInternal, err)
}
