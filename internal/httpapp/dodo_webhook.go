package httpapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

type dodoWebhookEvent struct {
	BusinessID string          `json:"business_id"`
	Type       string          `json:"type"`
	Timestamp  time.Time       `json:"timestamp"`
	Data       json.RawMessage `json:"data"`
}

type dodoCustomer struct {
	CustomerID string `json:"customer_id"`
}
type dodoMetadata struct {
	AccountID string `json:"account_id"`
	PlanID    string `json:"plan_id"`
}

type dodoSubscriptionData struct {
	SubscriptionID          string       `json:"subscription_id"`
	Customer                dodoCustomer `json:"customer"`
	ProductID               string       `json:"product_id"`
	Metadata                dodoMetadata `json:"metadata"`
	Status                  string       `json:"status"`
	PreviousBillingDate     time.Time    `json:"previous_billing_date"`
	NextBillingDate         time.Time    `json:"next_billing_date"`
	TrialPeriodDays         int          `json:"trial_period_days"`
	CancelAtNextBillingDate bool         `json:"cancel_at_next_billing_date"`
}

type dodoPaymentData struct {
	PaymentID         string       `json:"payment_id"`
	Customer          dodoCustomer `json:"customer"`
	SubscriptionID    string       `json:"subscription_id"`
	CheckoutSessionID string       `json:"checkout_session_id"`
	Metadata          dodoMetadata `json:"metadata"`
	Currency          string       `json:"currency"`
	TotalAmount       int64        `json:"total_amount"`
	Tax               *int64       `json:"tax"`
}

type dodoRefundData struct {
	RefundID string       `json:"refund_id"`
	Customer dodoCustomer `json:"customer"`
	Metadata dodoMetadata `json:"metadata"`
	Amount   *int64       `json:"amount"`
	Currency string       `json:"currency"`
	Reason   string       `json:"reason"`
}

func (s *Server) handleDodoWebhook(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		methodNotAllowed(response, http.MethodPost)
		return
	}
	if s.cfg.DodoWebhookSecret == "" {
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
	eventID := request.Header.Get("webhook-id")
	if !verifyDodoSignature(body, eventID, request.Header.Get("webhook-timestamp"), request.Header.Get("webhook-signature"), s.cfg.DodoWebhookSecret, time.Now().UTC(), 5*time.Minute) {
		writeProblem(response, http.StatusBadRequest, "invalid_webhook_signature", "The webhook signature is invalid.")
		return
	}
	var event dodoWebhookEvent
	if json.Unmarshal(body, &event) != nil || eventID == "" || event.Type == "" || event.Timestamp.IsZero() {
		writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The webhook payload is invalid.")
		return
	}
	sum := sha256.Sum256(body)
	receipt := store.BillingWebhookReceipt{ProviderEventID: eventID, EventType: event.Type, EventOccurredAt: event.Timestamp, PayloadSHA256: hex.EncodeToString(sum[:])}
	ignore := func() bool {
		if err := s.store.RecordIgnoredBillingWebhook(request.Context(), receipt, time.Now().UTC()); err != nil {
			s.internalError(response, request, err)
			return false
		}
		response.WriteHeader(http.StatusNoContent)
		return true
	}

	if event.Type == "payment.succeeded" {
		var payment dodoPaymentData
		if json.Unmarshal(event.Data, &payment) != nil || payment.PaymentID == "" || payment.Customer.CustomerID == "" || payment.SubscriptionID == "" || payment.TotalAmount < 0 {
			writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The payment payload is incomplete.")
			return
		}
		tax := int64(0)
		if payment.Tax != nil {
			tax = *payment.Tax
		}
		if tax < 0 || tax > payment.TotalAmount || len(payment.Currency) != 3 {
			writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The payment amount is invalid.")
			return
		}
		err := s.store.ApplyBillingPayment(request.Context(), store.BillingPayment{
			ProviderEventID: eventID, EventType: event.Type, ProviderCustomerID: payment.Customer.CustomerID,
			ProviderSubscriptionID: payment.SubscriptionID, ProviderTransactionID: payment.PaymentID,
			CheckoutSessionID: payment.CheckoutSessionID, AccountID: payment.Metadata.AccountID, PlanID: payment.Metadata.PlanID,
			CurrencyCode: payment.Currency, AmountMinor: payment.TotalAmount - tax, EventOccurredAt: event.Timestamp, PayloadSHA256: receipt.PayloadSHA256,
		}, time.Now().UTC())
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		response.WriteHeader(http.StatusNoContent)
		return
	}
	if event.Type == "refund.succeeded" {
		var refund dodoRefundData
		if json.Unmarshal(event.Data, &refund) != nil || refund.RefundID == "" || refund.Customer.CustomerID == "" || refund.Amount == nil || *refund.Amount < 0 || len(refund.Currency) != 3 {
			writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The refund payload is incomplete.")
			return
		}
		err := s.store.ApplyBillingRefundAdjustment(request.Context(), store.BillingRefundAdjustment{ProviderEventID: eventID, EventType: event.Type, ProviderCustomerID: refund.Customer.CustomerID, Reason: refund.Reason, CurrencyCode: refund.Currency, AmountMinor: *refund.Amount, EventOccurredAt: event.Timestamp, PayloadSHA256: receipt.PayloadSHA256}, time.Now().UTC())
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		response.WriteHeader(http.StatusNoContent)
		return
	}
	if !strings.HasPrefix(event.Type, "subscription.") {
		ignore()
		return
	}
	var subscription dodoSubscriptionData
	if json.Unmarshal(event.Data, &subscription) != nil || subscription.SubscriptionID == "" || subscription.Customer.CustomerID == "" || subscription.Metadata.AccountID == "" {
		writeProblem(response, http.StatusBadRequest, "invalid_webhook", "The subscription payload is incomplete.")
		return
	}
	planID, recognized := s.dodoPlanForProduct(subscription.ProductID)
	if !recognized || subscription.Metadata.PlanID != planID {
		ignore()
		return
	}
	status, changes := dodoSubscriptionStatus(subscription.Status, event.Type)
	if !changes {
		ignore()
		return
	}
	var trialEndsAt *time.Time
	if status == "active" && subscription.TrialPeriodDays > 0 && subscription.NextBillingDate.After(event.Timestamp) {
		status = "trialing"
		trialEndsAt = &subscription.NextBillingDate
	}
	cancelAtPeriodEnd := subscription.CancelAtNextBillingDate
	err = s.store.ApplyBillingLifecycleUpdate(request.Context(), store.BillingLifecycleUpdate{
		AccountID: subscription.Metadata.AccountID, PlanID: planID, ProviderEventID: eventID, EventType: event.Type,
		ProviderCustomerID: subscription.Customer.CustomerID, ProviderSubscriptionID: subscription.SubscriptionID,
		SubscriptionStatus: status, PeriodStart: subscription.PreviousBillingDate, PeriodEnd: subscription.NextBillingDate,
		TrialEndsAt: trialEndsAt, CancelAtPeriodEnd: &cancelAtPeriodEnd, EventOccurredAt: event.Timestamp, PayloadSHA256: receipt.PayloadSHA256,
	}, time.Now().UTC())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func dodoSubscriptionStatus(status, eventType string) (string, bool) {
	if eventType == "subscription.failed" || status == "pending" || status == "failed" {
		return "", false
	}
	switch status {
	case "active":
		return "active", true
	case "on_hold":
		return "past_due", true
	case "paused":
		return "paused", true
	case "cancelled", "expired":
		return "canceled", true
	}
	return "", false
}

func verifyDodoSignature(body []byte, eventID, timestampHeader, signatureHeader, secret string, now time.Time, tolerance time.Duration) bool {
	if eventID == "" || strings.Contains(eventID, ".") || secret == "" {
		return false
	}
	timestamp, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil || timestamp < 1 {
		return false
	}
	difference := now.Sub(time.Unix(timestamp, 0))
	if difference < -tolerance || difference > tolerance {
		return false
	}
	secret = strings.TrimPrefix(secret, "whsec_")
	key, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		key, err = base64.RawStdEncoding.DecodeString(secret)
	}
	if err != nil || len(key) == 0 {
		return false
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(eventID + "." + timestampHeader + "."))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	for _, candidate := range strings.Fields(signatureHeader) {
		version, encoded, ok := strings.Cut(candidate, ",")
		if !ok || version != "v1" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err == nil && hmac.Equal(decoded, expected) {
			return true
		}
	}
	return false
}
