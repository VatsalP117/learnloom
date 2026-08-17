package httpapp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCreatePaddleCheckoutUsesAccountAttributionAndApprovedReturnURL(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/transactions" {
			t.Fatalf("request=%s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer paddle-key" {
			t.Fatalf("authorization=%q", request.Header.Get("Authorization"))
		}
		if request.Header.Get("Paddle-Version") != "1" {
			t.Fatalf("paddle version=%q", request.Header.Get("Paddle-Version"))
		}
		var body struct {
			Items []struct {
				PriceID  string `json:"price_id"`
				Quantity int    `json:"quantity"`
			} `json:"items"`
			CollectionMode string         `json:"collection_mode"`
			CustomerID     string         `json:"customer_id"`
			CustomData     map[string]any `json:"custom_data"`
			Checkout       struct {
				URL string `json:"url"`
			} `json:"checkout"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Items) != 1 || body.Items[0].PriceID != "pri_pro" ||
			body.Items[0].Quantity != 1 || body.CollectionMode != "automatic" ||
			body.CustomData["account_id"] != "account-id" ||
			body.CustomerID != "ctm_returning" ||
			body.Checkout.URL != server.URL+"/checkout" {
			t.Fatalf("payload=%#v", body)
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusCreated)
		_, _ = response.Write([]byte(`{"data":{"id":"txn_test","checkout":{"url":"` + server.URL + `/checkout?_ptxn=txn_test"}}}`))
	}))
	defer server.Close()
	hosted := &Server{cfg: Config{
		AppOrigin: server.URL, PaddleAPIBaseURL: server.URL,
		PaddleAPIKey: "paddle-key", PaddleEssentialPriceID: "pri_essential", PaddleProPriceID: "pri_pro",
		PaddleWebhookSecret: "webhook-secret", PaddleClientToken: "test_token",
		PaddleHTTPClient: server.Client(),
	}}
	id, checkoutURL, err := hosted.createPaddleCheckout(t.Context(), "account-id", "pro", "ctm_returning")
	if err != nil || id != "txn_test" || !strings.Contains(checkoutURL, "_ptxn=txn_test") {
		t.Fatalf("id=%q url=%q err=%v", id, checkoutURL, err)
	}
}

func TestCreatePaddleCheckoutWorksWithoutExistingCustomer(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var body struct {
			CustomerID string `json:"customer_id"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.CustomerID != "" {
			t.Fatalf("new checkout must not carry a customer: %q", body.CustomerID)
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusCreated)
		_, _ = response.Write([]byte(`{"data":{"id":"txn_new","checkout":{"url":"` + server.URL + `/checkout?_ptxn=txn_new"}}}`))
	}))
	defer server.Close()
	hosted := &Server{cfg: Config{
		AppOrigin: server.URL, PaddleAPIBaseURL: server.URL,
		PaddleAPIKey: "paddle-key", PaddleEssentialPriceID: "pri_essential", PaddleProPriceID: "pri_pro",
		PaddleWebhookSecret: "webhook-secret", PaddleClientToken: "test_token",
		PaddleHTTPClient: server.Client(),
	}}
	id, checkoutURL, err := hosted.createPaddleCheckout(t.Context(), "account-id", "pro", "")
	if err != nil || id != "txn_new" || !strings.Contains(checkoutURL, "_ptxn=txn_new") {
		t.Fatalf("id=%q url=%q err=%v", id, checkoutURL, err)
	}
}

func TestCreatePaddleCheckoutSelectsEssentialServerSide(t *testing.T) {
	var selectedPrice, selectedPlan string
	var provider *httptest.Server
	provider = httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var body struct {
			Items []struct {
				PriceID string `json:"price_id"`
			} `json:"items"`
			CustomData map[string]any `json:"custom_data"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		selectedPrice = body.Items[0].PriceID
		selectedPlan, _ = body.CustomData["plan_id"].(string)
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusCreated)
		_, _ = response.Write([]byte(`{"data":{"id":"txn_essential","checkout":{"url":"` + provider.URL + `/checkout?_ptxn=txn_essential"}}}`))
	}))
	defer provider.Close()
	hosted := &Server{cfg: Config{
		AppOrigin: provider.URL, PaddleAPIBaseURL: provider.URL,
		PaddleAPIKey: "paddle-key", PaddleEssentialPriceID: "pri_essential", PaddleProPriceID: "pri_pro",
		PaddleWebhookSecret: "webhook-secret", PaddleClientToken: "test_token", PaddleHTTPClient: provider.Client(),
	}}
	if _, _, err := hosted.createPaddleCheckout(t.Context(), "account-id", "essential", ""); err != nil {
		t.Fatal(err)
	}
	if selectedPrice != "pri_essential" || selectedPlan != "essential" {
		t.Fatalf("selected price=%q plan=%q", selectedPrice, selectedPlan)
	}
}

func TestCreatePaddleCheckoutIgnoresNonCustomerReferences(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var body struct {
			CustomerID string `json:"customer_id"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.CustomerID != "" {
			t.Fatalf("invalid customer reference leaked into the request: %q", body.CustomerID)
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusCreated)
		_, _ = response.Write([]byte(`{"data":{"id":"txn_safe","checkout":{"url":"` + server.URL + `/checkout?_ptxn=txn_safe"}}}`))
	}))
	defer server.Close()
	hosted := &Server{cfg: Config{
		AppOrigin: server.URL, PaddleAPIBaseURL: server.URL,
		PaddleAPIKey: "paddle-key", PaddleEssentialPriceID: "pri_essential", PaddleProPriceID: "pri_pro",
		PaddleWebhookSecret: "webhook-secret", PaddleClientToken: "test_token",
		PaddleHTTPClient: server.Client(),
	}}
	if _, _, err := hosted.createPaddleCheckout(t.Context(), "account-id", "pro", "not-a-customer"); err != nil {
		t.Fatalf("checkout with no stored customer failed: %v", err)
	}
}

func TestProductionCheckoutRequiresExplicitCommerceApproval(t *testing.T) {
	t.Parallel()
	hosted := &Server{cfg: Config{
		Environment: "production", AppOrigin: "https://app.learnloom.blog",
		PaddleAPIBaseURL: "https://api.paddle.com", PaddleAPIKey: "paddle-key",
		PaddleEssentialPriceID: "pri_essential", PaddleProPriceID: "pri_pro", PaddleWebhookSecret: "webhook-secret",
		PaddleClientToken: "live_token",
	}}
	if hosted.paddleConfigured() {
		t.Fatal("production credentials alone enabled checkout")
	}
	hosted.cfg.PaidCommerceApproved = true
	if !hosted.paddleConfigured() {
		t.Fatal("explicitly approved production commerce remained disabled")
	}
}

func TestPaddleConfiguredRequiresClientToken(t *testing.T) {
	t.Parallel()
	hosted := &Server{cfg: Config{
		AppOrigin: "https://app.learnloom.blog", PaddleAPIBaseURL: "https://api.paddle.com",
		PaddleAPIKey: "paddle-key", PaddleEssentialPriceID: "pri_essential", PaddleProPriceID: "pri_pro",
		PaddleWebhookSecret: "webhook-secret",
	}}
	if hosted.paddleConfigured() {
		t.Fatal("billing was configured without a client token")
	}
	hosted.cfg.PaddleClientToken = "test_token"
	if !hosted.paddleConfigured() {
		t.Fatal("billing stayed disabled with the client token present")
	}
}

func TestBillingConfigEndpointExposesOnlyPublicFields(t *testing.T) {
	t.Parallel()
	hosted := &Server{cfg: Config{
		Environment: "staging", AppOrigin: "https://app.learnloom.blog",
		PaddleAPIBaseURL: "https://sandbox-api.paddle.com", PaddleAPIKey: "paddle-key",
		PaddleEssentialPriceID: "pri_essential", PaddleProPriceID: "pri_pro", PaddleWebhookSecret: "webhook-secret",
		PaddleClientToken: "test_pk_live-ish-secret",
	}}
	request := httptest.NewRequest(http.MethodGet, "/api/billing/config", nil)
	response := httptest.NewRecorder()
	hosted.handleBillingConfig(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status=%d cache=%q", response.Code, response.Header().Get("Cache-Control"))
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["commerceAvailable"] != true || body["environment"] != "sandbox" ||
		body["clientToken"] != "test_pk_live-ish-secret" {
		t.Fatalf("billing config payload=%#v", body)
	}
	for _, secret := range []string{"paddle-key", "webhook-secret", "pri_pro"} {
		if strings.Contains(response.Body.String(), secret) {
			t.Fatalf("billing config leaked secret %q", secret)
		}
	}
}

func TestBillingConfigEndpointOmitsTokenWhenUnavailable(t *testing.T) {
	t.Parallel()
	hosted := &Server{cfg: Config{Environment: "development", AppOrigin: "https://app.learnloom.test"}}
	request := httptest.NewRequest(http.MethodGet, "/api/billing/config", nil)
	response := httptest.NewRecorder()
	hosted.handleBillingConfig(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["commerceAvailable"] != false || body["environment"] != "sandbox" {
		t.Fatalf("unavailable billing config payload=%#v", body)
	}
	if _, present := body["clientToken"]; present {
		t.Fatalf("unavailable billing config leaked a client token: %#v", body)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/billing/config", nil)
	response = httptest.NewRecorder()
	hosted.handleBillingConfig(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST billing config status=%d", response.Code)
	}
}

func TestPaddleCheckoutURLPointsAtCheckoutPathWithTransaction(t *testing.T) {
	t.Parallel()
	hosted := &Server{cfg: Config{AppOrigin: "https://app.learnloom.blog"}}
	checkoutURL, err := hosted.paddleCheckoutURL("txn_test")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(checkoutURL)
	if err != nil || parsed.Path != "/checkout" ||
		parsed.Query().Get("_ptxn") != "txn_test" ||
		parsed.Host != "app.learnloom.blog" {
		t.Fatalf("checkout url=%q err=%v", checkoutURL, err)
	}
	if _, err := hosted.paddleCheckoutURL("not-a-transaction"); err == nil {
		t.Fatal("invalid transaction reference produced a checkout URL")
	}
}

func TestCreatePaddleCheckoutRejectsProviderRedirectToAnotherHost(t *testing.T) {
	provider := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusCreated)
		_, _ = response.Write([]byte(`{"data":{"id":"txn_test","checkout":{"url":"https://attacker.example/checkout"}}}`))
	}))
	defer provider.Close()
	hosted := &Server{cfg: Config{
		AppOrigin: "https://app.learnloom.blog", PaddleAPIBaseURL: provider.URL,
		PaddleAPIKey: "paddle-key", PaddleEssentialPriceID: "pri_essential", PaddleProPriceID: "pri_pro",
		PaddleWebhookSecret: "webhook-secret", PaddleClientToken: "test_client", PaddleHTTPClient: provider.Client(),
	}}
	if _, _, err := hosted.createPaddleCheckout(t.Context(), "account-id", "pro", ""); err == nil {
		t.Fatal("unapproved checkout host was accepted")
	}
}

func TestCreatePaddlePortalOnlyAcceptsPaddleHTTPSHost(t *testing.T) {
	provider := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/customers/ctm_test/portal-sessions" {
			t.Fatalf("path=%s", request.URL.Path)
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusCreated)
		_, _ = response.Write([]byte(`{"data":{"urls":{"general":{"overview":"https://customer-portal.paddle.com/session?token=temporary"}}}}`))
	}))
	defer provider.Close()
	hosted := &Server{cfg: Config{
		PaddleAPIBaseURL: provider.URL, PaddleAPIKey: "paddle-key",
		PaddleEssentialPriceID: "pri_essential", PaddleProPriceID: "pri_pro", PaddleWebhookSecret: "webhook-secret",
		PaddleClientToken: "test_client", PaddleHTTPClient: provider.Client(),
	}}
	portalURL, err := hosted.createPaddlePortal(t.Context(), "ctm_test")
	if err != nil || !strings.HasPrefix(portalURL, "https://customer-portal.paddle.com/") {
		t.Fatalf("url=%q err=%v", portalURL, err)
	}
}

func TestPaddleCompletedTransactionRequiresConfiguredProPrice(t *testing.T) {
	transaction := paddleTransactionData{}
	transaction.Items = append(transaction.Items, struct {
		Price struct {
			ID string `json:"id"`
		} `json:"price"`
	}{})
	transaction.Items[0].Price.ID = "pri_other"
	if paddleTransactionContainsPrice(transaction, "pri_pro") {
		t.Fatal("unrelated price was treated as Pro")
	}
	transaction.Items[0].Price.ID = "pri_pro"
	if !paddleTransactionContainsPrice(transaction, "pri_pro") {
		t.Fatal("configured Pro price was not recognized")
	}
	if paddleTransactionContainsPrice(transaction, "") {
		t.Fatal("empty configured price was recognized")
	}
}

func TestPaddleSubscriptionRequiresConfiguredProPrice(t *testing.T) {
	subscription := paddleSubscriptionData{}
	subscription.Items = append(subscription.Items, struct {
		Price struct {
			ID string `json:"id"`
		} `json:"price"`
	}{})
	subscription.Items[0].Price.ID = "pri_other"
	if paddleSubscriptionContainsPrice(subscription, "pri_pro") {
		t.Fatal("unrelated subscription price was treated as Pro")
	}
	subscription.Items[0].Price.ID = "pri_pro"
	if !paddleSubscriptionContainsPrice(subscription, "pri_pro") {
		t.Fatal("configured subscription Pro price was not recognized")
	}
	if paddleSubscriptionContainsPrice(subscription, "") {
		t.Fatal("empty subscription price was recognized")
	}
}
