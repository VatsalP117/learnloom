package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const currentSchemaVersion = 1

var (
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrForbidden        = errors.New("forbidden")
	ErrQuotaExceeded    = errors.New("quota exceeded")
	ErrClaimLost        = errors.New("Issue Claim was lost")
	ErrGenerationPaused = errors.New("generation is paused")
)

type Config struct {
	URL              string
	MaxConnections   int32
	MinConnections   int32
	StatementTimeout time.Duration
}

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, cfg Config) (*Store, error) {
	if cfg.URL == "" {
		return nil, errors.New("database URL is required")
	}
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, errors.New("database URL is invalid")
	}
	if cfg.MaxConnections > 0 {
		poolConfig.MaxConns = cfg.MaxConnections
	}
	if cfg.MinConnections >= 0 {
		poolConfig.MinConns = cfg.MinConnections
	}
	statementTimeout := cfg.StatementTimeout
	if statementTimeout == 0 {
		statementTimeout = 15 * time.Second
	}
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(
			ctx,
			"SELECT set_config('statement_timeout', $1, false)",
			fmt.Sprintf("%dms", statementTimeout.Milliseconds()),
		)
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	store := &Store{pool: pool}
	if err := store.pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return store, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ready(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("database readiness: %w", err)
	}
	version, err := s.SchemaVersion(ctx)
	if err != nil {
		return err
	}
	if version != currentSchemaVersion {
		return fmt.Errorf(
			"database schema is version %d; expected %d",
			version,
			currentSchemaVersion,
		)
	}
	return nil
}

func rollback(tx pgx.Tx) {
	_ = tx.Rollback(context.Background())
}
