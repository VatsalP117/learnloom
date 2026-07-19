package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateForRejectsIncompleteWebRole(t *testing.T) {
	cfg := Config{}
	err := cfg.ValidateFor("web")
	if err == nil {
		t.Fatal("expected incomplete web configuration to fail")
	}
	for _, required := range []string{
		"DATABASE_URL", "S3_BUCKET", "CLERK_SECRET_KEY", "CSRF_SECRET",
	} {
		if !strings.Contains(err.Error(), required) {
			t.Errorf("validation did not report %s: %v", required, err)
		}
	}
}

func TestValidateForRequiresHTTPSModel(t *testing.T) {
	cfg := validWorkerConfig()
	cfg.Model.BaseURL = "http://models.internal"
	if err := cfg.ValidateFor("worker"); err == nil ||
		!strings.Contains(err.Error(), "MODEL_BASE_URL") {
		t.Fatalf("expected HTTPS validation error, got %v", err)
	}
}

func TestValidateForAcceptsWorkerRole(t *testing.T) {
	cfg := validWorkerConfig()
	if err := cfg.ValidateFor("worker"); err != nil {
		t.Fatalf("valid worker config failed: %v", err)
	}
}

func TestValidateForRequiresEncryptedProductionDependencies(t *testing.T) {
	cfg := validWorkerConfig()
	cfg.Environment = "production"
	cfg.Database.URL = "postgres://database.example/learnloom?sslmode=disable"
	cfg.ObjectStore.Endpoint = "http://objects.example"
	err := cfg.ValidateFor("worker")
	if err == nil || !strings.Contains(err.Error(), "TLS in production") ||
		!strings.Contains(err.Error(), "S3_ENDPOINT") {
		t.Fatalf("expected encrypted dependency errors, got %v", err)
	}
}

func validWorkerConfig() Config {
	return Config{
		Environment: "development",
		Database:    Database{URL: "postgres://example", MaxConnections: 4},
		ObjectStore: ObjectStore{
			Bucket: "artifacts", AccessKeyID: "key", SecretAccessKey: "secret",
		},
		Model: Model{
			BaseURL: "https://api.example.com", APIKey: "model-secret",
			Retries: 2, MaxTokens: 1024,
		},
		Resend: Resend{APIKey: "resend-secret", From: "sender@example.com"},
		Worker: Worker{
			ClaimDuration: 5 * time.Minute, GlobalConcurrency: 2,
			AccountConcurrency: 1,
		},
	}
}
