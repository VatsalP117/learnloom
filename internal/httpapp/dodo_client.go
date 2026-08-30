package httpapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxDodoResponseBytes = 1 << 20

type dodoCheckoutRequest struct {
	ProductCart []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"product_cart"`
	Customer *struct {
		CustomerID string `json:"customer_id"`
	} `json:"customer,omitempty"`
	ReturnURL string            `json:"return_url"`
	Metadata  map[string]string `json:"metadata"`
}

type dodoCheckoutResponse struct {
	SessionID   string `json:"session_id"`
	CheckoutURL string `json:"checkout_url"`
}

func (s *Server) createDodoCheckout(ctx context.Context, accountID, planID, customerID string) (string, string, error) {
	if !s.dodoConfigured() {
		return "", "", errors.New("billing provider is unavailable")
	}
	productID, ok := s.dodoProductForPlan(planID)
	if !ok {
		return "", "", errors.New("billing plan is invalid")
	}
	var payload dodoCheckoutRequest
	payload.ProductCart = append(payload.ProductCart, struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}{productID, 1})
	if strings.HasPrefix(customerID, "cus_") {
		payload.Customer = &struct {
			CustomerID string `json:"customer_id"`
		}{customerID}
	}
	payload.ReturnURL = strings.TrimRight(s.cfg.AppOrigin, "/") + "/settings?checkout=complete"
	payload.Metadata = map[string]string{"account_id": accountID, "plan_id": planID}
	var result dodoCheckoutResponse
	if err := s.callDodo(ctx, http.MethodPost, "/checkouts", payload, &result); err != nil {
		return "", "", err
	}
	if !strings.HasPrefix(result.SessionID, "cks_") || !s.validDodoCheckoutURL(result.CheckoutURL) {
		return "", "", errors.New("billing provider returned an invalid checkout")
	}
	return result.SessionID, result.CheckoutURL, nil
}

func (s *Server) dodoCheckoutURL(sessionID string) (string, error) {
	if !strings.HasPrefix(sessionID, "cks_") {
		return "", errors.New("billing checkout reference is invalid")
	}
	host := "test.checkout.dodopayments.com"
	if s.dodoEnvironment() == "live_mode" {
		host = "checkout.dodopayments.com"
	}
	return "https://" + host + "/session/" + url.PathEscape(sessionID), nil
}

func (s *Server) validDodoCheckoutURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || !safeHTTPSURL(parsed) {
		return false
	}
	expected := "test.checkout.dodopayments.com"
	if s.dodoEnvironment() == "live_mode" {
		expected = "checkout.dodopayments.com"
	}
	return strings.EqualFold(parsed.Hostname(), expected)
}

func (s *Server) createDodoPortal(ctx context.Context, customerID string) (string, error) {
	if !s.dodoConfigured() || !strings.HasPrefix(customerID, "cus_") {
		return "", errors.New("billing provider reference is invalid")
	}
	path := "/customers/" + url.PathEscape(customerID) + "/customer-portal/session?return_url=" +
		url.QueryEscape(strings.TrimRight(s.cfg.AppOrigin, "/")+"/settings")
	var result struct {
		Link string `json:"link"`
	}
	if err := s.callDodo(ctx, http.MethodPost, path, struct{}{}, &result); err != nil {
		return "", err
	}
	parsed, err := url.Parse(result.Link)
	if err != nil || !safeHTTPSURL(parsed) ||
		(parsed.Hostname() != "customer.dodopayments.com" && !strings.HasSuffix(parsed.Hostname(), ".dodopayments.com")) {
		return "", errors.New("billing provider returned an invalid portal URL")
	}
	return result.Link, nil
}

func (s *Server) dodoConfigured() bool {
	approved := s.cfg.Environment != "production" || s.cfg.PaidCommerceApproved
	return approved && s.cfg.DodoAPIKey != "" && s.cfg.DodoAPIBaseURL != "" &&
		s.cfg.DodoWebhookSecret != "" && s.cfg.DodoEssentialProductID != "" && s.cfg.DodoProProductID != ""
}

func (s *Server) dodoProductForPlan(planID string) (string, bool) {
	switch planID {
	case "essential":
		return s.cfg.DodoEssentialProductID, s.cfg.DodoEssentialProductID != ""
	case "pro":
		return s.cfg.DodoProProductID, s.cfg.DodoProProductID != ""
	default:
		return "", false
	}
}

func (s *Server) dodoPlanForProduct(productID string) (string, bool) {
	if productID != "" && productID == s.cfg.DodoEssentialProductID {
		return "essential", true
	}
	if productID != "" && productID == s.cfg.DodoProProductID {
		return "pro", true
	}
	return "", false
}

func (s *Server) dodoEnvironment() string {
	if s.cfg.Environment == "production" {
		return "live_mode"
	}
	return "test_mode"
}

func (s *Server) callDodo(ctx context.Context, method, path string, payload, destination any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(s.cfg.DodoAPIBaseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+s.cfg.DodoAPIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	client := s.cfg.DodoHTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("call billing provider: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxDodoResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read billing provider response: %w", err)
	}
	if len(responseBody) > maxDodoResponseBytes {
		return errors.New("billing provider response is too large")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("billing provider returned HTTP %d", response.StatusCode)
	}
	if err := json.Unmarshal(responseBody, destination); err != nil {
		return errors.New("billing provider returned malformed JSON")
	}
	return nil
}

func safeHTTPSURL(parsed *url.URL) bool {
	return parsed != nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.IsAbs()
}
