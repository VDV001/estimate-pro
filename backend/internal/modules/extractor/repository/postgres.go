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
