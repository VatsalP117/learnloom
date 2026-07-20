package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListActiveSourceSpecs(ctx context.Context, newsletterID string) ([]domain.SourceSpec, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, newsletter_id::text, origin, state, display_name,
		       input_url, COALESCE(canonical_url, ''), scope,
		       COALESCE(kind, ''), item_limit,
		       COALESCE(discovery_reason, ''), COALESCE(rank_score, 0),
		       created_at, updated_at
		FROM source_specs
		WHERE newsletter_id = $1 AND state = 'active'
		ORDER BY origin DESC, created_at
	`, newsletterID)
	if err != nil {
		return nil, fmt.Errorf("list source specs: %w", err)
	}
	defer rows.Close()
	var specs []domain.SourceSpec
	for rows.Next() {
		var spec domain.SourceSpec
		var kindStr string
		if err := rows.Scan(
			&spec.ID, &spec.NewsletterID, &spec.Origin, &spec.State,
			&spec.DisplayName, &spec.InputURL, &spec.CanonicalURL,
			&spec.Scope, &kindStr, &spec.ItemLimit,
			&spec.DiscoveryReason, &spec.RankScore,
			&spec.CreatedAt, &spec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan source spec: %w", err)
		}
		if kindStr != "" {
			spec.Kind = domain.SourceKind(kindStr)
		}
		specs = append(specs, spec)
	}
	return specs, rows.Err()
}

func (s *Store) UpsertSourceEndpoint(ctx context.Context, endpoint domain.SourceEndpoint) (domain.SourceEndpoint, error) {
	var existingID string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO source_endpoints (
			id, source_spec_id, endpoint_url, canonical_url, kind,
			etag, last_modified, last_http_status, health,
			consecutive_failures, last_checked_at, last_success_at,
			last_changed_at, last_error, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $15)
		ON CONFLICT (source_spec_id, canonical_url) DO UPDATE SET
			endpoint_url = EXCLUDED.endpoint_url,
			kind = EXCLUDED.kind,
			etag = EXCLUDED.etag,
			last_modified = EXCLUDED.last_modified,
			last_http_status = EXCLUDED.last_http_status,
			health = EXCLUDED.health,
			consecutive_failures = EXCLUDED.consecutive_failures,
			last_checked_at = EXCLUDED.last_checked_at,
			last_success_at = EXCLUDED.last_success_at,
			last_changed_at = EXCLUDED.last_changed_at,
			last_error = EXCLUDED.last_error,
			updated_at = EXCLUDED.updated_at
		RETURNING id::text
	`, endpoint.ID, endpoint.SourceSpecID, endpoint.EndpointURL,
		endpoint.CanonicalURL, endpoint.Kind, endpoint.ETag,
		endpoint.LastModified, endpoint.LastHTTPStatus, endpoint.Health,
		endpoint.ConsecutiveFailures, endpoint.LastCheckedAt,
		endpoint.LastSuccessAt, endpoint.LastChangedAt, endpoint.LastError,
		endpoint.UpdatedAt).Scan(&existingID)
	if err != nil {
		return domain.SourceEndpoint{}, fmt.Errorf("upsert source endpoint: %w", err)
	}
	endpoint.ID = existingID
	return endpoint, nil
}

func (s *Store) GetSourceEndpoint(ctx context.Context, specID string) (domain.SourceEndpoint, bool, error) {
	var ep domain.SourceEndpoint
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, source_spec_id::text, endpoint_url, canonical_url, kind,
		       COALESCE(etag, ''), COALESCE(last_modified, ''), COALESCE(last_http_status, 0),
		       health, consecutive_failures, last_checked_at, last_success_at,
		       last_changed_at, COALESCE(last_error, ''), created_at, updated_at
		FROM source_endpoints
		WHERE source_spec_id = $1
		LIMIT 1
	`, specID).Scan(
		&ep.ID, &ep.SourceSpecID, &ep.EndpointURL, &ep.CanonicalURL, &ep.Kind,
		&ep.ETag, &ep.LastModified, &ep.LastHTTPStatus,
		&ep.Health, &ep.ConsecutiveFailures,
		&ep.LastCheckedAt, &ep.LastSuccessAt,
		&ep.LastChangedAt, &ep.LastError, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SourceEndpoint{}, false, nil
	}
	if err != nil {
		return domain.SourceEndpoint{}, false, fmt.Errorf("get source endpoint: %w", err)
	}
	return ep, true, nil
}

func (s *Store) InsertSourceSnapshot(ctx context.Context, snapshot domain.SourceSnapshot) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO source_snapshots (
			id, source_endpoint_id, item_key, title, canonical_url,
			author, published_at, content, content_source, content_sha256,
			metadata, fetched_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12)
		ON CONFLICT (source_endpoint_id, item_key, content_sha256) DO UPDATE SET
			id = source_snapshots.id
		RETURNING id::text
	`, snapshot.ID, snapshot.SourceEndpointID, snapshot.ItemKey, snapshot.Title,
		snapshot.CanonicalURL, nullString(snapshot.Author), snapshot.PublishedAt,
		snapshot.Content, snapshot.ContentSource, snapshot.ContentSHA256,
		snapshot.Metadata, snapshot.FetchedAt).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert source snapshot: %w", err)
	}
	return id, nil
}

func (s *Store) GetSourceSnapshots(ctx context.Context, endpointID string, limit int) ([]domain.SourceSnapshot, error) {
	if limit < 1 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (item_key)
			       id, source_endpoint_id, item_key, title, canonical_url,
			       author, published_at, content, content_source,
			       content_sha256, metadata, fetched_at
			FROM source_snapshots
			WHERE source_endpoint_id = $1
			ORDER BY item_key, fetched_at DESC
		)
		SELECT id::text, source_endpoint_id::text, item_key, title, canonical_url,
		       COALESCE(author, ''), published_at, content, content_source,
		       content_sha256, metadata::text, fetched_at
		FROM latest
		ORDER BY COALESCE(published_at, fetched_at) DESC
		LIMIT $2
	`, endpointID, limit)
	if err != nil {
		return nil, fmt.Errorf("get source snapshots: %w", err)
	}
	defer rows.Close()
	var snapshots []domain.SourceSnapshot
	for rows.Next() {
		var snapshot domain.SourceSnapshot
		if err := rows.Scan(
			&snapshot.ID, &snapshot.SourceEndpointID, &snapshot.ItemKey,
			&snapshot.Title, &snapshot.CanonicalURL, &snapshot.Author,
			&snapshot.PublishedAt, &snapshot.Content, &snapshot.ContentSource,
			&snapshot.ContentSHA256, &snapshot.Metadata, &snapshot.FetchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan source snapshot: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (s *Store) HasIssueSources(ctx context.Context, issueID string) (bool, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM issue_sources WHERE issue_id = $1)
	`, issueID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check issue sources: %w", err)
	}
	return exists, nil
}

func (s *Store) GetIssueSources(ctx context.Context, issueID string) ([]domain.SourceSnapshot, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ss.id::text, ss.source_endpoint_id::text, ss.item_key,
		       ss.title, ss.canonical_url,
		       COALESCE(ss.author, ''), ss.published_at, ss.content,
		       ss.content_source, ss.content_sha256, ss.metadata::text, ss.fetched_at
		FROM issue_sources isrc
		JOIN source_snapshots ss ON ss.id = isrc.source_snapshot_id
		WHERE isrc.issue_id = $1
		ORDER BY isrc.position
	`, issueID)
	if err != nil {
		return nil, fmt.Errorf("get issue sources: %w", err)
	}
	defer rows.Close()
	var snapshots []domain.SourceSnapshot
	for rows.Next() {
		var snapshot domain.SourceSnapshot
		if err := rows.Scan(
			&snapshot.ID, &snapshot.SourceEndpointID, &snapshot.ItemKey,
			&snapshot.Title, &snapshot.CanonicalURL, &snapshot.Author,
			&snapshot.PublishedAt, &snapshot.Content, &snapshot.ContentSource,
			&snapshot.ContentSHA256, &snapshot.Metadata, &snapshot.FetchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan issue source snapshot: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (s *Store) InsertIssueSources(ctx context.Context, issueID string, links []domain.IssueSource) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer rollback(tx)
	if err := tx.QueryRow(ctx, `
		SELECT id FROM issues WHERE id = $1 FOR UPDATE
	`, issueID).Scan(new(string)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrNotFound
		}
		return false, fmt.Errorf("lock Issue evidence: %w", err)
	}
	var frozen bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM issue_sources WHERE issue_id = $1)
	`, issueID).Scan(&frozen); err != nil {
		return false, fmt.Errorf("check frozen Issue evidence: %w", err)
	}
	if frozen {
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}
	for _, link := range links {
		if _, err := tx.Exec(ctx, `
			INSERT INTO issue_sources (issue_id, source_snapshot_id, position, created_at)
			VALUES ($1, $2, $3, $4)
		`, issueID, link.SourceSnapshotID, link.Position, link.CreatedAt); err != nil {
			return false, fmt.Errorf("insert issue source: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) GetSourceSummary(ctx context.Context, newsletterID string) (domain.SourceSummary, error) {
	var summary domain.SourceSummary
	var lastChecked *time.Time
	if err := s.pool.QueryRow(ctx, `
		SELECT
			COALESCE(count(DISTINCT ss.id) FILTER (WHERE ss.origin = 'provided'), 0)::int,
			COALESCE(count(DISTINCT ss.id) FILTER (WHERE ss.origin = 'discovered'), 0)::int,
			COALESCE(count(DISTINCT ss.id) FILTER (WHERE ss.state = 'active' AND (se.health IS NULL OR se.health IN ('unknown','healthy','stale'))), 0)::int,
			COALESCE(count(DISTINCT ss.id) FILTER (WHERE ss.state IN ('unhealthy', 'rejected') OR se.health IN ('failing','blocked')), 0)::int,
			max(se.last_checked_at)
		FROM source_specs ss
		LEFT JOIN source_endpoints se ON se.source_spec_id = ss.id
		WHERE ss.newsletter_id = $1
	`, newsletterID).Scan(
		&summary.Provided, &summary.Discovered,
		&summary.Healthy, &summary.NeedsAttention,
		&lastChecked,
	); err != nil {
		return domain.SourceSummary{}, fmt.Errorf("get source summary: %w", err)
	}
	summary.LastCheckedAt = lastChecked
	return summary, nil
}

func (s *Store) UpsertDiscoveredSourceSpec(
	ctx context.Context,
	spec domain.SourceSpec,
) (domain.SourceSpec, error) {
	if spec.ID == "" {
		spec.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	err := s.pool.QueryRow(ctx, `
		INSERT INTO source_specs (
			id, newsletter_id, origin, state, display_name, input_url,
			canonical_url, scope, kind, item_limit, discovery_reason,
			discovery_query, rank_score, created_at, updated_at
		)
		VALUES ($1, $2, 'discovered', $3, $4, $5, $6, $7, NULLIF($8, ''), $9, $10, $11, $12, $13, $13)
		ON CONFLICT (newsletter_id, canonical_url) WHERE canonical_url IS NOT NULL
		DO UPDATE SET updated_at = EXCLUDED.updated_at
		RETURNING id::text, newsletter_id::text, origin, state, display_name,
		          input_url, COALESCE(canonical_url, ''), scope,
		          COALESCE(kind, ''), item_limit, COALESCE(discovery_reason, ''),
		          COALESCE(discovery_query, ''), COALESCE(rank_score, 0),
		          created_at, updated_at
	`, spec.ID, spec.NewsletterID, spec.State, spec.DisplayName, spec.InputURL,
		spec.CanonicalURL, spec.Scope, spec.Kind, spec.ItemLimit,
		nullString(spec.DiscoveryReason), nullString(spec.DiscoveryQuery),
		spec.RankScore, now).Scan(
		&spec.ID, &spec.NewsletterID, &spec.Origin, &spec.State,
		&spec.DisplayName, &spec.InputURL, &spec.CanonicalURL, &spec.Scope,
		&spec.Kind, &spec.ItemLimit, &spec.DiscoveryReason,
		&spec.DiscoveryQuery, &spec.RankScore, &spec.CreatedAt, &spec.UpdatedAt,
	)
	if err != nil {
		return domain.SourceSpec{}, fmt.Errorf("upsert discovered source spec: %w", err)
	}
	return spec, nil
}

func (s *Store) SetSourceSpecState(
	ctx context.Context,
	specID string,
	state domain.SourceState,
	kind domain.SourceKind,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE source_specs
		SET state = $2, kind = COALESCE(NULLIF($3, ''), kind), updated_at = now()
		WHERE id = $1
	`, specID, state, kind)
	if err != nil {
		return fmt.Errorf("set source spec state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateDiscoveryRun(ctx context.Context, run domain.DiscoveryRun) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO discovery_runs (
			id, newsletter_id, issue_id, reason, state, query_bundle,
			returned_candidates, rejected_candidates, resolved_candidates,
			activated_candidates, error, started_at, completed_at
		)
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5, $6::jsonb, $7, $8, $9, $10, NULLIF($11, ''), $12, $13)
	`, run.ID, run.NewsletterID, run.IssueID, run.Reason, run.State,
		run.QueryBundle, run.ReturnedCandidates, run.RejectedCandidates,
		run.ResolvedCandidates, run.ActivatedCandidates, run.Error,
		run.StartedAt, run.CompletedAt)
	if err != nil {
		return fmt.Errorf("create discovery run: %w", err)
	}
	return nil
}

func (s *Store) CompleteDiscoveryRun(ctx context.Context, run domain.DiscoveryRun) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE discovery_runs
		SET state = $2, returned_candidates = $3, rejected_candidates = $4,
		    resolved_candidates = $5, activated_candidates = $6,
		    error = NULLIF($7, ''), completed_at = $8
		WHERE id = $1
	`, run.ID, run.State, run.ReturnedCandidates, run.RejectedCandidates,
		run.ResolvedCandidates, run.ActivatedCandidates, run.Error, run.CompletedAt)
	if err != nil {
		return fmt.Errorf("complete discovery run: %w", err)
	}
	return nil
}

func (s *Store) ListSourceCatalog(
	ctx context.Context,
	accountID, newsletterID string,
	limit int,
) ([]domain.SourceCatalogItem, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT ss.id::text, ss.display_name, COALESCE(ss.canonical_url, ss.input_url),
		       ss.origin, ss.scope, COALESCE(ss.kind, ''), ss.state,
		       CASE
		         WHEN ss.state = 'unhealthy' THEN 'failing'
		         WHEN ss.state = 'rejected' THEN 'blocked'
		         ELSE COALESCE(se.health, 'unknown')
		       END,
		       COALESCE(ss.discovery_reason, ''),
		       se.last_checked_at, se.last_success_at,
		       CASE
		         WHEN ss.state IN ('unhealthy', 'rejected')
		           OR se.health IN ('failing', 'blocked')
		         THEN 'Source could not be refreshed.'
		         ELSE ''
		       END
		FROM source_specs ss
		JOIN newsletters n ON n.id = ss.newsletter_id
		LEFT JOIN LATERAL (
			SELECT health, last_checked_at, last_success_at, last_error
			FROM source_endpoints
			WHERE source_spec_id = ss.id
			ORDER BY updated_at DESC
			LIMIT 1
		) se ON true
		WHERE ss.newsletter_id = $1 AND n.owner_account_id = $2
		ORDER BY CASE ss.origin WHEN 'provided' THEN 0 ELSE 1 END, ss.created_at
		LIMIT $3
	`, newsletterID, accountID, limit)
	if err != nil {
		return nil, fmt.Errorf("list source catalog: %w", err)
	}
	defer rows.Close()
	var result []domain.SourceCatalogItem
	for rows.Next() {
		var item domain.SourceCatalogItem
		if err := rows.Scan(
			&item.ID, &item.DisplayName, &item.CanonicalURL, &item.Origin,
			&item.Scope, &item.Kind, &item.State, &item.Health,
			&item.DiscoveryReason, &item.LastCheckedAt,
			&item.LastSuccessfulAt, &item.Error,
		); err != nil {
			return nil, fmt.Errorf("scan source catalog: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func nullString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
