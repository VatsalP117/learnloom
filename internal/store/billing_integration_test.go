package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
)

func TestGenerationEntitlementReservationIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	account, err := database.SyncAccountIdentity(
		ctx, "clerk-entitlement-"+uuid.NewString(), "entitlement@example.com",
		domain.AccountActive, now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := database.GetBillingEntitlement(ctx, account.ID, now)
	if err != nil || initial.PlanID != "none" || initial.CanGenerate ||
		initial.StreamAllowance == nil || *initial.StreamAllowance != 0 {
		t.Fatalf("paid-only initial entitlement=%#v err=%v", initial, err)
	}
	if err := activateIntegrationPlan(ctx, database, account.ID, "essential", now); err != nil {
		t.Fatal(err)
	}
	input := integrationNewsletterInput([]domain.SourceDefinition{{
		Name: "Entitlement source", URL: "https://example.com/entitlement", Limit: 5,
	}})
	created, err := database.CreateNewsletter(ctx, account.ID, input, 10, 20)
	if err != nil {
		t.Fatal(err)
	}
	entitlement, err := database.GetBillingEntitlement(ctx, account.ID, now)
	if err != nil || entitlement.PlanID != "essential" ||
		!entitlement.GenerationUnlimited || entitlement.GenerationUsed != 1 ||
		!entitlement.CanGenerate {
		t.Fatalf("initial entitlement=%#v err=%v", entitlement, err)
	}
	duplicate, err := database.EnqueueManualIssue(ctx, account.ID, created.Newsletter.ID, 20)
	if err != nil || duplicate.ID != created.FirstIssue.ID {
		t.Fatalf("idempotent issue=%#v err=%v", duplicate, err)
	}
	entitlement, err = database.GetBillingEntitlement(ctx, account.ID, now)
	if err != nil || entitlement.GenerationUsed != 1 {
		t.Fatalf("idempotent request consumed allowance: %#v err=%v", entitlement, err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE issues SET status = 'cancelled', completed_at = $2 WHERE id = $1
	`, created.FirstIssue.ID, now); err != nil {
		t.Fatal(err)
	}
	for expectedUsed := 2; expectedUsed <= 5; expectedUsed++ {
		issue, err := database.EnqueueManualIssue(ctx, account.ID, created.Newsletter.ID, 20)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.pool.Exec(ctx, `
			UPDATE issues SET status = 'cancelled', completed_at = $2 WHERE id = $1
		`, issue.ID, now); err != nil {
			t.Fatal(err)
		}
		entitlement, err = database.GetBillingEntitlement(ctx, account.ID, now)
		if err != nil || entitlement.GenerationUsed != expectedUsed {
			t.Fatalf("used=%d entitlement=%#v err=%v", expectedUsed, entitlement, err)
		}
	}
	entitlement, err = database.GetBillingEntitlement(ctx, account.ID, now)
	if err != nil || entitlement.GenerationUsed != 5 || !entitlement.CanGenerate ||
		!entitlement.GenerationUnlimited {
		t.Fatalf("unlimited generation entitlement=%#v err=%v", entitlement, err)
	}
	var issueCount int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM issues WHERE newsletter_id = $1
	`, created.Newsletter.ID).Scan(&issueCount); err != nil {
		t.Fatal(err)
	}
	if issueCount != 5 {
		t.Fatalf("unexpected generated issue count=%d", issueCount)
	}
}

func TestBillingLifecycleIsReplaySafeAndMonotonicIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	account, err := database.SyncAccountIdentity(
		ctx, "clerk-billing-lifecycle-"+uuid.NewString(), "billing@example.com",
		domain.AccountActive, now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	trialAt := now.Add(time.Minute)
	trialEnd := now.Add(14 * 24 * time.Hour)
	trial := BillingLifecycleUpdate{
		AccountID: account.ID, PlanID: "pro", ProviderEventID: "evt-trial-" + uuid.NewString(),
		EventType: "subscription.created", ProviderCustomerID: "ctm_test",
		ProviderSubscriptionID: "sub_test", SubscriptionStatus: "trialing",
		PeriodStart: now, PeriodEnd: trialEnd, TrialEndsAt: &trialEnd,
		EventOccurredAt: trialAt, PayloadSHA256: strings.Repeat("a", 64),
	}
	if err := database.ApplyBillingLifecycleUpdate(ctx, trial, trialAt); err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyBillingLifecycleUpdate(ctx, trial, trialAt.Add(time.Second)); err != nil {
		t.Fatalf("duplicate billing event was not idempotent: %v", err)
	}
	tampered := trial
	tampered.PayloadSHA256 = strings.Repeat("b", 64)
	if err := database.ApplyBillingLifecycleUpdate(ctx, tampered, trialAt.Add(2*time.Second)); err == nil {
		t.Fatal("billing replay with changed payload was accepted")
	}
	entitlement, err := database.GetBillingEntitlement(ctx, account.ID, trialAt)
	if err != nil || entitlement.PlanID != "pro" || entitlement.SubscriptionStatus != "trialing" ||
		!entitlement.GenerationUnlimited || !entitlement.CanGenerate {
		t.Fatalf("trial entitlement=%#v err=%v", entitlement, err)
	}
	pastDueAt := trialAt.Add(48 * time.Hour)
	pastDue := trial
	pastDue.ProviderEventID = "evt-past-due-" + uuid.NewString()
	pastDue.EventType = "transaction.payment_failed"
	pastDue.SubscriptionStatus = "past_due"
	pastDue.EventOccurredAt = pastDueAt
	if err := database.ApplyBillingLifecycleUpdate(ctx, pastDue, pastDueAt); err != nil {
		t.Fatal(err)
	}
	entitlement, err = database.GetBillingEntitlement(ctx, account.ID, pastDueAt)
	if err != nil || entitlement.EntitlementStatus != "grace" ||
		entitlement.GraceEndsAt == nil || !entitlement.CanGenerate {
		t.Fatalf("past-due entitlement=%#v err=%v", entitlement, err)
	}
	stale := trial
	stale.ProviderEventID = "evt-stale-" + uuid.NewString()
	stale.SubscriptionStatus = "active"
	stale.EventOccurredAt = trialAt.Add(-time.Minute)
	if err := database.ApplyBillingLifecycleUpdate(ctx, stale, pastDueAt); err != nil {
		t.Fatal(err)
	}
	entitlement, err = database.GetBillingEntitlement(ctx, account.ID, pastDueAt)
	if err != nil || entitlement.SubscriptionStatus != "past_due" {
		t.Fatalf("stale event overwrote billing state=%#v err=%v", entitlement, err)
	}
	refunded := trial
	refunded.ProviderEventID = "evt-refund-" + uuid.NewString()
	refunded.EventType = "transaction.refunded"
	refunded.SubscriptionStatus = "refunded"
	refunded.EventOccurredAt = pastDueAt.Add(time.Hour)
	if err := database.ApplyBillingLifecycleUpdate(ctx, refunded, refunded.EventOccurredAt); err != nil {
		t.Fatal(err)
	}
	entitlement, err = database.GetBillingEntitlement(ctx, account.ID, refunded.EventOccurredAt)
	if err != nil || entitlement.PlanID != "none" || entitlement.SubscriptionStatus != "refunded" || entitlement.CanGenerate {
		t.Fatalf("refunded entitlement=%#v err=%v", entitlement, err)
	}
	var eventCount int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM billing_webhook_events
		WHERE provider = 'paddle' AND event_id = $1
	`, trial.ProviderEventID).Scan(&eventCount); err != nil || eventCount != 1 {
		t.Fatalf("billing replay count=%d err=%v", eventCount, err)
	}
	ignored := BillingWebhookReceipt{
		ProviderEventID: "evt-ignored-" + uuid.NewString(),
		EventType:       "customer.updated", EventOccurredAt: refunded.EventOccurredAt,
		PayloadSHA256: strings.Repeat("c", 64),
	}
	if err := database.RecordIgnoredBillingWebhook(ctx, ignored, refunded.EventOccurredAt); err != nil {
		t.Fatal(err)
	}
	if err := database.RecordIgnoredBillingWebhook(ctx, ignored, refunded.EventOccurredAt); err != nil {
		t.Fatalf("ignored event replay failed: %v", err)
	}
	var processedAt *time.Time
	if err := database.pool.QueryRow(ctx, `
		SELECT processed_at FROM billing_webhook_events
		WHERE provider = 'paddle' AND event_id = $1
	`, ignored.ProviderEventID).Scan(&processedAt); err != nil || processedAt == nil {
		t.Fatalf("ignored webhook was not audited: processed_at=%v err=%v", processedAt, err)
	}
}

func TestBillingEntitlementPeriodRollsOnReadIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Second)
	account, err := database.SyncAccountIdentity(
		ctx, "clerk-billing-rollover-"+uuid.NewString(), "rollover@example.com",
		domain.AccountActive, now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.GetBillingEntitlement(ctx, account.ID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := database.pool.Exec(ctx, `
		UPDATE account_billing SET current_period_start = $2, current_period_end = $3
		WHERE account_id = $1
	`, account.ID, now.Add(-60*24*time.Hour), now.Add(-30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	entitlement, err := database.GetBillingEntitlement(ctx, account.ID, now)
	if err != nil || entitlement.PeriodStart.After(now) || !entitlement.PeriodEnd.After(now) ||
		entitlement.GenerationUsed != 0 {
		t.Fatalf("rolled entitlement=%#v err=%v", entitlement, err)
	}
}

func TestApprovedRefundIsFinancialFactNotSubscriptionCancellationIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	account, err := database.SyncAccountIdentity(
		ctx, "clerk-refund-"+uuid.NewString(), "refund@example.com",
		domain.AccountActive, now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	activation := BillingLifecycleUpdate{
		AccountID: account.ID, PlanID: "essential", ProviderEventID: "evt-paid-" + uuid.NewString(),
		EventType: "transaction.completed", ProviderCustomerID: "ctm_refund_test",
		ProviderSubscriptionID: "sub_refund_test", SubscriptionStatus: "active",
		PeriodStart: now, PeriodEnd: now.Add(30 * 24 * time.Hour),
		EventOccurredAt: now, PayloadSHA256: strings.Repeat("d", 64),
		CurrencyCode: "USD", AmountMinor: int64Pointer(1500),
	}
	if err := database.ApplyBillingLifecycleUpdate(ctx, activation, now); err != nil {
		t.Fatal(err)
	}
	refund := BillingRefundAdjustment{
		ProviderEventID: "evt-adjustment-" + uuid.NewString(),
		EventType:       "adjustment.updated", ProviderCustomerID: "ctm_refund_test",
		ProviderSubscriptionID: "sub_refund_test", Reason: "partial service credit",
		CurrencyCode: "usd", AmountMinor: 500, EventOccurredAt: now.Add(time.Minute),
		PayloadSHA256: strings.Repeat("e", 64),
	}
	if err := database.ApplyBillingRefundAdjustment(ctx, refund, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyBillingRefundAdjustment(ctx, refund, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("refund replay failed: %v", err)
	}
	entitlement, err := database.GetBillingEntitlement(ctx, account.ID, now.Add(time.Minute))
	if err != nil || entitlement.SubscriptionStatus != "active" || !entitlement.CanGenerate {
		t.Fatalf("refund changed subscription state=%#v err=%v", entitlement, err)
	}
	var revenueCount int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM billing_revenue_events
		WHERE provider_event_id = $1 AND event_type = 'refund'
	`, refund.ProviderEventID).Scan(&revenueCount); err != nil || revenueCount != 1 {
		t.Fatalf("refund revenue count=%d err=%v", revenueCount, err)
	}
}

func TestBillingFeedbackRequiresMatchingPlanStateIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	account, err := database.SyncAccountIdentity(
		ctx, "clerk-feedback-"+uuid.NewString(), "feedback@example.com",
		domain.AccountActive, now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.GetBillingEntitlement(ctx, account.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := database.RecordBillingFeedback(ctx, account.ID, "non_conversion",
		"too_expensive", "Still validating the habit.", now); err != nil {
		t.Fatal(err)
	}
	if err := database.RecordBillingFeedback(ctx, account.ID, "non_conversion",
		"insufficient_value", "The first cycle was not strong enough.", now.Add(time.Minute)); err != nil {
		t.Fatalf("feedback update failed: %v", err)
	}
	if err := database.RecordBillingFeedback(ctx, account.ID, "cancellation",
		"no_longer_needed", "", now); !errors.Is(err, ErrConflict) {
		t.Fatalf("active free account cancellation feedback err=%v", err)
	}
	if err := database.RecordBillingFeedback(ctx, account.ID, "non_conversion",
		"invented_reason", "", now); err == nil {
		t.Fatal("unknown feedback reason was accepted")
	}
	var reason string
	if err := database.pool.QueryRow(ctx, `
		SELECT reason_code FROM billing_feedback
		WHERE account_id = $1 AND context = 'non_conversion'
	`, account.ID).Scan(&reason); err != nil || reason != "insufficient_value" {
		t.Fatalf("feedback reason=%q err=%v", reason, err)
	}
}

func TestPendingBillingCheckoutIsReusedAndCompletedIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	account, err := database.SyncAccountIdentity(
		ctx, "clerk-checkout-"+uuid.NewString(), "checkout@example.com",
		domain.AccountActive, now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	firstID := "txn_first_" + uuid.NewString()
	selected, err := database.RecordPendingBillingCheckout(ctx, account.ID, firstID, "pro", now)
	if err != nil || selected != firstID {
		t.Fatalf("first checkout=%q err=%v", selected, err)
	}
	secondID := "txn_second_" + uuid.NewString()
	selected, err = database.RecordPendingBillingCheckout(ctx, account.ID, secondID, "pro", now.Add(time.Second))
	if err != nil || selected != firstID {
		t.Fatalf("concurrent checkout selected=%q err=%v", selected, err)
	}
	pending, err := database.GetPendingBillingCheckout(ctx, account.ID, "pro", now.Add(time.Minute))
	if err != nil || pending != firstID {
		t.Fatalf("pending checkout=%q err=%v", pending, err)
	}
	activation := BillingLifecycleUpdate{
		AccountID: account.ID, PlanID: "pro", ProviderEventID: "evt-checkout-paid-" + uuid.NewString(),
		EventType: "transaction.completed", ProviderTransactionID: firstID,
		ProviderCustomerID: "ctm_checkout", ProviderSubscriptionID: "sub_checkout",
		SubscriptionStatus: "active", PeriodStart: now,
		PeriodEnd: now.Add(30 * 24 * time.Hour), EventOccurredAt: now.Add(time.Minute),
		PayloadSHA256: strings.Repeat("f", 64),
		CurrencyCode:  "USD", AmountMinor: int64Pointer(1500),
	}
	if err := database.ApplyBillingLifecycleUpdate(ctx, activation, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	pending, err = database.GetPendingBillingCheckout(ctx, account.ID, "pro", now.Add(2*time.Minute))
	if err != nil || pending != "" {
		t.Fatalf("completed checkout remained pending=%q err=%v", pending, err)
	}
}

func int64Pointer(value int64) *int64 { return &value }

func TestBillingLifecycleRejectsUnknownAccountAndMissingRevenueIntegration(t *testing.T) {
	database := openIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC()
	unknown := BillingLifecycleUpdate{
		AccountID: uuid.NewString(), PlanID: "pro", ProviderEventID: "evt-unknown-" + uuid.NewString(),
		EventType: "subscription.created", ProviderCustomerID: "ctm_unknown",
		ProviderSubscriptionID: "sub_unknown", SubscriptionStatus: "trialing",
		PeriodStart: now, PeriodEnd: now.Add(14 * 24 * time.Hour),
		EventOccurredAt: now, PayloadSHA256: strings.Repeat("1", 64),
	}
	if err := database.ApplyBillingLifecycleUpdate(ctx, unknown, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown account lifecycle err=%v", err)
	}
	var receiptCount int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM billing_webhook_events WHERE event_id = $1
	`, unknown.ProviderEventID).Scan(&receiptCount); err != nil || receiptCount != 0 {
		t.Fatalf("failed unknown webhook persisted=%d err=%v", receiptCount, err)
	}
	account, err := database.SyncAccountIdentity(
		ctx, "clerk-missing-revenue-"+uuid.NewString(), "missing-revenue@example.com",
		domain.AccountActive, now.UnixMilli(),
	)
	if err != nil {
		t.Fatal(err)
	}
	missing := unknown
	missing.AccountID = account.ID
	missing.ProviderEventID = "evt-missing-revenue-" + uuid.NewString()
	missing.EventType = "transaction.completed"
	missing.SubscriptionStatus = "active"
	missing.PayloadSHA256 = strings.Repeat("2", 64)
	if err := database.ApplyBillingLifecycleUpdate(ctx, missing, now); err == nil {
		t.Fatal("completed transaction without revenue was accepted")
	}
	entitlement, err := database.GetBillingEntitlement(ctx, account.ID, now)
	if err != nil || entitlement.PlanID != "none" {
		t.Fatalf("missing revenue activated entitlement=%#v err=%v", entitlement, err)
	}
	activated := missing
	activated.ProviderEventID = "evt-subscription-activated-" + uuid.NewString()
	activated.EventType = "subscription.activated"
	activated.ProviderCustomerID = "ctm_subscription_activated"
	activated.ProviderSubscriptionID = "sub_subscription_activated"
	activated.EventOccurredAt = now.Add(time.Minute)
	activated.PayloadSHA256 = strings.Repeat("3", 64)
	if err := database.ApplyBillingLifecycleUpdate(ctx, activated, now.Add(time.Minute)); err != nil {
		t.Fatalf("subscription activation without transaction amount failed: %v", err)
	}
	entitlement, err = database.GetBillingEntitlement(ctx, account.ID, now.Add(time.Minute))
	if err != nil || entitlement.PlanID != "pro" || entitlement.SubscriptionStatus != "active" {
		t.Fatalf("subscription activation entitlement=%#v err=%v", entitlement, err)
	}
	var revenueCount int
	if err := database.pool.QueryRow(ctx, `
		SELECT count(*) FROM billing_revenue_events WHERE account_id = $1
	`, account.ID).Scan(&revenueCount); err != nil || revenueCount != 0 {
		t.Fatalf("subscription activation invented revenue=%d err=%v", revenueCount, err)
	}
}

func activateIntegrationPlan(
	ctx context.Context,
	database *Store,
	accountID, planID string,
	now time.Time,
) error {
	if _, err := database.GetBillingEntitlement(ctx, accountID, now); err != nil {
		return err
	}
	_, err := database.pool.Exec(ctx, `
		UPDATE account_billing SET
		  provider = 'paddle', plan_id = $2, subscription_status = 'active',
		  entitlement_status = 'active', current_period_start = $3,
		  current_period_end = $3::timestamptz + interval '30 days', updated_at = $3
		WHERE account_id = $1
	`, accountID, planID, now)
	return err
}
