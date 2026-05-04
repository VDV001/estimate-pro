// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package repository hosts the postgres-backed implementations of
// the extractor module's persistence ports declared in usecase/.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
)

// PostgresExtractionRepository persists Extraction aggregates and
// their audit events against a Postgres pool. All multi-row writes
// run inside a single tx so partial state is impossible.
type PostgresExtractionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresExtractionRepository(pool *pgxpool.Pool) *PostgresExtractionRepository {
	return &PostgresExtractionRepository{pool: pool}
}

// Create inserts a fresh Extraction row. The UNIQUE partial index on
// (document_id, document_version_id) WHERE status IN
// ('pending','processing','completed') guarantees no duplicate
// active extraction can be created — Postgres surfaces the
// constraint violation as a wrapped error which the caller can
// inspect via the underlying pgconn.PgError if needed.
func (r *PostgresExtractionRepository) Create(ctx context.Context, ext *domain.Extraction) error {
	const query = `INSERT INTO extractions
		(id, document_id, document_version_id, status, failure_reason,
		 created_at, updated_at, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(ctx, query,
		ext.ID, ext.DocumentID, ext.DocumentVersionID,
		string(ext.Status), ext.FailureReason,
		ext.CreatedAt, ext.UpdatedAt, ext.StartedAt, ext.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("extractor.repo.Create: %w", err)
	}
	return nil
}

// GetByID hydrates a complete Extraction (header + tasks). Missing
// rows surface as ErrExtractionNotFound, never a raw pgx.ErrNoRows.
func (r *PostgresExtractionRepository) GetByID(ctx context.Context, id string) (*domain.Extraction, error) {
	const query = `SELECT id, document_id, document_version_id, status,
		failure_reason, created_at, updated_at, started_at, completed_at
		FROM extractions WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)

	var (
		ext         domain.Extraction
		statusStr   string
		startedAt   *time.Time
		completedAt *time.Time
	)
	err := row.Scan(
		&ext.ID, &ext.DocumentID, &ext.DocumentVersionID, &statusStr,
		&ext.FailureReason, &ext.CreatedAt, &ext.UpdatedAt,
		&startedAt, &completedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrExtractionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("extractor.repo.GetByID: %w", err)
	}
	ext.Status = domain.ExtractionStatus(statusStr)
	ext.StartedAt = startedAt
	ext.CompletedAt = completedAt

	tasks, err := r.loadTasks(ctx, ext.ID)
	if err != nil {
		return nil, err
	}
	ext.Tasks = tasks

	return &ext, nil
}

// UpdateStatus persists a transitioned Extraction together with its
// audit event in a single tx. Missing extraction rows surface as
// ErrExtractionNotFound (zero rows updated). Either both writes
// commit or neither does.
func (r *PostgresExtractionRepository) UpdateStatus(ctx context.Context, ext *domain.Extraction, ev *domain.ExtractionEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("extractor.repo.UpdateStatus: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cmd, err := tx.Exec(ctx,
		`UPDATE extractions
		 SET status = $2, failure_reason = $3, updated_at = $4,
		     started_at = $5, completed_at = $6
		 WHERE id = $1`,
		ext.ID, string(ext.Status), ext.FailureReason, ext.UpdatedAt,
		ext.StartedAt, ext.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("extractor.repo.UpdateStatus: update: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrExtractionNotFound
	}

	if ev != nil {
		if _, err := tx.Exec(ctx,
			`INSERT INTO extraction_events
			 (id, extraction_id, from_status, to_status, error_message, actor, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			ev.ID, ev.ExtractionID, string(ev.FromStatus), string(ev.ToStatus),
			ev.ErrorMessage, ev.Actor, ev.CreatedAt,
		); err != nil {
			return fmt.Errorf("extractor.repo.UpdateStatus: insert event: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("extractor.repo.UpdateStatus: commit: %w", err)
	}
	return nil
}

// GetActiveByDocumentVersion returns the (at most one) extraction
// matching the UNIQUE partial index from migration 009 — i.e. the
// active or completed extraction for the (document_id,
// document_version_id) pair. Failed and cancelled extractions are
// excluded from the index and therefore not returned, freeing the
// pair for a retry Create.
func (r *PostgresExtractionRepository) GetActiveByDocumentVersion(ctx context.Context, documentID, documentVersionID string) (*domain.Extraction, error) {
	const query = `SELECT id FROM extractions
		WHERE document_id = $1 AND document_version_id = $2
		  AND status IN ('pending', 'processing', 'completed')
		LIMIT 1`
	var id string
	err := r.pool.QueryRow(ctx, query, documentID, documentVersionID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrExtractionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("extractor.repo.GetActiveByDocumentVersion: %w", err)
	}
	return r.GetByID(ctx, id)
}

// GetEvents returns the audit trail for an Extraction in
// chronological order.
func (r *PostgresExtractionRepository) GetEvents(ctx context.Context, extractionID string) ([]*domain.ExtractionEvent, error) {
	const query = `SELECT id, extraction_id, from_status, to_status,
		error_message, actor, created_at
		FROM extraction_events
		WHERE extraction_id = $1
		ORDER BY created_at ASC, id ASC`
	rows, err := r.pool.Query(ctx, query, extractionID)
	if err != nil {
		return nil, fmt.Errorf("extractor.repo.GetEvents: %w", err)
	}
	defer rows.Close()

	var events []*domain.ExtractionEvent
	for rows.Next() {
		var (
			ev       domain.ExtractionEvent
			fromStr  string
			toStr    string
		)
		if err := rows.Scan(&ev.ID, &ev.ExtractionID, &fromStr, &toStr,
			&ev.ErrorMessage, &ev.Actor, &ev.CreatedAt); err != nil {
			return nil, fmt.Errorf("extractor.repo.GetEvents: scan: %w", err)
		}
		ev.FromStatus = domain.ExtractionStatus(fromStr)
		ev.ToStatus = domain.ExtractionStatus(toStr)
		events = append(events, &ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("extractor.repo.GetEvents: rows: %w", err)
	}
	return events, nil
}

// SaveTasks replaces the persisted task list for an extraction in a
// single tx — the previous rows are cleared first so SaveTasks is
// safe to call repeatedly (e.g. on retry). Storage row IDs are
// generated here; ExtractedTask is a value object without identity.
func (r *PostgresExtractionRepository) SaveTasks(ctx context.Context, extractionID string, tasks []domain.ExtractedTask) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("extractor.repo.SaveTasks: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`DELETE FROM extracted_tasks WHERE extraction_id = $1`, extractionID); err != nil {
		return fmt.Errorf("extractor.repo.SaveTasks: delete: %w", err)
	}

	for i, task := range tasks {
		if _, err := tx.Exec(ctx,
			`INSERT INTO extracted_tasks (id, extraction_id, name, estimate_hint, ordinal)
			 VALUES ($1, $2, $3, $4, $5)`,
			uuid.New().String(), extractionID, task.Name, task.EstimateHint, i,
		); err != nil {
			return fmt.Errorf("extractor.repo.SaveTasks: insert[%d]: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("extractor.repo.SaveTasks: commit: %w", err)
	}
	return nil
}

// ListByProject returns every extraction whose underlying document
// belongs to projectID, newest first. Useful for the project-detail
// page's extractions tab. Returns an empty slice (not nil-error) for
// projects with no extractions.
func (r *PostgresExtractionRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Extraction, error) {
	const idsQuery = `SELECT e.id FROM extractions e
		JOIN documents d ON d.id = e.document_id
		WHERE d.project_id = $1
		ORDER BY e.created_at DESC, e.id DESC`
	rows, err := r.pool.Query(ctx, idsQuery, projectID)
	if err != nil {
		return nil, fmt.Errorf("extractor.repo.ListByProject: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("extractor.repo.ListByProject: scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("extractor.repo.ListByProject: rows: %w", err)
	}

	out := make([]*domain.Extraction, 0, len(ids))
	for _, id := range ids {
		ext, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("extractor.repo.ListByProject: hydrate %s: %w", id, err)
		}
		out = append(out, ext)
	}
	return out, nil
}

func (r *PostgresExtractionRepository) loadTasks(ctx context.Context, extractionID string) ([]domain.ExtractedTask, error) {
	const query = `SELECT name, estimate_hint
		FROM extracted_tasks
		WHERE extraction_id = $1
		ORDER BY ordinal ASC`
	rows, err := r.pool.Query(ctx, query, extractionID)
	if err != nil {
		return nil, fmt.Errorf("extractor.repo.loadTasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.ExtractedTask
	for rows.Next() {
		var t domain.ExtractedTask
		if err := rows.Scan(&t.Name, &t.EstimateHint); err != nil {
			return nil, fmt.Errorf("extractor.repo.loadTasks: scan: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("extractor.repo.loadTasks: rows: %w", err)
	}
	return tasks, nil
}
