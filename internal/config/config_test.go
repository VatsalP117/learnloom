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

func TestLoadNormalizesFeaturedSiteUsernames(t *testing.T) {
	t.Setenv("FEATURED_SITE_USERNAMES", " Maya,ada,maya,  lin ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := []string{"maya", "ada", "lin"}
	if strings.Join(cfg.HTTP.FeaturedSites, ",") != strings.Join(want, ",") {
		t.Fatalf("FeaturedSites = %#v, want %#v", cfg.HTTP.FeaturedSites, want)
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
		ReleaseVersion:               "0123456789abcdef0123456789abcdef01234567",
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
	cfg.ReleaseVersion = "0123456789abcdef0123456789abcdef01234567"
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
	cfg.ReleaseVersion = "0123456789abcdef0123456789abcdef01234567"
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
	cfg.ReleaseVersion = "0123456789abcdef0123456789abcdef01234567"
	cfg.AllowInsecurePrivateServices = true
	cfg.Database.URL = "postgres://database.example/learnloom?sslmode=disable"
	cfg.ObjectStore.Endpoint = "http://objects.example"
	err := cfg.ValidateFor("worker")
	if err == nil || !strings.Contains(err.Error(), "TLS in production") ||
		!strings.Contains(err.Error(), "S3_ENDPOINT") {
		t.Fatalf("public dependencies must remain encrypted, got %v", err)
	}
}

func TestValidateForRequiresImmutableReleaseOutsideDevelopment(t *testing.T) {
	t.Parallel()
	for _, environment := range []string{"staging", "production"} {
		for _, release := range []string{"", "unknown", "main", "0123456", strings.Repeat("A", 40)} {
			t.Run(environment+"_"+release, func(t *testing.T) {
				cfg := validWorkerConfig()
				cfg.Environment = environment
				cfg.ReleaseVersion = release
				if environment == "production" {
					cfg.AllowInsecurePrivateServices = true
					cfg.Database.URL = "postgres://learnloom:secret@postgres:5432/learnloom?sslmode=disable"
				}
				err := cfg.ValidateFor("worker")
				if err == nil || !strings.Contains(err.Error(), "LEARNLOOM_RELEASE_VERSION") {
					t.Fatalf("environment=%s release=%q should fail: %v", environment, release, err)
				}
			})
		}
	}

	for _, release := range []string{
		"0123456789abcdef0123456789abcdef01234567",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	} {
		cfg := validWorkerConfig()
		cfg.Environment = "staging"
		cfg.ReleaseVersion = release
		if err := cfg.ValidateFor("worker"); err != nil {
			t.Fatalf("full release %q failed: %v", release, err)
		}
	}
}

func TestValidateForSeparatesSandboxTestingFromApprovedProductionCommerce(t *testing.T) {
	t.Parallel()
	configurePaddle := func(cfg *Config) {
		cfg.Paddle.APIKey = "paddle-key"
		cfg.Paddle.WebhookSecret = "webhook-secret"
		cfg.Paddle.ProPriceID = "pri_pro"
	}

	staging := validWorkerConfig()
	staging.Environment = "staging"
	staging.ReleaseVersion = strings.Repeat("a", 40)
	configurePaddle(&staging)
	staging.Paddle.APIBaseURL = "https://api.paddle.com"
	if err := staging.ValidateFor("worker"); err == nil || !strings.Contains(err.Error(), "sandbox-api.paddle.com") {
		t.Fatalf("staging live Paddle endpoint should fail: %v", err)
	}
	staging.Paddle.APIBaseURL = "https://sandbox-api.paddle.com"
	if err := staging.ValidateFor("worker"); err != nil {
		t.Fatalf("staging sandbox Paddle failed: %v", err)
	}

	production := validWorkerConfig()
	production.Environment = "production"
	production.ReleaseVersion = strings.Repeat("b", 40)
	production.AllowInsecurePrivateServices = true
	production.Database.URL = "postgres://learnloom:secret@postgres:5432/learnloom?sslmode=disable"
	configurePaddle(&production)
	production.Paddle.APIBaseURL = "https://api.paddle.com"
	if err := production.ValidateFor("worker"); err == nil || !strings.Contains(err.Error(), "PAID_COMMERCE_APPROVED") {
		t.Fatalf("unapproved production commerce should fail: %v", err)
	}
	production.Paddle.CommerceApproved = true
	production.Paddle.ApprovalReference = "legal/entity-tax-refund-review-01"
	if err := production.ValidateFor("worker"); err != nil {
		t.Fatalf("approved production commerce failed: %v", err)
	}
	production.Paddle.APIBaseURL = "https://sandbox-api.paddle.com"
	if err := production.ValidateFor("worker"); err == nil || !strings.Contains(err.Error(), "api.paddle.com in production") {
		t.Fatalf("production sandbox Paddle endpoint should fail: %v", err)
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
			InputMicroUSDPerMillionTokens:  1_000_000,
			OutputMicroUSDPerMillionTokens: 1_000_000,
		},
		Resend: Resend{APIKey: "resend-secret", From: "sender@example.com"},
		Worker: Worker{
			ClaimDuration: 5 * time.Minute, GlobalConcurrency: 2,
			AccountConcurrency: 1, DailyModelBudgetMicroUSD: 10_000_000,
			ModelReservationMicroUSD: 1_000_000,
		},
		SourceIntelligence: SourceIntelligence{
			MinUsableItems: 4, TargetUsableItems: 8,
			DiscoveryMaxQueries: 5, DiscoveryMaxCandidates: 30,
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

func TestValidateForSentryDSN(t *testing.T) {
	cfg := validWorkerConfig()
	if err := cfg.ValidateFor("worker"); err != nil {
		t.Fatalf("baseline worker config failed: %v", err)
	}
	cfg.Sentry.DSN = "not-a-url"
	if err := cfg.ValidateFor("worker"); err == nil ||
		!strings.Contains(err.Error(), "SENTRY_DSN") {
		t.Fatalf("expected SENTRY_DSN validation error, got %v", err)
	}
	cfg.Sentry.DSN = "https://example.sentry.io/1234567"
	if err := cfg.ValidateFor("worker"); err != nil {
		t.Fatalf("valid SENTRY_DSN failed: %v", err)
	}
}
