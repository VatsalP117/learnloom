package httpapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/VatsalP117/learnloom/internal/store"
)

type paddleWebhookEvent struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Data       json.RawMessage `json:"data"`
}

type paddleSubscriptionData struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Status     string `json:"status"`
	CustomData struct {
		AccountID string `json:"account_id"`
	} `json:"custom_data"`
	CurrentBillingPeriod *struct {
		StartsAt time.Time `json:"starts_at"`
		EndsAt   time.Time `json:"ends_at"`
	} `json:"current_billing_period"`
	NextBilledAt *time.Time `json:"next_billed_at"`
	Items        []struct {
		Price struct {
			ID string `json:"id"`
		} `json:"price"`
	} `json:"items"`
	ScheduledChange *struct {
		Action      string     `json:"action"`
		EffectiveAt *time.Time `json:"effective_at"`
	} `json:"scheduled_change"`
}

type paddleTransactionData struct {
	ID             string `json:"id"`
	CustomerID     string `json:"customer_id"`
	SubscriptionID string `json:"subscription_id"`
	CustomData     struct {
		AccountID string `json:"account_id"`
	} `json:"custom_data"`
	BillingPeriod *struct {
		StartsAt time.Time `json:"starts_at"`
		EndsAt   time.Time `json:"ends_at"`
	} `json:"billing_period"`
	CurrencyCode string `json:"currency_code"`
	Details      struct {
		Totals struct {
			Subtotal     string  `json:"subtotal"`
			Fee          *string `json:"fee"`
			CurrencyCode string  `json:"currency_code"`
		} `json:"totals"`
	} `json:"details"`
	Items []struct {
		Price struct {
			ID string `json:"id"`
		} `json:"price"`
	} `json:"items"`
}

type paddleAdjustmentData struct {
	ID             string `json:"id"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	CustomerID     string `json:"customer_id"`
	SubscriptionID string `json:"subscription_id"`
	Reason         string `json:"reason"`
	CurrencyCode   string `json:"currency_code"`
	Totals         struct {
		Total        string `json:"total"`
		CurrencyCode string `json:"currency_code"`
	} `json:"totals"`
}

func (s *Server) handlePaddleWebhook(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		methodNotAllowed(response, http.MethodPost)
		return
	}
	if s.cfg.PaddleWebhookSecret == "" {
		writeProblem(response, http.StatusNotFound, "not_found", "Page not found.")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, s.cfg.MaxRequestBodyBytes)
	body, err := io.ReadAll(request.Body)
	if err != nil {
		var maximum *http.MaxBytesError
		if errors.As(err, &maximum) {
			writeProblem(response, http.StatusRequestEntityTooLarge, "request_too_large", "The webhook is too large.")
		} else {
			writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The webhook body is invalid.")
		}
		return
	}
	if !verifyPaddleSignature(
		body, request.Header.Get("Paddle-Signature"), s.cfg.PaddleWebhookSecret,
		time.Now().UTC(), 5*time.Minute,
	) {
		writeProblem(response, http.StatusBadRequest, "invalid_webhook_signature", "The webhook signature is invalid.")
		return
	}
	var event paddleWebhookEvent
	if json.Unmarshal(body, &event) != nil || event.EventID == "" ||
		event.EventType == "" || event.OccurredAt.IsZero() {
		writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The webhook payload is invalid.")
		return
	}
	sum := sha256.Sum256(body)
	recordIgnored := func() bool {
		err := s.store.RecordIgnoredBillingWebhook(request.Context(), store.BillingWebhookReceipt{
			ProviderEventID: event.EventID, EventType: event.EventType,
			EventOccurredAt: event.OccurredAt, PayloadSHA256: hex.EncodeToString(sum[:]),
		}, time.Now().UTC())
		if err != nil {
			s.internalError(response, request, err)
			return false
		}
		response.WriteHeader(http.StatusNoContent)
		return true
	}
	if !strings.HasPrefix(event.EventType, "subscription.") &&
		!strings.HasPrefix(event.EventType, "transaction.") &&
		!strings.HasPrefix(event.EventType, "adjustment.") {
		recordIgnored()
		return
	}
	if strings.HasPrefix(event.EventType, "adjustment.") {
		var adjustment paddleAdjustmentData
		if json.Unmarshal(event.Data, &adjustment) != nil || adjustment.ID == "" ||
			adjustment.CustomerID == "" {
			writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The adjustment payload is incomplete.")
			return
		}
		if adjustment.Action != "refund" || adjustment.Status != "approved" {
			recordIgnored()
			return
		}
		amount, parseErr := strconv.ParseInt(adjustment.Totals.Total, 10, 64)
		currency := adjustment.Totals.CurrencyCode
		if currency == "" {
			currency = adjustment.CurrencyCode
		}
		if parseErr != nil || amount < 0 || len(currency) != 3 {
			writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The adjustment amount is invalid.")
			return
		}
		err = s.store.ApplyBillingRefundAdjustment(request.Context(), store.BillingRefundAdjustment{
			ProviderEventID: event.EventID, EventType: event.EventType,
			ProviderCustomerID:     adjustment.CustomerID,
			ProviderSubscriptionID: adjustment.SubscriptionID,
			Reason:                 adjustment.Reason, CurrencyCode: currency, AmountMinor: amount,
			EventOccurredAt: event.OccurredAt, PayloadSHA256: hex.EncodeToString(sum[:]),
		}, time.Now().UTC())
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		response.WriteHeader(http.StatusNoContent)
		return
	}
	if strings.HasPrefix(event.EventType, "transaction.") {
		var transaction paddleTransactionData
		if json.Unmarshal(event.Data, &transaction) != nil ||
			transaction.ID == "" || transaction.CustomData.AccountID == "" {
			writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The transaction payload is incomplete.")
			return
		}
		planID, recognized := s.paddlePlanForTransaction(transaction)
		if !recognized {
			recordIgnored()
			return
		}
		status := ""
		switch event.EventType {
		case "transaction.completed":
			status = "active"
		case "transaction.payment_failed":
			status = "past_due"
		default:
			recordIgnored()
			return
		}
		periodStart, periodEnd := time.Time{}, time.Time{}
		if transaction.BillingPeriod != nil {
			periodStart = transaction.BillingPeriod.StartsAt
			periodEnd = transaction.BillingPeriod.EndsAt
		}
		var amountMinor, providerFeeMinor *int64
		if transaction.Details.Totals.Subtotal != "" {
			amount, parseErr := strconv.ParseInt(transaction.Details.Totals.Subtotal, 10, 64)
			if parseErr != nil || amount < 0 {
				writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The transaction amount is invalid.")
				return
			}
			amountMinor = &amount
		}
		if transaction.Details.Totals.Fee != nil {
			fee, parseErr := strconv.ParseInt(*transaction.Details.Totals.Fee, 10, 64)
			if parseErr != nil || fee < 0 {
				writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The transaction fee is invalid.")
				return
			}
			providerFeeMinor = &fee
		}
		currency := transaction.Details.Totals.CurrencyCode
		if currency == "" {
			currency = transaction.CurrencyCode
		}
		err = s.store.ApplyBillingLifecycleUpdate(request.Context(), store.BillingLifecycleUpdate{
			AccountID:       transaction.CustomData.AccountID,
			PlanID:          planID,
			ProviderEventID: event.EventID, EventType: event.EventType,
			ProviderCustomerID:     transaction.CustomerID,
			ProviderSubscriptionID: transaction.SubscriptionID,
			ProviderTransactionID:  transaction.ID,
			SubscriptionStatus:     status, PeriodStart: periodStart, PeriodEnd: periodEnd,
			EventOccurredAt: event.OccurredAt, PayloadSHA256: hex.EncodeToString(sum[:]),
			CurrencyCode: currency, AmountMinor: amountMinor,
			ProviderFeeMinor: providerFeeMinor,
		}, time.Now().UTC())
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		response.WriteHeader(http.StatusNoContent)
		return
	}
	var subscription paddleSubscriptionData
	if json.Unmarshal(event.Data, &subscription) != nil ||
		subscription.ID == "" || subscription.CustomerID == "" ||
		subscription.CustomData.AccountID == "" {
		writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The subscription payload is incomplete.")
		return
	}
	planID, recognized := s.paddlePlanForSubscription(subscription)
	if !recognized {
		recordIgnored()
		return
	}
	status := subscription.Status
	if event.EventType == "subscription.canceled" {
		status = "canceled"
	}
	periodStart, periodEnd := time.Time{}, time.Time{}
	if subscription.CurrentBillingPeriod != nil {
		periodStart = subscription.CurrentBillingPeriod.StartsAt
		periodEnd = subscription.CurrentBillingPeriod.EndsAt
	}
	var trialEndsAt *time.Time
	if status == "trialing" {
		trialEndsAt = subscription.NextBilledAt
	}
	// cancel_at_period_end is true only when Paddle reports a scheduled
	// cancellation that takes effect in the future; every other subscription
	// state (including a removed or expired scheduled change) clears it.
	cancelAtPeriodEnd := status == "canceled"
	if subscription.ScheduledChange != nil {
		cancelAtPeriodEnd = subscription.ScheduledChange.Action == "cancel" &&
			subscription.ScheduledChange.EffectiveAt != nil &&
			subscription.ScheduledChange.EffectiveAt.After(event.OccurredAt)
	}
	err = s.store.ApplyBillingLifecycleUpdate(request.Context(), store.BillingLifecycleUpdate{
		AccountID:       subscription.CustomData.AccountID,
		PlanID:          planID,
		ProviderEventID: event.EventID, EventType: event.EventType,
		ProviderCustomerID:     subscription.CustomerID,
		ProviderSubscriptionID: subscription.ID, SubscriptionStatus: status,
		PeriodStart: periodStart, PeriodEnd: periodEnd, TrialEndsAt: trialEndsAt,
		CancelAtPeriodEnd: &cancelAtPeriodEnd,
		EventOccurredAt:   event.OccurredAt, PayloadSHA256: hex.EncodeToString(sum[:]),
	}, time.Now().UTC())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func paddleTransactionContainsPrice(transaction paddleTransactionData, priceID string) bool {
	if priceID == "" {
		return false
	}
	for _, item := range transaction.Items {
		if item.Price.ID == priceID {
			return true
		}
	}
	return false
}

func paddleSubscriptionContainsPrice(subscription paddleSubscriptionData, priceID string) bool {
	if priceID == "" {
		return false
	}
	for _, item := range subscription.Items {
		if item.Price.ID == priceID {
			return true
		}
	}
	return false
}

func (s *Server) paddlePlanForTransaction(transaction paddleTransactionData) (string, bool) {
	planID := ""
	for _, item := range transaction.Items {
		matched, ok := s.paddlePlanForPrice(item.Price.ID)
		if !ok {
			continue
		}
		if planID != "" && planID != matched {
			return "", false
		}
		planID = matched
	}
	return planID, planID != ""
}

func (s *Server) paddlePlanForSubscription(subscription paddleSubscriptionData) (string, bool) {
	planID := ""
	for _, item := range subscription.Items {
		matched, ok := s.paddlePlanForPrice(item.Price.ID)
		if !ok {
			continue
		}
		if planID != "" && planID != matched {
			return "", false
		}
		planID = matched
	}
	return planID, planID != ""
}

func verifyPaddleSignature(
	body []byte,
	header, secret string,
	now time.Time,
	tolerance time.Duration,
) bool {
	var timestamp int64
	var signatures []string
	for part := range strings.SplitSeq(header, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "ts":
			timestamp, _ = strconv.ParseInt(value, 10, 64)
		case "h1":
			signatures = append(signatures, value)
		}
	}
	if timestamp < 1 || len(signatures) == 0 || secret == "" {
		return false
	}
	signedAt := time.Unix(timestamp, 0)
	difference := now.Sub(signedAt)
	if difference < -tolerance || difference > tolerance {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	_, _ = mac.Write([]byte(":"))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	for _, signature := range signatures {
		decoded, err := hex.DecodeString(signature)
		if err == nil && hmac.Equal(decoded, expected) {
			return true
		}
	}
	return false
}
