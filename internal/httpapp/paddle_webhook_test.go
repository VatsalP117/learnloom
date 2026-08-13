package httpapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

func TestVerifyPaddleSignatureChecksBodyAndFreshness(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	body := []byte(`{"event_id":"evt_1"}`)
	header := signedPaddleHeader(body, "secret", now.Unix())
	if !verifyPaddleSignature(body, header, "secret", now, 5*time.Minute) {
		t.Fatal("valid Paddle signature was rejected")
	}
	if verifyPaddleSignature([]byte(`{}`), header, "secret", now, 5*time.Minute) {
		t.Fatal("tampered Paddle body was accepted")
	}
	if verifyPaddleSignature(body, header, "secret", now.Add(6*time.Minute), 5*time.Minute) {
		t.Fatal("stale Paddle signature was accepted")
	}
}

func signedPaddleHeader(body []byte, secret string, timestamp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strconv.FormatInt(timestamp, 10) + ":"))
	_, _ = mac.Write(body)
	return "ts=" + strconv.FormatInt(timestamp, 10) + ";h1=" + hex.EncodeToString(mac.Sum(nil))
}
