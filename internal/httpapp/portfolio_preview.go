package httpapp

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/VatsalP117/learnloom/internal/source"
)

type PortfolioPlanner func(context.Context, string) (source.PortfolioPreview, error)

type researchPlanPreview struct {
	InitialConcepts  []string `json:"initialConcepts"`
	LikelyFirstTitle string   `json:"likelyFirstLesson"`
	Objective        string   `json:"objective"`
	MinimumMinutes   int      `json:"minimumPreparationMinutes"`
	MaximumMinutes   int      `json:"maximumPreparationMinutes"`
}

func (s *Server) SetPortfolioPlanner(planner PortfolioPlanner) {
	s.portfolioPlanner = planner
}

func (s *Server) previewSourcePortfolio(
	response http.ResponseWriter,
	request *http.Request,
	current session,
) {
	if s.portfolioPlanner == nil {
		writeProblem(
			response,
			http.StatusServiceUnavailable,
			"source_preview_unavailable",
			"Source discovery is not available right now. You can still continue with sources you choose.",
		)
		return
	}
	if !s.allowAction(response, request, "source-portfolio-preview", time.Hour, 12) {
		return
	}
	var body struct {
		Topic             string `json:"topic"`
		LearnerGoal       string `json:"learnerGoal"`
		LearnerLevel      string `json:"learnerLevel"`
		OnboardingDraftID string `json:"onboardingDraftId"`
	}
	if !decodeJSON(response, request, s.cfg.MaxRequestBodyBytes, &body) {
		return
	}
	body.Topic = strings.TrimSpace(body.Topic)
	body.LearnerGoal = strings.TrimSpace(body.LearnerGoal)
	body.LearnerLevel = strings.TrimSpace(body.LearnerLevel)
	if body.Topic == "" || len([]rune(body.Topic)) > 400 {
		writeProblem(
			response,
			http.StatusBadRequest,
			"invalid_topic",
			"Enter a topic or question of 400 characters or fewer.",
		)
		return
	}
	if len([]rune(body.LearnerGoal)) > 500 {
		writeProblem(response, http.StatusBadRequest, "invalid_learner_goal", "The desired outcome must be 500 characters or fewer.")
		return
	}
	if body.LearnerLevel != "" && body.LearnerLevel != "beginner" &&
		body.LearnerLevel != "intermediate" && body.LearnerLevel != "advanced" {
		writeProblem(response, http.StatusBadRequest, "invalid_learner_level", "Learner level must be beginner, intermediate, or advanced.")
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), 15*time.Second)
	defer cancel()
	preview, err := s.portfolioPlanner(ctx, body.Topic)
	if err != nil {
		s.logger.WarnContext(ctx, "source portfolio preview failed", "error", err)
		writeProblem(
			response,
			http.StatusServiceUnavailable,
			"source_preview_unavailable",
			"Learnloom couldn’t preview the source portfolio right now. You can retry or continue with sources you choose.",
		)
		return
	}
	if body.OnboardingDraftID != "" {
		if err := s.store.RecordOnboardingPreviewReached(
			ctx, current.Account.ID, body.OnboardingDraftID, time.Now().UTC(),
		); err != nil {
			writeStoreError(response, err)
			return
		}
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"rankingVersion": preview.RankingVersion,
		"items":          preview.Items,
		"missingRoles":   preview.MissingRoles,
		"warnings":       preview.Warnings,
		"researchPlan": buildResearchPlanPreview(
			body.Topic,
			body.LearnerGoal,
			body.LearnerLevel,
		),
	})
}

func buildResearchPlanPreview(topic, learnerGoal, learnerLevel string) researchPlanPreview {
	topic = truncatePreviewText(strings.TrimSpace(topic), 90)
	goal := truncatePreviewText(strings.TrimSpace(learnerGoal), 90)
	objective := "Explain the core mechanisms, test the evidence, and recognize important limitations."
	switch learnerLevel {
	case "beginner":
		objective = "Build a clear vocabulary, explain the basic mechanism, and avoid the most common misconceptions."
	case "advanced":
		objective = "Stress-test the dominant model, compare competing evidence, and identify consequential edge cases."
	}
	application := "Practical applications and failure modes"
	if goal != "" {
		application = "Apply the model to: " + goal
	}
	return researchPlanPreview{
		InitialConcepts: []string{
			"Foundations and boundaries of " + topic,
			"Core mechanisms and causal relationships",
			"Evidence quality and counterarguments",
			application,
		},
		LikelyFirstTitle: "Build a working model of " + topic,
		Objective:        objective,
		MinimumMinutes:   5,
		MaximumMinutes:   15,
	}
}

func truncatePreviewText(value string, maximum int) string {
	if utf8.RuneCountInString(value) <= maximum {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:maximum-1])) + "…"
}
