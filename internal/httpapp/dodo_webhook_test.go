package httpapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"testing"
	"time"
)

func TestVerifyDodoSignatureChecksBodyAndFreshness(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	body := []byte(`{"type":"subscription.active"}`)
	secretBytes := []byte("01234567890123456789012345678901")
	secret := "whsec_" + base64.StdEncoding.EncodeToString(secretBytes)
	timestamp := strconv.FormatInt(now.Unix(), 10)
	mac := hmac.New(sha256.New, secretBytes)
	_, _ = mac.Write([]byte("msg_1." + timestamp + "."))
	_, _ = mac.Write(body)
	signature := "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !verifyDodoSignature(body, "msg_1", timestamp, signature, secret, now, 5*time.Minute) {
		t.Fatal("valid signature rejected")
	}
	if verifyDodoSignature([]byte(`{}`), "msg_1", timestamp, signature, secret, now, 5*time.Minute) {
		t.Fatal("tampered body accepted")
	}
	if verifyDodoSignature(body, "msg_1", timestamp, signature, secret, now.Add(6*time.Minute), 5*time.Minute) {
		t.Fatal("stale signature accepted")
	}
}

func TestDodoSubscriptionStatusDoesNotGrantFailedCheckout(t *testing.T) {
	if _, ok := dodoSubscriptionStatus("failed", "subscription.failed"); ok {
		t.Fatal("failed initial subscription changed entitlement")
	}
	for input, expected := range map[string]string{"active": "active", "on_hold": "past_due", "paused": "paused", "cancelled": "canceled", "expired": "canceled"} {
		if got, ok := dodoSubscriptionStatus(input, "subscription.updated"); !ok || got != expected {
			t.Fatalf("status %s => %s %t", input, got, ok)
		}
	}
}
