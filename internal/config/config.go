package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment                  string
	LogLevel                     string
	AllowInsecurePrivateServices bool
	HTTP                         HTTP
	Database                     Database
	ObjectStore                  ObjectStore
	Model                        Model
	Clerk                        Clerk
	Resend                       Resend
	Worker                       Worker
	Limits                       Limits
	SourceIntelligence           SourceIntelligence
}

type SourceIntelligence struct {
	DiscoveryEnabled       bool
	SearXNGBaseURL         string
	SearXNGTimeout         time.Duration
	DiscoveryMaxQueries    int
	DiscoveryMaxCandidates int
	DiscoveryMaxActive     int
	MinUsableItems         int
	TargetUsableItems      int
	RefreshInterval        time.Duration
	DefaultMaxStaleAge     time.Duration
}

type HTTP struct {
	Address         string
	RootDomain      string
	ApexOrigin      string
	AppOrigin       string
	CSRFSecret      string
	StaticDirectory string
	TrustedProxy    []net.IPNet
}

type Database struct {
	URL              string
	MaxConnections   int32
	MinConnections   int32
	StatementTimeout time.Duration
}

type ObjectStore struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

type Model struct {
	BaseURL        string
	APIKey         string
	Name           string
	Timeout        time.Duration
	Retries        int
	MaxTokens      int
	MaxConcurrency int
}

type Clerk struct {
	SecretKey      string
	PublishableKey string
	JWTKey         string
	WebhookSecret  string
	FrontendOrigin string
}

type Resend struct {
	APIKey        string
	From          string
	SubjectPrefix string
}

type Worker struct {
	PollInterval        time.Duration
	ClaimDuration       time.Duration
	MaxIssueAttempts    int
	MaxDeliveryAttempts int
	AccountConcurrency  int
	GlobalConcurrency   int
	DailyAccountLimit   int
	DailyGlobalLimit    int
	MetricsAddress      string
}

type Limits struct {
	MaxSources                int
	MaxFeedBytes              int64
	MaxArticleBytes           int64
	MaxArticleCharacters      int
	MaxItemCharacters         int
	MaxIntermediateCharacters int
	HistoryEntries            int
	MaxNewslettersPerAccount  int
	RequestBodyBytes          int64
}

func Load() (Config, error) {
	cfg := Config{
		Environment: env("LEARNLOOM_ENV", "development"),
		LogLevel:    env("LOG_LEVEL", "info"),
		AllowInsecurePrivateServices: envBool(
			"ALLOW_INSECURE_PRIVATE_SERVICES",
			false,
		),
		HTTP: HTTP{
			Address:         env("HTTP_ADDR", ":3000"),
			RootDomain:      strings.ToLower(env("LEARNLOOM_ROOT_DOMAIN", "learnloom.blog")),
			AppOrigin:       env("LEARNLOOM_APP_ORIGIN", "https://app.learnloom.blog"),
			CSRFSecret:      os.Getenv("CSRF_SECRET"),
			StaticDirectory: env("FRONTEND_DIR", "web/dist"),
		},
		Database: Database{
			URL:              os.Getenv("DATABASE_URL"),
			MaxConnections:   int32(envInt("DATABASE_MAX_CONNECTIONS", 20)),
			MinConnections:   int32(envInt("DATABASE_MIN_CONNECTIONS", 2)),
			StatementTimeout: envDuration("DATABASE_STATEMENT_TIMEOUT", 15*time.Second),
		},
		ObjectStore: ObjectStore{
			Bucket:          os.Getenv("S3_BUCKET"),
			Region:          env("S3_REGION", "us-east-1"),
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
			UsePathStyle:    envBool("S3_USE_PATH_STYLE", false),
		},
		Model: Model{
			BaseURL:        env("MODEL_BASE_URL", "https://api.deepseek.com"),
			APIKey:         os.Getenv("MODEL_API_KEY"),
			Name:           env("MODEL_NAME", "deepseek-chat"),
			Timeout:        envDuration("MODEL_TIMEOUT", 10*time.Minute),
			Retries:        envInt("MODEL_RETRIES", 2),
			MaxTokens:      envInt("MODEL_MAX_TOKENS", 8192),
			MaxConcurrency: envInt("MODEL_MAX_CONCURRENCY", 4),
		},
		Clerk: Clerk{
			SecretKey:      os.Getenv("CLERK_SECRET_KEY"),
			PublishableKey: os.Getenv("CLERK_PUBLISHABLE_KEY"),
			JWTKey:         strings.ReplaceAll(os.Getenv("CLERK_JWT_KEY"), `\n`, "\n"),
			WebhookSecret:  os.Getenv("CLERK_WEBHOOK_SECRET"),
			FrontendOrigin: os.Getenv("CLERK_FRONTEND_ORIGIN"),
		},
		Resend: Resend{
			APIKey:        os.Getenv("RESEND_API_KEY"),
			From:          os.Getenv("RESEND_FROM"),
			SubjectPrefix: env("RESEND_SUBJECT_PREFIX", "Learnloom"),
		},
		Worker: Worker{
			PollInterval:        envDuration("WORKER_POLL_INTERVAL", 2*time.Second),
			ClaimDuration:       envDuration("WORKER_CLAIM_DURATION", 15*time.Minute),
			MaxIssueAttempts:    envInt("WORKER_MAX_ISSUE_ATTEMPTS", 3),
			MaxDeliveryAttempts: envInt("WORKER_MAX_DELIVERY_ATTEMPTS", 6),
			AccountConcurrency:  envInt("ACCOUNT_GENERATION_CONCURRENCY", 1),
			GlobalConcurrency:   envInt("GLOBAL_GENERATION_CONCURRENCY", 4),
			DailyAccountLimit:   envInt("ACCOUNT_DAILY_GENERATION_LIMIT", 5),
			DailyGlobalLimit:    envInt("GLOBAL_DAILY_GENERATION_LIMIT", 1000),
			MetricsAddress:      env("WORKER_METRICS_ADDR", ":9090"),
		},
		Limits: Limits{
			MaxSources:                envInt("MAX_SOURCES_PER_NEWSLETTER", 12),
			MaxFeedBytes:              envInt64("MAX_FEED_BYTES", 2<<20),
			MaxArticleBytes:           envInt64("MAX_ARTICLE_BYTES", 512<<10),
			MaxArticleCharacters:      envInt("MAX_ARTICLE_CHARACTERS", 16_000),
			MaxItemCharacters:         envInt("MAX_ITEM_CHARACTERS", 1_800),
			MaxIntermediateCharacters: envInt("MAX_INTERMEDIATE_CHARACTERS", 24_000),
			HistoryEntries:            envInt("LEARNING_HISTORY_ENTRIES", 14),
			MaxNewslettersPerAccount:  envInt("MAX_NEWSLETTERS_PER_ACCOUNT", 10),
			RequestBodyBytes:          envInt64("MAX_REQUEST_BODY_BYTES", 1<<20),
		},
		SourceIntelligence: SourceIntelligence{
			DiscoveryEnabled:       envBool("SOURCE_DISCOVERY_ENABLED", false),
			SearXNGBaseURL:         os.Getenv("SEARXNG_BASE_URL"),
			SearXNGTimeout:         envDuration("SEARXNG_TIMEOUT", 8*time.Second),
			DiscoveryMaxQueries:    envInt("SOURCE_DISCOVERY_MAX_QUERIES", 4),
			DiscoveryMaxCandidates: envInt("SOURCE_DISCOVERY_MAX_CANDIDATES", 30),
			DiscoveryMaxActive:     envInt("SOURCE_DISCOVERY_MAX_ACTIVE", 8),
			MinUsableItems:         envInt("SOURCE_MIN_USABLE_ITEMS", 4),
			TargetUsableItems:      envInt("SOURCE_TARGET_USABLE_ITEMS", 8),
			RefreshInterval:        envDuration("SOURCE_REFRESH_INTERVAL", 12*time.Hour),
			DefaultMaxStaleAge:     envDuration("SOURCE_DEFAULT_MAX_STALE_AGE", 720*time.Hour),
		},
	}
	cfg.HTTP.ApexOrigin = "https://" + cfg.HTTP.RootDomain
	return cfg, nil
}

func LoadFor(role string) (Config, error) {
	cfg, err := Load()
	if err != nil {
		return Config{}, err
	}
	return cfg, cfg.ValidateFor(role)
}

func (c Config) ValidateFor(role string) error {
	var problems []error
	if c.Environment != "development" && c.Environment != "staging" &&
		c.Environment != "production" {
		problems = append(problems, errors.New("LEARNLOOM_ENV must be development, staging, or production"))
	}
	required := map[string]string{
		"DATABASE_URL": c.Database.URL,
	}
	switch role {
	case "web":
		required["S3_BUCKET"] = c.ObjectStore.Bucket
		required["CLERK_SECRET_KEY"] = c.Clerk.SecretKey
		required["CLERK_PUBLISHABLE_KEY"] = c.Clerk.PublishableKey
		required["CLERK_WEBHOOK_SECRET"] = c.Clerk.WebhookSecret
		required["CSRF_SECRET"] = c.HTTP.CSRFSecret
	case "worker":
		required["S3_BUCKET"] = c.ObjectStore.Bucket
		required["MODEL_API_KEY"] = c.Model.APIKey
		required["RESEND_API_KEY"] = c.Resend.APIKey
		required["RESEND_FROM"] = c.Resend.From
	case "migrate":
	default:
		problems = append(problems, fmt.Errorf("unknown runtime role %q", role))
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, fmt.Errorf("%s is required", name))
		}
	}
	if role == "web" && c.Environment == "production" && strings.TrimSpace(c.Clerk.JWTKey) == "" {
		problems = append(problems, errors.New("CLERK_JWT_KEY is required in production"))
	}
	if role == "web" && c.Environment == "production" && strings.TrimSpace(c.Clerk.FrontendOrigin) == "" {
		problems = append(problems, errors.New("CLERK_FRONTEND_ORIGIN is required in production"))
	}
	urls := map[string]string{}
	if role == "web" {
		urls["LEARNLOOM_APP_ORIGIN"] = c.HTTP.AppOrigin
	}
	if role == "worker" {
		urls["MODEL_BASE_URL"] = c.Model.BaseURL
	}
	for name, raw := range urls {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			problems = append(problems, fmt.Errorf("%s must be an HTTPS origin without credentials", name))
		}
	}
	databaseURL, databaseErr := url.Parse(c.Database.URL)
	if databaseErr != nil || databaseURL.Host == "" ||
		(databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql") {
		problems = append(problems, errors.New("DATABASE_URL must be a Postgres URL"))
	} else if c.Environment == "production" {
		sslMode := strings.ToLower(databaseURL.Query().Get("sslmode"))
		encrypted := sslMode == "require" || sslMode == "verify-ca" || sslMode == "verify-full"
		privateException := c.AllowInsecurePrivateServices && isPrivateServiceHost(databaseURL.Hostname())
		if !encrypted && !privateException {
			problems = append(problems, errors.New("DATABASE_URL must require TLS in production"))
		}
	}
	if c.Environment == "production" && c.ObjectStore.Endpoint != "" {
		endpoint, err := url.Parse(c.ObjectStore.Endpoint)
		privateException := err == nil && c.AllowInsecurePrivateServices &&
			endpoint.Scheme == "http" && isPrivateServiceHost(endpoint.Hostname())
		if err != nil || endpoint.Host == "" || endpoint.User != nil ||
			(endpoint.Scheme != "https" && !privateException) {
			problems = append(problems, errors.New("S3_ENDPOINT must use HTTPS in production"))
		}
	}
	if (c.ObjectStore.AccessKeyID == "") != (c.ObjectStore.SecretAccessKey == "") {
		problems = append(problems, errors.New(
			"S3_ACCESS_KEY_ID and S3_SECRET_ACCESS_KEY must be set together",
		))
	}
	if role == "web" {
		app, _ := url.Parse(c.HTTP.AppOrigin)
		if app != nil && app.Path != "" && app.Path != "/" {
			problems = append(problems, errors.New("LEARNLOOM_APP_ORIGIN must not contain a path"))
		}
		root := strings.TrimSpace(strings.ToLower(c.HTTP.RootDomain))
		if root == "" || strings.ContainsAny(root, `/:@`) ||
			strings.HasPrefix(root, ".") || strings.HasSuffix(root, ".") {
			problems = append(problems, errors.New("LEARNLOOM_ROOT_DOMAIN is invalid"))
		} else if app != nil && (!strings.EqualFold(app.Hostname(), "app."+root) ||
			(c.Environment == "production" && app.Port() != "")) {
			problems = append(problems, errors.New(
				"LEARNLOOM_APP_ORIGIN must use the app subdomain",
			))
		}
		if len(c.HTTP.CSRFSecret) < 32 {
			problems = append(problems, errors.New("CSRF_SECRET must contain at least 32 characters"))
		}
		if c.Environment == "production" {
			clerkOrigin, err := url.Parse(c.Clerk.FrontendOrigin)
			if err != nil || clerkOrigin.Scheme != "https" ||
				clerkOrigin.Host == "" || clerkOrigin.User != nil ||
				(clerkOrigin.Path != "" && clerkOrigin.Path != "/") {
				problems = append(problems, errors.New("CLERK_FRONTEND_ORIGIN must be an HTTPS origin"))
			}
		}
	}
	if c.Database.MaxConnections < 1 || c.Database.MinConnections < 0 ||
		c.Database.MinConnections > c.Database.MaxConnections {
		problems = append(problems, errors.New("database connection limits are invalid"))
	}
	if role == "worker" && (c.Model.Retries < 0 || c.Model.Retries > 5 || c.Model.MaxTokens < 256) {
		problems = append(problems, errors.New("model retry or token limits are invalid"))
	}
	if role == "worker" && (c.Worker.ClaimDuration < time.Minute || c.Worker.GlobalConcurrency < 1 ||
		c.Worker.AccountConcurrency < 1) {
		problems = append(problems, errors.New("worker limits are invalid"))
	}
	if role == "worker" {
		sourceCfg := c.SourceIntelligence
		if sourceCfg.MinUsableItems < 1 ||
			sourceCfg.TargetUsableItems < sourceCfg.MinUsableItems ||
			sourceCfg.DiscoveryMaxQueries < 1 ||
			sourceCfg.DiscoveryMaxCandidates < 1 ||
			sourceCfg.DiscoveryMaxActive < 1 ||
			sourceCfg.RefreshInterval <= 0 ||
			sourceCfg.DefaultMaxStaleAge <= 0 {
			problems = append(problems, errors.New("source intelligence limits are invalid"))
		}
		if sourceCfg.DiscoveryEnabled {
			parsed, err := url.Parse(sourceCfg.SearXNGBaseURL)
			if err != nil || parsed.Host == "" || parsed.User != nil ||
				(parsed.Scheme != "http" && parsed.Scheme != "https") {
				problems = append(problems, errors.New(
					"SEARXNG_BASE_URL must be an HTTP(S) origin without credentials when discovery is enabled",
				))
			}
		}
	}
	return errors.Join(problems...)
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(env(name, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return value
}

func envInt64(name string, fallback int64) int64 {
	value, err := strconv.ParseInt(env(name, strconv.FormatInt(fallback, 10)), 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func envBool(name string, fallback bool) bool {
	value, err := strconv.ParseBool(env(name, strconv.FormatBool(fallback)))
	if err != nil {
		return fallback
	}
	return value
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(env(name, fallback.String()))
	if err != nil {
		return fallback
	}
	return value
}

func isPrivateServiceHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return false
	}
	if host == "localhost" || (!strings.Contains(host, ".") && !strings.Contains(host, ":")) {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}
