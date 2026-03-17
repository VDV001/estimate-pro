package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/daniilrusanov/estimate-pro/backend/internal/modules/document/domain"
)

// Document repository

type PostgresDocumentRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresDocumentRepository(pool *pgxpool.Pool) *PostgresDocumentRepository {
	return &PostgresDocumentRepository{pool: pool}
}

func (r *PostgresDocumentRepository) Create(ctx context.Context, doc *domain.Document) error {
	query := `INSERT INTO documents (id, project_id, title, uploaded_by, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, query, doc.ID, doc.ProjectID, doc.Title, doc.UploadedBy, doc.CreatedAt)
	if err != nil {
		return fmt.Errorf("document.Repository.Create: %w", err)
	}
	return nil
}

func (r *PostgresDocumentRepository) GetByID(ctx context.Context, id string) (*domain.Document, error) {
	query := `SELECT id, project_id, title, uploaded_by, created_at FROM documents WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	d := &domain.Document{}
	err := row.Scan(&d.ID, &d.ProjectID, &d.Title, &d.UploadedBy, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrDocumentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("document.Repository.GetByID: %w", err)
	}
	return d, nil
}

func (r *PostgresDocumentRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Document, error) {
	query := `SELECT id, project_id, title, uploaded_by, created_at
		FROM documents WHERE project_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("document.Repository.ListByProject: %w", err)
	}
	defer rows.Close()

	var docs []*domain.Document
	for rows.Next() {
		d := &domain.Document{}
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.Title, &d.UploadedBy, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("document.Repository.ListByProject scan: %w", err)
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func (r *PostgresDocumentRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("document.Repository.Delete: %w", err)
	}
	return nil
}

// Version repository

type PostgresVersionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresVersionRepository(pool *pgxpool.Pool) *PostgresVersionRepository {
	return &PostgresVersionRepository{pool: pool}
}

func (r *PostgresVersionRepository) Create(ctx context.Context, v *domain.DocumentVersion) error {
	query := `INSERT INTO document_versions (id, document_id, version_number, file_key, file_type, file_size, parsed_status, confidence_score, uploaded_by, uploaded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.pool.Exec(ctx, query, v.ID, v.DocumentID, v.VersionNumber, v.FileKey, v.FileType, v.FileSize, v.ParsedStatus, v.ConfidenceScore, v.UploadedBy, v.UploadedAt)
	if err != nil {
		return fmt.Errorf("version.Repository.Create: %w", err)
	}
	return nil
}

func (r *PostgresVersionRepository) GetByID(ctx context.Context, id string) (*domain.DocumentVersion, error) {
	query := `SELECT id, document_id, version_number, file_key, file_type, file_size, parsed_status, confidence_score, uploaded_by, uploaded_at
		FROM document_versions WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	v := &domain.DocumentVersion{}
	err := row.Scan(&v.ID, &v.DocumentID, &v.VersionNumber, &v.FileKey, &v.FileType, &v.FileSize, &v.ParsedStatus, &v.ConfidenceScore, &v.UploadedBy, &v.UploadedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrVersionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("version.Repository.GetByID: %w", err)
	}
	return v, nil
}

func (r *PostgresVersionRepository) ListByDocument(ctx context.Context, documentID string) ([]*domain.DocumentVersion, error) {
	query := `SELECT id, document_id, version_number, file_key, file_type, file_size, parsed_status, confidence_score, uploaded_by, uploaded_at
		FROM document_versions WHERE document_id = $1 ORDER BY version_number DESC`
	rows, err := r.pool.Query(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("version.Repository.ListByDocument: %w", err)
	}
	defer rows.Close()

	var versions []*domain.DocumentVersion
	for rows.Next() {
		v := &domain.DocumentVersion{}
		if err := rows.Scan(&v.ID, &v.DocumentID, &v.VersionNumber, &v.FileKey, &v.FileType, &v.FileSize, &v.ParsedStatus, &v.ConfidenceScore, &v.UploadedBy, &v.UploadedAt); err != nil {
			return nil, fmt.Errorf("version.Repository.ListByDocument scan: %w", err)
		}
		versions = append(versions, v)
	}
	return versions, nil
}

func (r *PostgresVersionRepository) GetLatestByDocument(ctx context.Context, documentID string) (*domain.DocumentVersion, error) {
	query := `SELECT id, document_id, version_number, file_key, file_type, file_size, parsed_status, confidence_score, uploaded_by, uploaded_at
		FROM document_versions WHERE document_id = $1 ORDER BY version_number DESC LIMIT 1`
	row := r.pool.QueryRow(ctx, query, documentID)
	v := &domain.DocumentVersion{}
	err := row.Scan(&v.ID, &v.DocumentID, &v.VersionNumber, &v.FileKey, &v.FileType, &v.FileSize, &v.ParsedStatus, &v.ConfidenceScore, &v.UploadedBy, &v.UploadedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrVersionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("version.Repository.GetLatestByDocument: %w", err)
	}
	return v, nil
}
