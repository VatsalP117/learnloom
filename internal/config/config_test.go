package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadExpandsEscapedNewlinesInClerkJWTKey(t *testing.T) {
	t.Setenv("CLERK_JWT_KEY", `-----BEGIN PUBLIC KEY-----\nabc\n-----END PUBLIC KEY-----`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := "-----BEGIN PUBLIC KEY-----\nabc\n-----END PUBLIC KEY-----"
	if cfg.Clerk.JWTKey != want {
		t.Errorf("Clerk.JWTKey = %q, want %q", cfg.Clerk.JWTKey, want)
	}
}

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

func TestValidateForAllowsProductionWebWithoutStaticClerkJWTKey(t *testing.T) {
	cfg := Config{
		Environment:                  "production",
		AllowInsecurePrivateServices: true,
		Database: Database{
			URL:            "postgres://learnloom:secret@postgres:5432/learnloom?sslmode=disable",
			MaxConnections: 4,
		},
		ObjectStore: ObjectStore{
			Bucket: "artifacts", Endpoint: "http://minio:9000",
			AccessKeyID: "key", SecretAccessKey: "secret",
		},
		HTTP: HTTP{
			RootDomain: "learnloom.blog", AppOrigin: "https://app.learnloom.blog",
			CSRFSecret: "a-32-character-production-csrf-secret",
		},
		Clerk: Clerk{
			SecretKey: "sk_live_example", PublishableKey: "pk_live_example",
			WebhookSecret: "whsec_example", FrontendOrigin: "https://clerk.learnloom.blog",
		},
	}

	if err := cfg.ValidateFor("web"); err != nil {
		t.Fatalf("production web config without static Clerk JWT key failed: %v", err)
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

func TestValidateForAllowsExplicitPrivateProductionDependencies(t *testing.T) {
	cfg := validWorkerConfig()
	cfg.Environment = "production"
	cfg.AllowInsecurePrivateServices = true
	cfg.Database.URL = "postgres://learnloom:secret@postgres:5432/learnloom?sslmode=disable"
	cfg.ObjectStore.Endpoint = "http://minio:9000"
	if err := cfg.ValidateFor("worker"); err != nil {
		t.Fatalf("private production dependencies should be accepted with explicit opt-in: %v", err)
	}
}

func TestValidateForDoesNotRelaxPublicProductionDependencies(t *testing.T) {
	cfg := validWorkerConfig()
	cfg.Environment = "production"
	cfg.AllowInsecurePrivateServices = true
	cfg.Database.URL = "postgres://database.example/learnloom?sslmode=disable"
	cfg.ObjectStore.Endpoint = "http://objects.example"
	err := cfg.ValidateFor("worker")
	if err == nil || !strings.Contains(err.Error(), "TLS in production") ||
		!strings.Contains(err.Error(), "S3_ENDPOINT") {
		t.Fatalf("public dependencies must remain encrypted, got %v", err)
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
		SourceIntelligence: SourceIntelligence{
			MinUsableItems: 4, TargetUsableItems: 8,
			DiscoveryMaxQueries: 4, DiscoveryMaxCandidates: 30,
			DiscoveryMaxActive: 8, MaxConcurrency: 4,
			RefreshInterval:    12 * time.Hour,
			DefaultMaxStaleAge: 30 * 24 * time.Hour,
		},
	}
}

func TestValidateForRequiresSearXNGWhenDiscoveryEnabled(t *testing.T) {
	cfg := validWorkerConfig()
	cfg.SourceIntelligence.DiscoveryEnabled = true
	if err := cfg.ValidateFor("worker"); err == nil ||
		!strings.Contains(err.Error(), "SEARXNG_BASE_URL") {
		t.Fatalf("expected SearXNG validation error, got %v", err)
	}
	cfg.SourceIntelligence.SearXNGBaseURL = "http://searxng:8080"
	if err := cfg.ValidateFor("worker"); err != nil {
		t.Fatalf("valid internal SearXNG URL failed: %v", err)
	}
}
