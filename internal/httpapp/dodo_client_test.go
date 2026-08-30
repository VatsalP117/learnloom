package httpapp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateDodoCheckoutUsesServerOwnedProductAndMetadata(t *testing.T) {
	provider := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checkouts" || r.Header.Get("Authorization") != "Bearer dodo-key" {
			t.Fatalf("request=%s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body dodoCheckoutRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.ProductCart) != 1 || body.ProductCart[0].ProductID != "pdt_pro" || body.Metadata["account_id"] != "account-id" || body.Metadata["plan_id"] != "pro" || body.Customer == nil || body.Customer.CustomerID != "cus_returning" {
			t.Fatalf("payload=%#v", body)
		}
		_, _ = w.Write([]byte(`{"session_id":"cks_test","checkout_url":"https://test.checkout.dodopayments.com/session/cks_test"}`))
	}))
	defer provider.Close()
	s := &Server{cfg: Config{Environment: "staging", AppOrigin: "https://app.example.com", DodoAPIBaseURL: provider.URL, DodoAPIKey: "dodo-key", DodoWebhookSecret: "d2Vic2VjcmV0", DodoEssentialProductID: "pdt_essential", DodoProProductID: "pdt_pro", DodoHTTPClient: provider.Client()}}
	id, checkoutURL, err := s.createDodoCheckout(t.Context(), "account-id", "pro", "cus_returning")
	if err != nil || id != "cks_test" || !strings.Contains(checkoutURL, "cks_test") {
		t.Fatalf("id=%q url=%q err=%v", id, checkoutURL, err)
	}
}

func TestDodoCheckoutAndPortalHostsAreRestricted(t *testing.T) {
	s := &Server{cfg: Config{Environment: "staging"}}
	if !s.validDodoCheckoutURL("https://test.checkout.dodopayments.com/session/cks_ok") || s.validDodoCheckoutURL("https://evil.example/session/cks_bad") {
		t.Fatal("checkout host allowlist failed")
	}
	if url, err := s.dodoCheckoutURL("cks_reuse"); err != nil || url != "https://test.checkout.dodopayments.com/session/cks_reuse" {
		t.Fatalf("url=%q err=%v", url, err)
	}
}

func TestProductionDodoCheckoutRequiresApproval(t *testing.T) {
	s := &Server{cfg: Config{Environment: "production", DodoAPIBaseURL: "https://live.dodopayments.com", DodoAPIKey: "key", DodoWebhookSecret: "secret", DodoEssentialProductID: "pdt_e", DodoProProductID: "pdt_p"}}
	if s.dodoConfigured() {
		t.Fatal("credentials alone enabled production commerce")
	}
	s.cfg.PaidCommerceApproved = true
	if !s.dodoConfigured() {
		t.Fatal("approved complete configuration stayed disabled")
	}
}
