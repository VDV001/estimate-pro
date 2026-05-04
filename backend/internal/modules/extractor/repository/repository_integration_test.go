// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package repository_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/domain"
	"github.com/VDV001/estimate-pro/backend/internal/modules/extractor/repository"
	"github.com/VDV001/estimate-pro/backend/internal/testutil"
)

// extractorTestFixture sets up a postgres testcontainer plus the
// minimum upstream rows (user → workspace → project → document →
// document_version) that an extraction can reference. Returned IDs
// live as long as the *testing.T does.
type extractorTestFixture struct {
	pool      *pgxpool.Pool
	userID    string
	projectID string
	documentID         string
	documentVersionID  string
}

func newExtractorTestFixture(t *testing.T) *extractorTestFixture {
	t.Helper()
	pool := testutil.SetupPostgres(t)

	userID := insertTestUser(t, pool)
	wsID := insertTestWorkspace(t, pool, userID)
	projectID := insertTestProject(t, pool, wsID, userID)
	documentID := insertTestDocument(t, pool, projectID, userID)
	versionID := insertTestDocumentVersion(t, pool, documentID, userID)

	return &extractorTestFixture{
		pool:              pool,
		userID:            userID,
		projectID:         projectID,
		documentID:        documentID,
		documentVersionID: versionID,
	}
}

func insertTestUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.New().String()
	now := time.Now()
	_, err := pool.Exec(t.Context(),
		`INSERT INTO users (id, email, password_hash, name, preferred_locale, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, uuid.New().String()+"@test.com", "$2a$10$xx", "Extractor User", "ru", now, now)
	if err != nil {
		t.Fatalf("insertTestUser: %v", err)
	}
	return id
}

func insertTestWorkspace(t *testing.T, pool *pgxpool.Pool, ownerID string) string {
	t.Helper()
	id := uuid.New().String()
	_, err := pool.Exec(t.Context(),
		`INSERT INTO workspaces (id, name, owner_id, created_at) VALUES ($1,$2,$3,$4)`,
		id, "ws", ownerID, time.Now())
	if err != nil {
		t.Fatalf("insertTestWorkspace: %v", err)
	}
	return id
}

func insertTestProject(t *testing.T, pool *pgxpool.Pool, wsID, ownerID string) string {
	t.Helper()
	id := uuid.New().String()
	now := time.Now()
	_, err := pool.Exec(t.Context(),
		`INSERT INTO projects (id, workspace_id, name, status, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, wsID, "extractor-proj", "active", ownerID, now, now)
	if err != nil {
		t.Fatalf("insertTestProject: %v", err)
	}
	return id
}

func insertTestDocument(t *testing.T, pool *pgxpool.Pool, projID, userID string) string {
	t.Helper()
	id := uuid.New().String()
	_, err := pool.Exec(t.Context(),
		`INSERT INTO documents (id, project_id, title, uploaded_by, created_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		id, projID, "Test Spec", userID, time.Now())
	if err != nil {
		t.Fatalf("insertTestDocument: %v", err)
	}
	return id
}

func insertTestDocumentVersion(t *testing.T, pool *pgxpool.Pool, docID, userID string) string {
	t.Helper()
	id := uuid.New().String()
	_, err := pool.Exec(t.Context(),
		`INSERT INTO document_versions (id, document_id, version_number, file_key, file_type, file_size, parsed_status, uploaded_by, uploaded_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, docID, 1, "spec.pdf", "pdf", 1024, "pending", userID, time.Now())
	if err != nil {
		t.Fatalf("insertTestDocumentVersion: %v", err)
	}
	return id
}

func mustNewExtraction(t *testing.T, docID, versionID string) *domain.Extraction {
	t.Helper()
	ext, err := domain.NewExtraction(docID, versionID)
	if err != nil {
		t.Fatalf("NewExtraction: %v", err)
	}
	return ext
}

// ---------- Create + GetByID ----------

func TestPostgresExtractionRepository_CreateAndGetByID(t *testing.T) {
	fx := newExtractorTestFixture(t)
	repo := repository.NewPostgresExtractionRepository(fx.pool)
	ctx := t.Context()

	ext := mustNewExtraction(t, fx.documentID, fx.documentVersionID)

	if err := repo.Create(ctx, ext); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, ext.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != ext.ID {
		t.Errorf("ID=%q, want %q", got.ID, ext.ID)
	}
	if got.DocumentID != fx.documentID {
		t.Errorf("DocumentID=%q, want %q", got.DocumentID, fx.documentID)
	}
	if got.DocumentVersionID != fx.documentVersionID {
		t.Errorf("DocumentVersionID=%q, want %q", got.DocumentVersionID, fx.documentVersionID)
	}
	if got.Status != domain.StatusPending {
		t.Errorf("Status=%q, want %q", got.Status, domain.StatusPending)
	}
	if got.FailureReason != "" {
		t.Errorf("FailureReason=%q, want empty", got.FailureReason)
	}
	if len(got.Tasks) != 0 {
		t.Errorf("Tasks len=%d, want 0", len(got.Tasks))
	}
	if got.StartedAt != nil {
		t.Errorf("StartedAt=%v, want nil", *got.StartedAt)
	}
	if got.CompletedAt != nil {
		t.Errorf("CompletedAt=%v, want nil", *got.CompletedAt)
	}
	// Postgres truncates to microseconds; our domain timestamps come from time.Now()
	// at nanosecond precision, so compare with a microsecond tolerance.
	if d := got.CreatedAt.Sub(ext.CreatedAt); d > time.Microsecond || d < -time.Microsecond {
		t.Errorf("CreatedAt drift = %v, want within ±1µs", d)
	}
}

func TestPostgresExtractionRepository_GetByID_NotFound(t *testing.T) {
	fx := newExtractorTestFixture(t)
	repo := repository.NewPostgresExtractionRepository(fx.pool)

	_, err := repo.GetByID(t.Context(), uuid.New().String())
	if !errors.Is(err, domain.ErrExtractionNotFound) {
		t.Fatalf("err=%v, want errors.Is %v", err, domain.ErrExtractionNotFound)
	}
}

func TestPostgresExtractionRepository_Create_RejectsDuplicateActive(t *testing.T) {
	fx := newExtractorTestFixture(t)
	repo := repository.NewPostgresExtractionRepository(fx.pool)
	ctx := t.Context()

	first := mustNewExtraction(t, fx.documentID, fx.documentVersionID)
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	// Second pending extraction for the same (document, version) must fail —
	// this is the idempotency guarantee from the UNIQUE partial index.
	second := mustNewExtraction(t, fx.documentID, fx.documentVersionID)
	if err := repo.Create(ctx, second); err == nil {
		t.Fatal("Create: expected unique-violation error on duplicate active extraction, got nil")
	}
}
