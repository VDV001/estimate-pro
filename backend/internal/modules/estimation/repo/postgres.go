package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VDV001/estimate-pro/backend/internal/modules/estimation/domain"
)

// EstimationRepository

type PostgresEstimationRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresEstimationRepository(pool *pgxpool.Pool) *PostgresEstimationRepository {
	return &PostgresEstimationRepository{pool: pool}
}

func (r *PostgresEstimationRepository) Create(ctx context.Context, est *domain.Estimation) error {
	query := `INSERT INTO estimations (id, project_id, document_version_id, submitted_by, status, submitted_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	var docVersionID *string
	if est.DocumentVersionID != "" {
		docVersionID = &est.DocumentVersionID
	}

	var submittedAt *time.Time
	if !est.SubmittedAt.IsZero() {
		submittedAt = &est.SubmittedAt
	}

	_, err := r.pool.Exec(ctx, query, est.ID, est.ProjectID, docVersionID, est.SubmittedBy, est.Status, submittedAt, est.CreatedAt)
	if err != nil {
		return fmt.Errorf("estimation.Repository.Create: %w", err)
	}
	return nil
}

func (r *PostgresEstimationRepository) GetByID(ctx context.Context, id string) (*domain.Estimation, error) {
	query := `SELECT id, project_id, document_version_id, submitted_by, status, submitted_at, created_at
		FROM estimations WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	e := &domain.Estimation{}
	var docVersionID *string
	var submittedAt *time.Time
	err := row.Scan(&e.ID, &e.ProjectID, &docVersionID, &e.SubmittedBy, &e.Status, &submittedAt, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrEstimationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("estimation.Repository.GetByID: %w", err)
	}
	if docVersionID != nil {
		e.DocumentVersionID = *docVersionID
	}
	if submittedAt != nil {
		e.SubmittedAt = *submittedAt
	}
	return e, nil
}

func (r *PostgresEstimationRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Estimation, error) {
	query := `SELECT id, project_id, document_version_id, submitted_by, status, submitted_at, created_at
		FROM estimations WHERE project_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("estimation.Repository.ListByProject: %w", err)
	}
	defer rows.Close()

	var estimations []*domain.Estimation
	for rows.Next() {
		e := &domain.Estimation{}
		var docVersionID *string
		var submittedAt *time.Time
		if err := rows.Scan(&e.ID, &e.ProjectID, &docVersionID, &e.SubmittedBy, &e.Status, &submittedAt, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("estimation.Repository.ListByProject scan: %w", err)
		}
		if docVersionID != nil {
			e.DocumentVersionID = *docVersionID
		}
		if submittedAt != nil {
			e.SubmittedAt = *submittedAt
		}
		estimations = append(estimations, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("estimation.Repository.ListByProject iteration: %w", err)
	}
	return estimations, nil
}

func (r *PostgresEstimationRepository) UpdateStatus(ctx context.Context, id string, status domain.Status) error {
	var err error
	var tag pgconn.CommandTag

	if status == domain.StatusSubmitted {
		tag, err = r.pool.Exec(ctx, `UPDATE estimations SET status = $2, submitted_at = NOW() WHERE id = $1`, id, string(status))
	} else {
		tag, err = r.pool.Exec(ctx, `UPDATE estimations SET status = $2 WHERE id = $1`, id, string(status))
	}

	if err != nil {
		return fmt.Errorf("estimation.Repository.UpdateStatus: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEstimationNotFound
	}
	return nil
}

func (r *PostgresEstimationRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM estimations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("estimation.Repository.Delete: %w", err)
	}
	return nil
}

// ItemRepository

type PostgresItemRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresItemRepository(pool *pgxpool.Pool) *PostgresItemRepository {
	return &PostgresItemRepository{pool: pool}
}

func (r *PostgresItemRepository) CreateBatch(ctx context.Context, items []*domain.EstimationItem) error {
	if len(items) == 0 {
		return nil
	}

	query := `INSERT INTO estimation_items (id, estimation_id, task_name, min_hours, likely_hours, max_hours, sort_order, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	batch := &pgx.Batch{}
	for _, item := range items {
		batch.Queue(query, item.ID, item.EstimationID, item.TaskName, item.MinHours, item.LikelyHours, item.MaxHours, item.SortOrder, item.Note)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range items {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("estimation.ItemRepository.CreateBatch: %w", err)
		}
	}
	return nil
}

func (r *PostgresItemRepository) ListByEstimation(ctx context.Context, estimationID string) ([]*domain.EstimationItem, error) {
	query := `SELECT id, estimation_id, task_name, min_hours, likely_hours, max_hours, sort_order, note
		FROM estimation_items WHERE estimation_id = $1 ORDER BY sort_order`
	rows, err := r.pool.Query(ctx, query, estimationID)
	if err != nil {
		return nil, fmt.Errorf("estimation.ItemRepository.ListByEstimation: %w", err)
	}
	defer rows.Close()

	var items []*domain.EstimationItem
	for rows.Next() {
		item := &domain.EstimationItem{}
		if err := rows.Scan(&item.ID, &item.EstimationID, &item.TaskName, &item.MinHours, &item.LikelyHours, &item.MaxHours, &item.SortOrder, &item.Note); err != nil {
			return nil, fmt.Errorf("estimation.ItemRepository.ListByEstimation scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("estimation.ItemRepository.ListByEstimation iteration: %w", err)
	}
	return items, nil
}

func (r *PostgresItemRepository) DeleteByEstimation(ctx context.Context, estimationID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM estimation_items WHERE estimation_id = $1`, estimationID)
	if err != nil {
		return fmt.Errorf("estimation.ItemRepository.DeleteByEstimation: %w", err)
	}
	return nil
}
