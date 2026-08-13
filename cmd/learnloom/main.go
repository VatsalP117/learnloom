package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/VatsalP117/learnloom/internal/artifact"
	"github.com/VatsalP117/learnloom/internal/config"
	"github.com/VatsalP117/learnloom/internal/delivery"
	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/dossier"
	"github.com/VatsalP117/learnloom/internal/execution"
	"github.com/VatsalP117/learnloom/internal/httpapp"
	"github.com/VatsalP117/learnloom/internal/source"
	"github.com/VatsalP117/learnloom/internal/store"
)

var buildReleaseVersion = "unknown"

func main() {
	installBuildReleaseVersion()
	role := ""
	if len(os.Args) == 2 {
		role = os.Args[1]
	}
	if role != "web" && role != "worker" && role != "migrate" {
		fmt.Fprintln(os.Stderr, "usage: learnloom <web|worker|migrate>")
		os.Exit(2)
	}
	cfg, err := config.LoadFor(role)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration: %v\n", err)
		os.Exit(1)
	}
	logger := newLogger(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()
	var runErr error
	switch role {
	case "migrate":
		runErr = runMigrate(ctx, cfg)
	case "web":
		runErr = runWeb(ctx, cfg, logger)
	case "worker":
		runErr = runWorker(ctx, cfg, logger)
	}
	if runErr != nil && !errors.Is(runErr, context.Canceled) &&
		!errors.Is(runErr, http.ErrServerClosed) {
		logger.Error("runtime stopped", "role", role, "error", runErr)
		os.Exit(1)
	}
}

func installBuildReleaseVersion() {
	if strings.TrimSpace(os.Getenv("LEARNLOOM_RELEASE_VERSION")) != "" ||
		strings.TrimSpace(buildReleaseVersion) == "" || buildReleaseVersion == "unknown" {
		return
	}
	_ = os.Setenv("LEARNLOOM_RELEASE_VERSION", buildReleaseVersion)
}

func runMigrate(ctx context.Context, cfg config.Config) error {
	database, err := openDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer database.Close()
	return database.Migrate(ctx)
}

func runWeb(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
) error {
	database, err := openDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := database.Ready(ctx); err != nil {
		return err
	}
	artifacts, err := openArtifacts(ctx, cfg)
	if err != nil {
		return err
	}
	static := os.DirFS(cfg.HTTP.StaticDirectory)
	if _, err := fs.Stat(static, "index.html"); err != nil {
		return fmt.Errorf("frontend build is unavailable at %s: %w", cfg.HTTP.StaticDirectory, err)
	}
	sourceValidator := source.New(source.Config{
		FeedTimeout:    8 * time.Second,
		MaxFeedBytes:   cfg.Limits.MaxFeedBytes,
		MaxConcurrency: 3,
		MaxRedirects:   3,
		URLPolicy:      database,
	})
	handler, err := httpapp.NewServer(
		httpapp.Config{
			Environment: cfg.Environment, ReleaseVersion: cfg.ReleaseVersion,
			RootDomain: cfg.HTTP.RootDomain, ApexOrigin: cfg.HTTP.ApexOrigin,
			AppOrigin: cfg.HTTP.AppOrigin, CSRFSecret: cfg.HTTP.CSRFSecret,
			ClerkSecretKey: cfg.Clerk.SecretKey, ClerkJWTKey: cfg.Clerk.JWTKey,
			ClerkWebhookSecret:       cfg.Clerk.WebhookSecret,
			ClerkFrontendOrigin:      cfg.Clerk.FrontendOrigin,
			PaddleWebhookSecret:      cfg.Paddle.WebhookSecret,
			PaddleAPIKey:             cfg.Paddle.APIKey,
			PaddleAPIBaseURL:         cfg.Paddle.APIBaseURL,
			PaddleProPriceID:         cfg.Paddle.ProPriceID,
			PaidCommerceApproved:     cfg.Paddle.CommerceApproved,
			MaxRequestBodyBytes:      cfg.Limits.RequestBodyBytes,
			MaxNewsletters:           cfg.Limits.MaxNewslettersPerAccount,
			DailyAccountLimit:        cfg.Worker.DailyAccountLimit,
			DailyModelBudgetMicroUSD: cfg.Worker.DailyModelBudgetMicroUSD,
			ModelReservationMicroUSD: cfg.Worker.ModelReservationMicroUSD,
			MaxDeliveryAttempts:      cfg.Worker.MaxDeliveryAttempts,
			ResendConfigured:         cfg.Resend.APIKey != "" && cfg.Resend.From != "",
			SourceDiscovery:          cfg.SourceIntelligence.DiscoveryEnabled,
			FeaturedSites:            cfg.HTTP.FeaturedSites,
			Static:                   static,
		},
		database,
		artifacts,
		[]httpapp.Readiness{database, artifacts},
		logger,
	)
	if err != nil {
		return err
	}
	handler.SetSourceValidator(func(
		ctx context.Context,
		definition domain.SourceDefinition,
	) ([]domain.SourceItem, []string, error) {
		return sourceValidator.Fetch(ctx, []domain.SourceDefinition{definition}, 3)
	})
	if cfg.SourceIntelligence.DiscoveryEnabled {
		previewSearcher, err := source.NewSearXNG(source.SearXNGConfig{
			BaseURL: cfg.SourceIntelligence.SearXNGBaseURL,
			Timeout: cfg.SourceIntelligence.SearXNGTimeout,
		})
		if err != nil {
			return err
		}
		handler.SetPortfolioPlanner(func(
			ctx context.Context,
			topic string,
		) (source.PortfolioPreview, error) {
			return source.PreviewPortfolio(
				ctx,
				previewSearcher,
				topic,
				cfg.SourceIntelligence.DiscoveryMaxQueries,
				cfg.SourceIntelligence.DiscoveryMaxCandidates,
				cfg.SourceIntelligence.DiscoveryMaxActive,
			)
		})
	}
	server := &http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	operations := &http.Server{
		Addr:              cfg.HTTP.MetricsAddress,
		Handler:           handler.OperationalHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	return runWebServers(ctx, server, operations, logger)
}

func runWebServers(
	ctx context.Context,
	public, operations *http.Server,
	logger *slog.Logger,
) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan error, 2)
	go func() {
		results <- runHTTPServer(runCtx, public, logger, "web")
	}()
	go func() {
		results <- runHTTPServer(runCtx, operations, logger, "web operations")
	}()
	first := <-results
	cancel()
	second := <-results
	for _, err := range []error{first, second} {
		if err != nil && !errors.Is(err, context.Canceled) &&
			!errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	return nil
}

func runWorker(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
) error {
	database, err := openDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := database.Ready(ctx); err != nil {
		return err
	}
	artifacts, err := openArtifacts(ctx, cfg)
	if err != nil {
		return err
	}
	model, err := dossier.NewOpenAIModel(dossier.ModelConfig{
		BaseURL: cfg.Model.BaseURL, APIKey: cfg.Model.APIKey,
		Model: cfg.Model.Name, StructuredOutput: cfg.Model.StructuredOutput,
		MaxTokens: cfg.Model.MaxTokens,
		Timeout:   cfg.Model.Timeout, Retries: cfg.Model.Retries,
		MaxConcurrency:                 cfg.Model.MaxConcurrency,
		InputMicroUSDPerMillionTokens:  cfg.Model.InputMicroUSDPerMillionTokens,
		OutputMicroUSDPerMillionTokens: cfg.Model.OutputMicroUSDPerMillionTokens,
	})
	if err != nil {
		return err
	}
	acquisition := source.New(source.Config{
		MaxFeedBytes:         cfg.Limits.MaxFeedBytes,
		MaxArticleBytes:      cfg.Limits.MaxArticleBytes,
		MaxArticleCharacters: cfg.Limits.MaxArticleCharacters,
		MaxConcurrency:       cfg.SourceIntelligence.MaxConcurrency,
		URLPolicy:            database,
	})
	producer, err := dossier.NewGenerator(
		acquisition,
		model,
		dossier.GenerationConfig{
			ModelName: cfg.Model.Name, MaxItems: 18,
			MaxItemCharacters:         cfg.Limits.MaxItemCharacters,
			MaxArticleCharacters:      cfg.Limits.MaxArticleCharacters,
			MaxIntermediateCharacters: cfg.Limits.MaxIntermediateCharacters,
			HistoryEntries:            cfg.Limits.HistoryEntries,
		},
	)
	if err != nil {
		return err
	}
	sourceSvc := source.NewService(
		database,
		acquisition,
		source.ServiceConfig{
			DiscoveryEnabled:       cfg.SourceIntelligence.DiscoveryEnabled,
			MinUsableItems:         cfg.SourceIntelligence.MinUsableItems,
			TargetUsableItems:      cfg.SourceIntelligence.TargetUsableItems,
			DiscoveryMaxQueries:    cfg.SourceIntelligence.DiscoveryMaxQueries,
			DiscoveryMaxCandidates: cfg.SourceIntelligence.DiscoveryMaxCandidates,
			DiscoveryMaxActive:     cfg.SourceIntelligence.DiscoveryMaxActive,
			MaxItems:               18,
			MaxItemCharacters:      cfg.Limits.MaxItemCharacters,
			RefreshInterval:        cfg.SourceIntelligence.RefreshInterval,
			DefaultMaxStaleAge:     cfg.SourceIntelligence.DefaultMaxStaleAge,
			MaxConcurrency:         cfg.SourceIntelligence.MaxConcurrency,
		},
	)
	if cfg.SourceIntelligence.DiscoveryEnabled {
		searcher, err := source.NewSearXNG(source.SearXNGConfig{
			BaseURL: cfg.SourceIntelligence.SearXNGBaseURL,
			Timeout: cfg.SourceIntelligence.SearXNGTimeout,
		})
		if err != nil {
			return err
		}
		sourceSvc.WithSearcher(searcher)
	}
	mailer, err := delivery.NewResend(delivery.Config{
		APIKey: cfg.Resend.APIKey, From: cfg.Resend.From,
		SubjectPrefix: cfg.Resend.SubjectPrefix,
	})
	if err != nil {
		return err
	}
	worker, err := execution.New(
		database,
		producer,
		artifacts,
		mailer,
		sourceSvc,
		execution.Config{
			PollInterval:             cfg.Worker.PollInterval,
			ClaimDuration:            cfg.Worker.ClaimDuration,
			MaxIssueAttempts:         cfg.Worker.MaxIssueAttempts,
			MaxDeliveryAttempts:      cfg.Worker.MaxDeliveryAttempts,
			AccountConcurrency:       cfg.Worker.AccountConcurrency,
			GlobalConcurrency:        cfg.Worker.GlobalConcurrency,
			IssueTimeout:             cfg.Worker.IssueTimeout,
			DailyAccountLimit:        cfg.Worker.DailyAccountLimit,
			DailyGlobalLimit:         cfg.Worker.DailyGlobalLimit,
			DailyModelBudgetMicroUSD: cfg.Worker.DailyModelBudgetMicroUSD,
			ModelReservationMicroUSD: cfg.Worker.ModelReservationMicroUSD,
			HistoryEntries:           cfg.Limits.HistoryEntries,
			RootDomain:               cfg.HTTP.RootDomain,
			AppOrigin:                cfg.HTTP.AppOrigin,
			AttemptContext: store.IssueAttemptContext{
				WorkerID:          workerIdentity(),
				DeploymentVersion: cfg.ReleaseVersion,
				ModelName:         cfg.Model.Name,
				PipelineVersion:   dossier.PipelineVersion,
			},
		},
		logger,
	)
	if err != nil {
		return err
	}
	metrics := workerMetricsServer(
		cfg.Worker.MetricsAddress,
		worker,
		[]httpapp.Readiness{database, artifacts, model},
	)
	metricsErrors := make(chan error, 1)
	go func() {
		logger.Info("worker metrics listening", "address", cfg.Worker.MetricsAddress)
		metricsErrors <- metrics.ListenAndServe()
	}()
	workerErrors := make(chan error, 1)
	executionCtx, cancelExecution := context.WithCancel(context.Background())
	defer cancelExecution()
	go func() { workerErrors <- worker.Run(executionCtx) }()
	select {
	case <-ctx.Done():
		logger.Info("worker draining active Claims")
		worker.BeginDrain()
		drainTimer := time.NewTimer(cfg.Worker.ClaimDuration + time.Minute)
		select {
		case err := <-workerErrors:
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
		case <-drainTimer.C:
			logger.Warn("worker drain deadline reached; releasing active Claims")
			cancelExecution()
			if err := <-workerErrors; err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
		}
		if !drainTimer.Stop() {
			select {
			case <-drainTimer.C:
			default:
			}
		}
	case err := <-workerErrors:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
	case err := <-metricsErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	cancelExecution()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return metrics.Shutdown(shutdownCtx)
}

func workerIdentity() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return "unknown"
	}
	return hostname
}

func openDatabase(ctx context.Context, cfg config.Config) (*store.Store, error) {
	return store.Open(ctx, store.Config{
		URL: cfg.Database.URL, MaxConnections: cfg.Database.MaxConnections,
		MinConnections:   cfg.Database.MinConnections,
		StatementTimeout: cfg.Database.StatementTimeout,
	})
}

func openArtifacts(ctx context.Context, cfg config.Config) (*artifact.Store, error) {
	return artifact.New(ctx, artifact.Config{
		Bucket: cfg.ObjectStore.Bucket, Region: cfg.ObjectStore.Region,
		Endpoint: cfg.ObjectStore.Endpoint, AccessKeyID: cfg.ObjectStore.AccessKeyID,
		SecretAccessKey: cfg.ObjectStore.SecretAccessKey,
		UsePathStyle:    cfg.ObjectStore.UsePathStyle,
		CacheBytes:      cfg.ObjectStore.CacheBytes,
	})
}

func runHTTPServer(
	ctx context.Context,
	server *http.Server,
	logger *slog.Logger,
	role string,
) error {
	errorsChannel := make(chan error, 1)
	go func() {
		logger.Info(role+" listening", "address", server.Addr)
		errorsChannel <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errorsChannel:
		return err
	}
}

func newLogger(level string) *slog.Logger {
	var parsed slog.Level
	if err := parsed.UnmarshalText([]byte(level)); err != nil {
		parsed = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parsed}))
}
