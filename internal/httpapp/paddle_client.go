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

const maxPaddleResponseBytes = 1 << 20

type paddleCheckoutRequest struct {
	Items []struct {
		PriceID  string `json:"price_id"`
		Quantity int    `json:"quantity"`
	} `json:"items"`
	CollectionMode string         `json:"collection_mode"`
	CustomerID     string         `json:"customer_id,omitempty"`
	CustomData     map[string]any `json:"custom_data"`
	Checkout       struct {
		URL string `json:"url"`
	} `json:"checkout"`
}

type paddleCheckoutResponse struct {
	Data struct {
		ID       string `json:"id"`
		Checkout struct {
			URL string `json:"url"`
		} `json:"checkout"`
	} `json:"data"`
}

type paddlePortalResponse struct {
	Data struct {
		URLs struct {
			General struct {
				Overview string `json:"overview"`
			} `json:"general"`
		} `json:"urls"`
	} `json:"data"`
}

func (s *Server) createPaddleCheckout(
	ctx context.Context,
	accountID, planID, customerID string,
) (string, string, error) {
	if !s.paddleConfigured() {
		return "", "", errors.New("billing provider is unavailable")
	}
	priceID, ok := s.paddlePriceForPlan(planID)
	if !ok {
		return "", "", errors.New("billing plan is invalid")
	}
	if customerID != "" && !strings.HasPrefix(customerID, "ctm_") {
		customerID = ""
	}
	var payload paddleCheckoutRequest
	payload.Items = append(payload.Items, struct {
		PriceID  string `json:"price_id"`
		Quantity int    `json:"quantity"`
	}{PriceID: priceID, Quantity: 1})
	payload.CollectionMode = "automatic"
	payload.CustomerID = customerID
	payload.CustomData = map[string]any{"account_id": accountID, "plan_id": planID}
	payload.Checkout.URL = strings.TrimRight(s.cfg.AppOrigin, "/") + "/checkout"
	var response paddleCheckoutResponse
	if err := s.callPaddle(ctx, http.MethodPost, "/transactions", payload, &response); err != nil {
		return "", "", err
	}
	checkoutURL, err := url.Parse(response.Data.Checkout.URL)
	appOrigin, originErr := url.Parse(s.cfg.AppOrigin)
	if response.Data.ID == "" || !strings.HasPrefix(response.Data.ID, "txn_") ||
		err != nil || originErr != nil || !safeHTTPSURL(checkoutURL) ||
		!strings.EqualFold(checkoutURL.Host, appOrigin.Host) {
		return "", "", errors.New("billing provider returned an invalid checkout")
	}
	return response.Data.ID, response.Data.Checkout.URL, nil
}

func (s *Server) paddleCheckoutURL(transactionID string) (string, error) {
	if !strings.HasPrefix(transactionID, "txn_") {
		return "", errors.New("billing transaction reference is invalid")
	}
	checkoutURL, err := url.Parse(strings.TrimRight(s.cfg.AppOrigin, "/") + "/checkout")
	if err != nil || !safeHTTPSURL(checkoutURL) {
		return "", errors.New("billing return origin is invalid")
	}
	query := checkoutURL.Query()
	query.Set("_ptxn", transactionID)
	checkoutURL.RawQuery = query.Encode()
	return checkoutURL.String(), nil
}

func (s *Server) createPaddlePortal(ctx context.Context, customerID string) (string, error) {
	if !s.paddleConfigured() || !strings.HasPrefix(customerID, "ctm_") {
		return "", errors.New("billing provider reference is invalid")
	}
	var response paddlePortalResponse
	path := "/customers/" + url.PathEscape(customerID) + "/portal-sessions"
	if err := s.callPaddle(ctx, http.MethodPost, path, struct{}{}, &response); err != nil {
		return "", err
	}
	portalURL, err := url.Parse(response.Data.URLs.General.Overview)
	if err != nil || !safeHTTPSURL(portalURL) ||
		(portalURL.Hostname() != "paddle.com" && !strings.HasSuffix(portalURL.Hostname(), ".paddle.com")) {
		return "", errors.New("billing provider returned an invalid portal URL")
	}
	return response.Data.URLs.General.Overview, nil
}

func (s *Server) paddleConfigured() bool {
	approved := s.cfg.Environment != "production" || s.cfg.PaidCommerceApproved
	return approved && s.cfg.PaddleAPIKey != "" && s.cfg.PaddleAPIBaseURL != "" &&
		s.cfg.PaddleClientToken != "" && s.cfg.PaddleEssentialPriceID != "" &&
		s.cfg.PaddleProPriceID != "" && s.cfg.PaddleWebhookSecret != ""
}

func (s *Server) paddlePriceForPlan(planID string) (string, bool) {
	switch planID {
	case "essential":
		return s.cfg.PaddleEssentialPriceID, s.cfg.PaddleEssentialPriceID != ""
	case "pro":
		return s.cfg.PaddleProPriceID, s.cfg.PaddleProPriceID != ""
	default:
		return "", false
	}
}

func (s *Server) paddlePlanForPrice(priceID string) (string, bool) {
	if priceID != "" && priceID == s.cfg.PaddleEssentialPriceID {
		return "essential", true
	}
	if priceID != "" && priceID == s.cfg.PaddleProPriceID {
		return "pro", true
	}
	return "", false
}

// handleBillingConfig serves the public, unauthenticated commerce
// configuration needed by the client-side checkout page. It exposes only
// whether checkout is available, the Paddle environment, and the public
// client-side token. Secret fields (API key, webhook secret, price IDs) are
// never included. Responses are no-store so downstream caches cannot pin an
// old token or stale availability.
func (s *Server) handleBillingConfig(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		methodNotAllowed(response, http.MethodGet)
		return
	}
	payload := map[string]any{
		"commerceAvailable": false,
		"environment":       s.paddleEnvironment(),
	}
	if s.paddleConfigured() {
		payload["commerceAvailable"] = true
		payload["clientToken"] = s.cfg.PaddleClientToken
	}
	writeJSON(response, http.StatusOK, payload)
}

func (s *Server) paddleEnvironment() string {
	if s.cfg.Environment == "production" {
		return "production"
	}
	return "sandbox"
}

func (s *Server) callPaddle(
	ctx context.Context,
	method, path string,
	payload any,
	destination any,
) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method,
		strings.TrimRight(s.cfg.PaddleAPIBaseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+s.cfg.PaddleAPIKey)
	request.Header.Set("Paddle-Version", "1")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	client := s.cfg.PaddleHTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("call billing provider: %w", err)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, maxPaddleResponseBytes+1)
	responseBody, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("read billing provider response: %w", err)
	}
	if len(responseBody) > maxPaddleResponseBytes {
		return errors.New("billing provider response is too large")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// Do not copy provider response bodies into logs: they may contain
		// customer or payment metadata.
		return fmt.Errorf("billing provider returned HTTP %d", response.StatusCode)
	}
	if err := json.Unmarshal(responseBody, destination); err != nil {
		return errors.New("billing provider returned malformed JSON")
	}
	return nil
}

func safeHTTPSURL(parsed *url.URL) bool {
	return parsed != nil && parsed.Scheme == "https" && parsed.Host != "" &&
		parsed.User == nil && parsed.IsAbs()
}
