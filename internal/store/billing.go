package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type BillingLifecycleUpdate struct {
	AccountID              string
	ProviderEventID        string
	EventType              string
	ProviderCustomerID     string
	ProviderSubscriptionID string
	ProviderTransactionID  string
	SubscriptionStatus     string
	PeriodStart            time.Time
	PeriodEnd              time.Time
	TrialEndsAt            *time.Time
	EventOccurredAt        time.Time
	PayloadSHA256          string
	Reason                 string
	CurrencyCode           string
	AmountMinor            *int64
	ProviderFeeMinor       *int64
}

func (s *Store) GetPendingBillingCheckout(
	ctx context.Context,
	accountID string,
	now time.Time,
) (string, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer rollback(tx)
	if _, err := tx.Exec(ctx, `
		UPDATE billing_checkout_sessions SET status = 'expired', updated_at = $2
		WHERE account_id = $1 AND status = 'pending'
		  AND created_at < $2::timestamptz - interval '30 minutes'
	`, accountID, now); err != nil {
		return "", err
	}
	var transactionID string
	err = tx.QueryRow(ctx, `
		SELECT transaction_id FROM billing_checkout_sessions
		WHERE account_id = $1 AND status = 'pending'
	`, accountID).Scan(&transactionID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return "", err
		}
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return transactionID, nil
}

func (s *Store) RecordPendingBillingCheckout(
	ctx context.Context,
	accountID, transactionID string,
	now time.Time,
) (string, error) {
	if accountID == "" || !strings.HasPrefix(transactionID, "txn_") {
		return "", errors.New("billing checkout is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer rollback(tx)
	if _, err := tx.Exec(ctx, `
		UPDATE billing_checkout_sessions SET status = 'expired', updated_at = $2
		WHERE account_id = $1 AND status = 'pending'
		  AND created_at < $2::timestamptz - interval '30 minutes'
	`, accountID, now); err != nil {
		return "", err
	}
	var selectedID string
	err = tx.QueryRow(ctx, `
		INSERT INTO billing_checkout_sessions (
		  transaction_id, account_id, status, created_at, updated_at
		) VALUES ($1, $2, 'pending', $3, $3)
		ON CONFLICT (account_id) WHERE status = 'pending' DO UPDATE SET
		  updated_at = billing_checkout_sessions.updated_at
		RETURNING transaction_id
	`, transactionID, accountID, now).Scan(&selectedID)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return selectedID, nil
}

type BillingWebhookReceipt struct {
	ProviderEventID string
	EventType       string
	EventOccurredAt time.Time
	PayloadSHA256   string
}

type BillingRefundAdjustment struct {
	ProviderEventID        string
	EventType              string
	ProviderCustomerID     string
	ProviderSubscriptionID string
	Reason                 string
	CurrencyCode           string
	AmountMinor            int64
	EventOccurredAt        time.Time
	PayloadSHA256          string
}

// ApplyBillingRefundAdjustment records an approved partial or full refund as
// a financial fact without guessing that the related subscription was
// canceled. Paddle reports cancellation separately through subscription events.
func (s *Store) ApplyBillingRefundAdjustment(
	ctx context.Context,
	adjustment BillingRefundAdjustment,
	now time.Time,
) error {
	adjustment.CurrencyCode = strings.ToUpper(strings.TrimSpace(adjustment.CurrencyCode))
	if adjustment.ProviderEventID == "" || adjustment.EventType == "" ||
		adjustment.ProviderCustomerID == "" || adjustment.EventOccurredAt.IsZero() ||
		len(adjustment.PayloadSHA256) != 64 || len(adjustment.CurrencyCode) != 3 ||
		adjustment.AmountMinor < 0 || len(adjustment.Reason) > 500 {
		return errors.New("billing refund adjustment is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	inserted, err := recordBillingWebhookTx(ctx, tx, adjustment.ProviderEventID,
		adjustment.EventType, adjustment.EventOccurredAt, adjustment.PayloadSHA256, now)
	if err != nil {
		return err
	}
	if !inserted {
		return tx.Commit(ctx)
	}
	var accountID string
	err = tx.QueryRow(ctx, `
		SELECT account_id::text FROM account_billing
		WHERE provider = 'paddle' AND provider_customer_id = $1
		  AND ($2 = '' OR provider_subscription_id = $2)
		FOR UPDATE
	`, adjustment.ProviderCustomerID, adjustment.ProviderSubscriptionID).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO billing_lifecycle_events (
		  account_id, event_name, provider_event_id, occurred_at,
		  metadata, reason, currency_code, amount_minor
		) VALUES ($1, 'refund_issued', $2, $3,
		          jsonb_build_object('event_type', $4::text),
		          NULLIF($5, ''), $6, $7)
		ON CONFLICT (provider_event_id, event_name)
		  WHERE provider_event_id IS NOT NULL DO NOTHING
	`, accountID, adjustment.ProviderEventID, adjustment.EventOccurredAt,
		adjustment.EventType, adjustment.Reason, adjustment.CurrencyCode,
		adjustment.AmountMinor); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO billing_revenue_events (
		  provider_event_id, account_id, event_type, currency_code,
		  amount_minor, occurred_at
		) VALUES ($1, $2, 'refund', $3, $4, $5)
		ON CONFLICT (provider_event_id) DO NOTHING
	`, adjustment.ProviderEventID, accountID, adjustment.CurrencyCode,
		adjustment.AmountMinor, adjustment.EventOccurredAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE billing_webhook_events SET processed_at = $2, error = NULL
		WHERE provider = 'paddle' AND event_id = $1
	`, adjustment.ProviderEventID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RecordIgnoredBillingWebhook durably acknowledges a valid signed event that
// does not change Learnloom entitlements. Keeping these receipts makes webhook
// subscriptions auditable and prevents repeated delivery from becoming noise.
func (s *Store) RecordIgnoredBillingWebhook(
	ctx context.Context,
	receipt BillingWebhookReceipt,
	now time.Time,
) error {
	if receipt.ProviderEventID == "" || receipt.EventType == "" ||
		receipt.EventOccurredAt.IsZero() || len(receipt.PayloadSHA256) != 64 {
		return errors.New("billing webhook receipt is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	inserted, err := recordBillingWebhookTx(ctx, tx, receipt.ProviderEventID,
		receipt.EventType, receipt.EventOccurredAt, receipt.PayloadSHA256, now)
	if err != nil {
		return err
	}
	if inserted {
		if _, err := tx.Exec(ctx, `
			UPDATE billing_webhook_events SET processed_at = $2, error = NULL
			WHERE provider = 'paddle' AND event_id = $1
		`, receipt.ProviderEventID, now); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func recordBillingWebhookTx(
	ctx context.Context,
	tx pgx.Tx,
	eventID, eventType string,
	eventOccurredAt time.Time,
	payloadSHA256 string,
	now time.Time,
) (bool, error) {
	tag, err := tx.Exec(ctx, `
		INSERT INTO billing_webhook_events (
		  provider, event_id, event_type, event_occurred_at,
		  received_at, payload_sha256
		)
		VALUES ('paddle', $1, $2, $3, $4, $5)
		ON CONFLICT (provider, event_id) DO NOTHING
	`, eventID, eventType, eventOccurredAt, now, payloadSHA256)
	if err != nil {
		return false, fmt.Errorf("record billing webhook: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}
	var existingHash string
	if err := tx.QueryRow(ctx, `
		SELECT payload_sha256 FROM billing_webhook_events
		WHERE provider = 'paddle' AND event_id = $1
	`, eventID).Scan(&existingHash); err != nil {
		return false, fmt.Errorf("load billing webhook replay: %w", err)
	}
	if existingHash != payloadSHA256 {
		return false, errors.New("billing webhook event ID was replayed with a different payload")
	}
	return false, nil
}

func (s *Store) ApplyBillingLifecycleUpdate(
	ctx context.Context,
	update BillingLifecycleUpdate,
	now time.Time,
) error {
	if update.AccountID == "" || update.ProviderEventID == "" || update.EventType == "" ||
		len(update.PayloadSHA256) != 64 || update.EventOccurredAt.IsZero() {
		return errors.New("billing lifecycle update is invalid")
	}
	update.CurrencyCode = strings.ToUpper(strings.TrimSpace(update.CurrencyCode))
	if update.AmountMinor != nil && (*update.AmountMinor < 0 || len(update.CurrencyCode) != 3) {
		return errors.New("billing lifecycle amount is invalid")
	}
	if update.ProviderFeeMinor != nil && *update.ProviderFeeMinor < 0 {
		return errors.New("billing lifecycle provider fee is invalid")
	}
	if len(update.Reason) > 500 {
		return errors.New("billing lifecycle reason is too long")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	status, entitlement, lifecycleName, graceEndsAt, err := billingStatusProjection(
		update.SubscriptionStatus, update.EventType, update.PeriodEnd, now,
	)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	inserted, err := recordBillingWebhookTx(ctx, tx, update.ProviderEventID,
		update.EventType, update.EventOccurredAt, update.PayloadSHA256, now)
	if err != nil {
		return err
	}
	if !inserted {
		return tx.Commit(ctx)
	}
	if err := ensureBillingAccountTx(ctx, tx, update.AccountID, now); err != nil {
		return err
	}
	periodStart := update.PeriodStart
	periodEnd := update.PeriodEnd
	if periodStart.IsZero() {
		periodStart = now.UTC().Truncate(24 * time.Hour)
	}
	if periodEnd.IsZero() || !periodEnd.After(periodStart) {
		periodEnd = periodStart.Add(30 * 24 * time.Hour)
	}
	var updated bool
	err = tx.QueryRow(ctx, `
		WITH changed AS (
		  UPDATE account_billing SET
		    provider = 'paddle',
			    provider_customer_id = COALESCE(NULLIF($2, ''), provider_customer_id),
			    provider_subscription_id = COALESCE(NULLIF($3, ''), provider_subscription_id),
			    plan_id = $4, subscription_status = $5, entitlement_status = $6,
		    current_period_start = $7, current_period_end = $8,
		    trial_ends_at = $9, grace_ends_at = $10,
		    cancel_at_period_end = $11,
		    canceled_at = CASE WHEN $5 IN ('canceled', 'refunded') THEN $12 ELSE NULL END,
		    provider_event_at = $12, updated_at = $13
		  WHERE account_id = $1
		    AND (provider_event_at IS NULL OR provider_event_at <= $12)
		  RETURNING 1
		)
		SELECT EXISTS (SELECT 1 FROM changed)
	`, update.AccountID, update.ProviderCustomerID, update.ProviderSubscriptionID,
		billingPlanForStatus(status), status, entitlement, periodStart, periodEnd,
		update.TrialEndsAt, graceEndsAt, status == "canceled",
		update.EventOccurredAt, now).Scan(&updated)
	if err != nil {
		return fmt.Errorf("apply billing lifecycle: %w", err)
	}
	if updated && lifecycleName != "" {
		if lifecycleName == "payment_succeeded" && update.AmountMinor == nil {
			return errors.New("completed billing transaction is missing revenue totals")
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO billing_lifecycle_events (
			  account_id, event_name, provider_event_id, occurred_at, metadata,
			  reason, currency_code, amount_minor
			)
			VALUES ($1, $2, $3, $4, jsonb_build_object('event_type', $5::text),
			        NULLIF($6, ''), NULLIF($7, ''), $8)
			ON CONFLICT (provider_event_id, event_name)
			  WHERE provider_event_id IS NOT NULL DO NOTHING
		`, update.AccountID, lifecycleName, update.ProviderEventID,
			update.EventOccurredAt, update.EventType, update.Reason,
			update.CurrencyCode, update.AmountMinor); err != nil {
			return err
		}
		if update.AmountMinor != nil && (lifecycleName == "payment_succeeded" || lifecycleName == "refund_issued") {
			revenueType := "payment"
			if lifecycleName == "refund_issued" {
				revenueType = "refund"
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO billing_revenue_events (
				  provider_event_id, account_id, event_type, currency_code,
				  amount_minor, provider_fee_minor, occurred_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (provider_event_id) DO NOTHING
			`, update.ProviderEventID, update.AccountID, revenueType,
				update.CurrencyCode, *update.AmountMinor, update.ProviderFeeMinor,
				update.EventOccurredAt); err != nil {
				return err
			}
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE billing_webhook_events SET processed_at = $2, error = NULL
		WHERE provider = 'paddle' AND event_id = $1
	`, update.ProviderEventID, now); err != nil {
		return err
	}
	if lifecycleName == "payment_succeeded" && update.ProviderTransactionID != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE billing_checkout_sessions SET
			  status = 'completed', updated_at = $2
			WHERE transaction_id = $1 AND account_id = $3 AND status = 'pending'
		`, update.ProviderTransactionID, now, update.AccountID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func billingStatusProjection(
	status, eventType string,
	periodEnd, now time.Time,
) (string, string, string, *time.Time, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	lifecycle := ""
	switch status {
	case "trialing":
		lifecycle = "trial_started"
		return status, "active", lifecycle, nil, nil
	case "active":
		if eventType == "transaction.completed" {
			lifecycle = "payment_succeeded"
		} else if strings.Contains(eventType, "resumed") {
			lifecycle = "subscription_reactivated"
		}
		return status, "active", lifecycle, nil, nil
	case "past_due":
		grace := now.Add(7 * 24 * time.Hour)
		if periodEnd.After(now) {
			grace = periodEnd.Add(7 * 24 * time.Hour)
		}
		lifecycle = "payment_failed"
		return status, "grace", lifecycle, &grace, nil
	case "paused":
		return status, "generation_paused", "payment_failed", nil, nil
	case "canceled":
		return status, "generation_paused", "subscription_canceled", nil, nil
	case "refunded":
		return status, "generation_paused", "refund_issued", nil, nil
	default:
		return "", "", "", nil, fmt.Errorf("unsupported billing status %q", status)
	}
}

func billingPlanForStatus(status string) string {
	if status == "trialing" || status == "active" || status == "past_due" {
		return "pro"
	}
	return "free"
}

type BillingEntitlement struct {
	PlanID              string     `json:"planId"`
	PlanName            string     `json:"planName"`
	SubscriptionStatus  string     `json:"subscriptionStatus"`
	EntitlementStatus   string     `json:"entitlementStatus"`
	GenerationAllowance int        `json:"generationAllowance"`
	GenerationUsed      int        `json:"generationUsed"`
	GenerationRemaining int        `json:"generationRemaining"`
	PeriodStart         time.Time  `json:"periodStart"`
	PeriodEnd           time.Time  `json:"periodEnd"`
	TrialEndsAt         *time.Time `json:"trialEndsAt,omitempty"`
	GraceEndsAt         *time.Time `json:"graceEndsAt,omitempty"`
	CancelAtPeriodEnd   bool       `json:"cancelAtPeriodEnd"`
	CanGenerate         bool       `json:"canGenerate"`
}

var billingFeedbackReasons = map[string]bool{
	"too_expensive": true, "insufficient_value": true,
	"quality_concerns": true, "reliability_concerns": true,
	"allowance_too_low": true, "missing_feature": true,
	"no_longer_needed": true, "other": true,
}

func (s *Store) RecordBillingFeedback(
	ctx context.Context,
	accountID, feedbackContext, reasonCode, note string,
	now time.Time,
) error {
	feedbackContext = strings.TrimSpace(feedbackContext)
	reasonCode = strings.TrimSpace(reasonCode)
	note = strings.TrimSpace(note)
	if (feedbackContext != "non_conversion" && feedbackContext != "cancellation") ||
		!billingFeedbackReasons[reasonCode] || len([]rune(note)) > 1000 {
		return errors.New("billing feedback is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO billing_feedback (
		  account_id, context, reason_code, note, subscription_status,
		  created_at, updated_at
		)
		SELECT billing.account_id, $2, $3, NULLIF($4, ''),
		       billing.subscription_status, $5, $5
		FROM account_billing billing
		WHERE billing.account_id = $1
		  AND (
		    ($2 = 'non_conversion' AND billing.plan_id = 'free')
		    OR ($2 = 'cancellation' AND billing.subscription_status IN ('canceled', 'refunded'))
		  )
		ON CONFLICT (account_id, context) DO UPDATE SET
		  reason_code = EXCLUDED.reason_code, note = EXCLUDED.note,
		  subscription_status = EXCLUDED.subscription_status,
		  updated_at = EXCLUDED.updated_at
	`, accountID, feedbackContext, reasonCode, note, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

func (s *Store) GetBillingProviderCustomerID(ctx context.Context, accountID string) (string, error) {
	var customerID string
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(provider_customer_id, '')
		FROM account_billing WHERE account_id = $1 AND provider = 'paddle'
	`, accountID).Scan(&customerID)
	if errors.Is(err, pgx.ErrNoRows) || customerID == "" {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return customerID, nil
}

func (s *Store) RecordBillingLifecycleEvent(
	ctx context.Context,
	accountID, eventName, subjectID string,
	now time.Time,
) error {
	if eventName != "checkout_started" && eventName != "paywall_exposed" {
		return errors.New("client billing lifecycle event is invalid")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO billing_lifecycle_events (
		  account_id, event_name, provider_event_id, occurred_at, metadata
		)
		VALUES ($1, $2, NULLIF($3, ''), $4, '{}'::jsonb)
		ON CONFLICT (provider_event_id, event_name)
		  WHERE provider_event_id IS NOT NULL DO NOTHING
	`, accountID, eventName, subjectID, now)
	return err
}

func ensureBillingAccountTx(
	ctx context.Context,
	tx pgx.Tx,
	accountID string,
	now time.Time,
) error {
	periodStart := now.UTC().Truncate(24 * time.Hour)
	tag, err := tx.Exec(ctx, `
		INSERT INTO account_billing (
		  account_id, provider, plan_id, subscription_status,
		  entitlement_status, current_period_start, current_period_end,
		  created_at, updated_at
		)
		SELECT id, 'none', 'free', 'free', 'active', $2::timestamptz,
		       $2::timestamptz + interval '30 days', $2::timestamptz, $2::timestamptz
		FROM accounts WHERE id = $1 AND status = 'active'
		ON CONFLICT (account_id) DO NOTHING
	`, accountID, periodStart)
	if err != nil {
		return fmt.Errorf("ensure Account billing: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
			  SELECT 1 FROM account_billing billing
			  JOIN accounts account ON account.id = billing.account_id
			  WHERE billing.account_id = $1 AND account.status = 'active'
			)
		`, accountID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
	}
	return nil
}

func reserveGenerationUsageTx(
	ctx context.Context,
	tx pgx.Tx,
	accountID, issueID string,
	now time.Time,
) error {
	if err := ensureBillingAccountTx(ctx, tx, accountID, now); err != nil {
		return err
	}
	var planID, entitlementStatus string
	var allowance int
	var periodStart, periodEnd time.Time
	var graceEndsAt *time.Time
	err := tx.QueryRow(ctx, `
		SELECT billing.plan_id, billing.entitlement_status,
		       plan.generation_allowance,
		       billing.current_period_start, billing.current_period_end,
		       billing.grace_ends_at
		FROM account_billing billing
		JOIN billing_plans plan ON plan.id = billing.plan_id AND plan.active
		WHERE billing.account_id = $1
		FOR UPDATE OF billing
	`, accountID).Scan(
		&planID, &entitlementStatus, &allowance,
		&periodStart, &periodEnd, &graceEndsAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrEntitlementRequired
	}
	if err != nil {
		return fmt.Errorf("load generation entitlement: %w", err)
	}
	if now.Before(periodStart) || !now.Before(periodEnd) {
		periodStart = now.UTC().Truncate(24 * time.Hour)
		periodEnd = periodStart.Add(30 * 24 * time.Hour)
		if _, err := tx.Exec(ctx, `
			UPDATE account_billing SET
			  current_period_start = $2, current_period_end = $3, updated_at = $2
			WHERE account_id = $1
		`, accountID, periodStart, periodEnd); err != nil {
			return err
		}
	}
	if entitlementStatus == "generation_paused" ||
		(entitlementStatus == "grace" && (graceEndsAt == nil || !now.Before(*graceEndsAt))) {
		return ErrEntitlementRequired
	}
	var used int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(sum(units), 0)::int
		FROM generation_usage_reservations
		WHERE account_id = $1 AND period_start = $2
		  AND state IN ('reserved', 'consumed')
	`, accountID, periodStart).Scan(&used); err != nil {
		return err
	}
	if used >= allowance {
		return ErrEntitlementRequired
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO generation_usage_reservations (
		  issue_id, account_id, plan_id, period_start, units, state, reserved_at
		)
		VALUES ($1, $2, $3, $4, 1, 'reserved', $5)
		ON CONFLICT (issue_id) DO NOTHING
	`, issueID, accountID, planID, periodStart, now)
	if err != nil {
		return fmt.Errorf("reserve generation usage: %w", err)
	}
	return nil
}

func (s *Store) GetBillingEntitlement(
	ctx context.Context,
	accountID string,
	now time.Time,
) (BillingEntitlement, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return BillingEntitlement{}, err
	}
	defer rollback(tx)
	if err := ensureBillingAccountTx(ctx, tx, accountID, now); err != nil {
		return BillingEntitlement{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE account_billing billing SET
		  current_period_start = date_trunc('day', $2::timestamptz),
		  current_period_end = date_trunc('day', $2::timestamptz)
		    + make_interval(days => plan.period_days),
		  updated_at = $2
		FROM billing_plans plan
		WHERE billing.account_id = $1 AND plan.id = billing.plan_id
		  AND ($2::timestamptz < billing.current_period_start
		       OR $2::timestamptz >= billing.current_period_end)
	`, accountID, now); err != nil {
		return BillingEntitlement{}, fmt.Errorf("roll billing entitlement period: %w", err)
	}
	var result BillingEntitlement
	err = tx.QueryRow(ctx, `
		SELECT billing.plan_id, plan.display_name, billing.subscription_status,
		       billing.entitlement_status, plan.generation_allowance,
		       COALESCE(usage.used, 0)::int,
		       GREATEST(0, plan.generation_allowance - COALESCE(usage.used, 0))::int,
		       billing.current_period_start, billing.current_period_end,
		       billing.trial_ends_at, billing.grace_ends_at,
		       billing.cancel_at_period_end
		FROM account_billing billing
		JOIN billing_plans plan ON plan.id = billing.plan_id
		LEFT JOIN LATERAL (
		  SELECT sum(units)::int AS used
		  FROM generation_usage_reservations reservation
		  WHERE reservation.account_id = billing.account_id
		    AND reservation.period_start = billing.current_period_start
		    AND reservation.state IN ('reserved', 'consumed')
		) usage ON true
		WHERE billing.account_id = $1
	`, accountID).Scan(
		&result.PlanID, &result.PlanName, &result.SubscriptionStatus,
		&result.EntitlementStatus, &result.GenerationAllowance,
		&result.GenerationUsed, &result.GenerationRemaining,
		&result.PeriodStart, &result.PeriodEnd, &result.TrialEndsAt,
		&result.GraceEndsAt, &result.CancelAtPeriodEnd,
	)
	if err != nil {
		return BillingEntitlement{}, err
	}
	result.CanGenerate = result.GenerationRemaining > 0 &&
		(result.EntitlementStatus == "active" ||
			(result.EntitlementStatus == "grace" && result.GraceEndsAt != nil && now.Before(*result.GraceEndsAt)))
	if err := tx.Commit(ctx); err != nil {
		return BillingEntitlement{}, err
	}
	return result, nil
}
