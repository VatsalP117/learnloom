package store

import (
	"strings"
	"testing"
)

func TestMigrationVersion(t *testing.T) {
	t.Parallel()
	version, err := migrationVersion("001_initial.sql")
	if err != nil || version != 1 {
		t.Fatalf("unexpected version=%d err=%v", version, err)
	}
	for _, name := range []string{"initial.sql", "000_invalid.sql", "x_bad.sql"} {
		if _, err := migrationVersion(name); err == nil {
			t.Errorf("%q should be rejected", name)
		}
	}
}

func TestEmbeddedMigrationLedgerIsContiguous(t *testing.T) {
	t.Parallel()
	version, err := expectedSchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if version != 41 {
		t.Fatalf("embedded migration version = %d, want 41", version)
	}
}

func TestPaidOnlyStreamPackagingMigration(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/041_stream_allowance.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"('none', 'No active plan', 0, 30, 0, true",
		"('essential', 'Essential', NULL, 30, 3, true",
		"generation_allowance = NULL",
		"stream_allowance = NULL",
		"active = false",
		"WHERE plan_id = 'free'",
		"ADD COLUMN plan_id text REFERENCES billing_plans(id)",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("paid-only stream migration missing %q", expected)
		}
	}
}

func TestSourceRetrievalPolicyMigrationIsAppendOnlyAndAudited(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/040_source_retrieval_policy.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE source_retrieval_policy_events",
		"'exact_url', 'registrable_domain'",
		"'block', 'unblock'",
		"actor_account_id uuid REFERENCES accounts(id) ON DELETE SET NULL",
		"CREATE VIEW current_source_retrieval_policy",
		"Append-only operator policy",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("source retrieval policy migration missing %q", expected)
		}
	}
}

func TestOperatorModerationAuditSurvivesAccountErasure(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/039_operator_moderation_audit.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"ALTER COLUMN actor_account_id DROP NOT NULL",
		"ON DELETE SET NULL",
		"actions survive later account erasure",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("operator moderation migration missing %q", expected)
		}
	}
}

func TestEvidenceClassificationMigrationIsAppendOnlyAndFailClosed(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/038_account_evidence_classifications.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE account_evidence_classifications",
		"'real_user', 'founder', 'test'",
		"external_design_partner",
		"classified_by text NOT NULL",
		"evidence_reference text NOT NULL",
		"account_evidence_classifications_current",
		"CREATE VIEW current_account_evidence_classifications",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("evidence classification migration missing %q", expected)
		}
	}
	if strings.Contains(body, "primary_email") {
		t.Fatal("evidence classification must not retain account email")
	}
}

func TestArtifactCleanupMigrationIsDurableAndClaimSafe(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/037_artifact_cleanup_queue.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE artifact_cleanup_queue",
		"artifact_key text PRIMARY KEY",
		"claim_token uuid",
		"claim_expires_at timestamptz",
		"attempt_count integer NOT NULL DEFAULT 0",
		"CREATE INDEX artifact_cleanup_queue_available",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("artifact cleanup migration missing %q", expected)
		}
	}
}

func TestOperationalCOGSMigrationIsBoundedAndIdempotent(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/036_operational_cogs.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE operational_cogs_events",
		"external_reference text PRIMARY KEY",
		"'search', 'email', 'storage', 'support', 'infrastructure', 'other'",
		"cost_microusd bigint NOT NULL",
		"account_id uuid REFERENCES accounts(id) ON DELETE SET NULL",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("operational COGS migration missing %q", expected)
		}
	}
	for _, forbidden := range []string{"description text", "metadata jsonb", "email_address"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("operational COGS migration stores forbidden detail %q", forbidden)
		}
	}
}

func TestPublicActivationAttributionMigrationPreservesSignupBoundary(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/035_public_activation_attribution.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"ADD COLUMN activated_at timestamptz",
		"activated_at >= converted_at",
		"public_attribution_activated_owner_period",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("public activation attribution migration missing %q", expected)
		}
	}
}

func TestBillingCheckoutMigrationCollapsesPendingSessions(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/034_billing_checkout_sessions.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE billing_checkout_sessions",
		"status IN ('pending', 'completed', 'expired')",
		"billing_checkout_one_pending_per_account",
		"WHERE status = 'pending'",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("billing checkout migration missing %q", expected)
		}
	}
}

func TestDesignPartnerMigrationEnforcesConsentAndValueFirstPayment(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/033_design_partner_beta.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE design_partner_participants",
		"research_consent_at timestamptz",
		"payment_asked_at IS NULL OR value_cycle_completed_at IS NOT NULL",
		"CREATE TABLE design_partner_sessions",
		"'setup_observation'", "'weekly_outcome_interview'",
		"CREATE TABLE design_partner_quality_samples",
		"unsupported_claim_count integer NOT NULL",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("design partner migration missing %q", expected)
		}
	}
	for _, forbidden := range []string{"email text", "transcript text", "verbatim_notes text"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("design partner migration stores forbidden research data %q", forbidden)
		}
	}
}

func TestBillingFeedbackMigrationUsesStableBoundedTaxonomy(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/032_billing_feedback.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE billing_feedback",
		"context IN ('non_conversion', 'cancellation')",
		"'too_expensive'", "'insufficient_value'", "'quality_concerns'",
		"char_length(note) BETWEEN 1 AND 1000",
		"UNIQUE (account_id, context)",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("billing feedback migration missing %q", expected)
		}
	}
}

func TestBillingEconomicsMigrationRecordsRevenueAndReasons(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/031_billing_economics.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"ADD COLUMN reason text",
		"ADD COLUMN currency_code text",
		"ADD COLUMN amount_minor bigint",
		"CREATE TABLE billing_revenue_events",
		"provider_fee_minor bigint",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("billing economics migration missing %q", expected)
		}
	}
}

func TestBillingEntitlementsMigrationIsAtomicAndLifecycleReady(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/030_billing_entitlements.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE billing_plans",
		"CREATE TABLE account_billing",
		"CREATE TABLE generation_usage_reservations",
		"DEFERRABLE INITIALLY DEFERRED",
		"entitlement_status IN ('active', 'grace', 'generation_paused')",
		"CREATE TABLE billing_lifecycle_events",
		"CREATE TABLE billing_webhook_events",
		"'entitlement_deferred'",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("billing entitlement migration missing %q", expected)
		}
	}
}

func TestPublicPathFollowersMigrationRequiresConfirmedOptInLifecycle(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/029_public_path_followers.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE public_path_followers",
		"status IN ('pending', 'confirmed', 'unsubscribed')",
		"confirmation_token_hash text NOT NULL",
		"unsubscribe_token_hash text NOT NULL",
		"CREATE TABLE public_follow_unsubscribe_tokens",
		"CREATE TABLE public_follow_deliveries",
		"UNIQUE (newsletter_id, email_hash)",
		"'follow'",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("public path followers migration missing %q", expected)
		}
	}
}

func TestPublicGrowthAnalyticsMigrationIsPseudonymousAndOwnerScoped(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/028_public_growth_analytics.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE public_growth_events",
		"visitor_fingerprint text NOT NULL",
		"UNIQUE (issue_id, event_name, channel, visitor_fingerprint, visitor_day)",
		"CREATE TABLE public_attribution_conversions",
		"converted_account_id uuid NOT NULL UNIQUE",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("public growth analytics migration missing %q", expected)
		}
	}
	for _, forbidden := range []string{"ip_address", "user_agent text", "visitor_email"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("public growth analytics migration stores forbidden identifier %q", forbidden)
		}
	}
}

func TestPublicationStateMigrationHasSafeCompatibilityPlan(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/027_publication_states.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"SET publication_state = 'private'",
		"ALTER COLUMN publication_state SET DEFAULT 'draft'",
		"publication_state IN ('private', 'draft', 'published')",
		"first_publish_reviewed_at timestamptz",
		"lesson_publication_default text NOT NULL DEFAULT 'draft'",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("publication state migration missing %q", expected)
		}
	}
}

func TestReentryBacklogMigrationPreservesDismissedLessons(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/026_reentry_backlog.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE lesson_backlog_dismissals",
		"issue_id uuid NOT NULL REFERENCES issues(id) ON DELETE CASCADE",
		"reason text NOT NULL CHECK (reason IN ('reentry_reset'))",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("re-entry backlog migration missing %q", expected)
		}
	}
}

func TestLearningRhythmMigrationStoresDesiredAndEffectiveCadence(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/025_learning_rhythm.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"rhythm_mode text NOT NULL",
		"effective_rhythm_mode text NOT NULL",
		"selected_weekdays smallint[] NOT NULL",
		"auto_throttle_enabled boolean NOT NULL",
		"requested_lesson_type text",
		"CREATE TABLE rhythm_decisions",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("learning rhythm migration missing %q", expected)
		}
	}
}

func TestTodayFocusMigrationStoresExplainableSelection(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/024_today_focus.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE today_focus_selections",
		"reason_text text NOT NULL",
		"score_components jsonb NOT NULL",
		"CHECK (kind IN ('lesson', 'review', 'reentry', 'clear'))",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("Today focus migration missing %q", expected)
		}
	}
}

func TestOnboardingDraftMigrationAddsDurableFunnelState(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/022_onboarding_drafts_events.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE onboarding_drafts",
		"account_id uuid NOT NULL UNIQUE",
		"revision bigint NOT NULL DEFAULT 1",
		"CREATE TABLE onboarding_draft_completions",
		"octet_length(payload::text) <= 32768",
		"'source_preview_reached'",
		"'activation_completed'",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("onboarding migration missing %q", expected)
		}
	}
}

func TestLessonRetrievalMigrationSeparatesDraftRevealAndSkip(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/023_lesson_retrieval_responses.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"CREATE TABLE lesson_retrieval_responses",
		"revealed_at timestamptz",
		"char_length(response_text) <= 2000",
		"skipped AND response_text = '' AND revealed_at IS NOT NULL",
		"PRIMARY KEY (account_id, issue_id, prompt_key)",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("Lesson Retrieval migration missing %q", expected)
		}
	}
}

func TestSourcePortfolioApprovalMigrationStopsBeforeGeneration(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/021_source_portfolio_approval.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"source_review_mode text NOT NULL DEFAULT 'auto'",
		"source_approved_at timestamptz",
		"'awaiting_approval', 'generated'",
		"issues_awaiting_source_approval",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("source approval migration missing %q: %s", expected, body)
		}
	}
}

func TestSourcePreferencesMigrationAddsReversibleOwnerControls(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/020_source_preferences.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"preference text NOT NULL DEFAULT 'neutral'",
		"'preferred', 'blocked'",
		"source_specs_newsletter_preference",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("source preference migration missing %q: %s", expected, body)
		}
	}
}

func TestSourceRankingMigrationAddsAuditableComponents(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/019_source_ranking_audit.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"source_role text",
		"ranking_version text",
		"score_components jsonb",
		"source_specs_newsletter_role",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("source ranking migration missing %q: %s", expected, body)
		}
	}
}

func TestIssueDeferralMigrationAddsTerminalEvidenceState(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/018_issue_deferrals.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"'deferred', 'cancelled'",
		"'deferred', 'abandoned'",
		"issues_deferred_timeline",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("Issue deferral migration missing %q: %s", expected, body)
		}
	}
}

func TestSearchIndexingMigrationDefaultsOffAndRequiresPublicVisibility(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/005_site_search_indexing.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"search_indexing boolean NOT NULL DEFAULT false",
		"NOT search_indexing OR visibility = 'public'",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("migration missing %q: %s", expected, body)
		}
	}
}
