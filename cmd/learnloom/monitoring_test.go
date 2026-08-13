package main

import (
	"os"
	"strings"
	"testing"
)

func TestLaunchCriticalGenerationAlertsRemainProvisioned(t *testing.T) {
	raw, err := os.ReadFile("../../infra/monitoring/learnloom-rules.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rules := string(raw)
	for _, required := range []string{
		"LearnloomGenerationFailureRateRegression",
		"learnloom:generation_failure_ratio_30m",
		"LearnloomConsecutiveAccountFailures",
		"learnloom_accounts_with_consecutive_generation_failures",
		"LearnloomIssueQueueStale",
		"LearnloomProviderOutputTruncation",
		"learnloom_model_output_truncations_total",
		"LearnloomDailyModelBudgetReached",
		"LearnloomBillingWebhookStuck",
		"learnloom_billing_webhook_oldest_seconds",
		"LearnloomBillingGenerationPaused",
		"LearnloomArtifactCleanupStale",
		"learnloom_artifact_cleanup_oldest_seconds",
	} {
		if !strings.Contains(rules, required) {
			t.Fatalf("monitoring rules are missing %q", required)
		}
	}
}

func TestOperationsDashboardIncludesCommercialAndCleanupBacklogs(t *testing.T) {
	raw, err := os.ReadFile("../../infra/monitoring/learnloom-dashboard.json")
	if err != nil {
		t.Fatal(err)
	}
	dashboard := string(raw)
	for _, required := range []string{
		"Artifact cleanup health",
		"learnloom_artifact_cleanups_pending",
		"learnloom_artifact_cleanup_oldest_seconds",
		"Billing lifecycle",
		"learnloom_billing_webhooks_pending",
		"learnloom_billing_accounts",
		"learnloom_billing_checkouts_pending",
	} {
		if !strings.Contains(dashboard, required) {
			t.Fatalf("operations dashboard is missing %q", required)
		}
	}
}
