package httpapp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
			body.Checkout.URL != server.URL+"/settings?checkout=complete" {
			t.Fatalf("payload=%#v", body)
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusCreated)
		_, _ = response.Write([]byte(`{"data":{"id":"txn_test","checkout":{"url":"` + server.URL + `/settings?checkout=complete&_ptxn=txn_test"}}}`))
	}))
	defer server.Close()
	hosted := &Server{cfg: Config{
		AppOrigin: server.URL, PaddleAPIBaseURL: server.URL,
		PaddleAPIKey: "paddle-key", PaddleProPriceID: "pri_pro",
		PaddleWebhookSecret: "webhook-secret", PaddleHTTPClient: server.Client(),
	}}
	id, checkoutURL, err := hosted.createPaddleCheckout(t.Context(), "account-id")
	if err != nil || id != "txn_test" || !strings.Contains(checkoutURL, "_ptxn=txn_test") {
		t.Fatalf("id=%q url=%q err=%v", id, checkoutURL, err)
	}
}

func TestProductionCheckoutRequiresExplicitCommerceApproval(t *testing.T) {
	t.Parallel()
	hosted := &Server{cfg: Config{
		Environment: "production", AppOrigin: "https://app.learnloom.blog",
		PaddleAPIBaseURL: "https://api.paddle.com", PaddleAPIKey: "paddle-key",
		PaddleProPriceID: "pri_pro", PaddleWebhookSecret: "webhook-secret",
	}}
	if hosted.paddleConfigured() {
		t.Fatal("production credentials alone enabled checkout")
	}
	hosted.cfg.PaidCommerceApproved = true
	if !hosted.paddleConfigured() {
		t.Fatal("explicitly approved production commerce remained disabled")
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
		PaddleAPIKey: "paddle-key", PaddleProPriceID: "pri_pro",
		PaddleWebhookSecret: "webhook-secret", PaddleHTTPClient: provider.Client(),
	}}
	if _, _, err := hosted.createPaddleCheckout(t.Context(), "account-id"); err == nil {
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
		PaddleProPriceID: "pri_pro", PaddleWebhookSecret: "webhook-secret",
		PaddleHTTPClient: provider.Client(),
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
