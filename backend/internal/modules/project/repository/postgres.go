package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VDV001/estimate-pro/backend/internal/modules/project/domain"
)

// Project repository

type PostgresProjectRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresProjectRepository(pool *pgxpool.Pool) *PostgresProjectRepository {
	return &PostgresProjectRepository{pool: pool}
}

func (r *PostgresProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	query := `INSERT INTO projects (id, workspace_id, name, description, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query, project.ID, project.WorkspaceID, project.Name, project.Description, project.Status, project.CreatedBy, project.CreatedAt, project.UpdatedAt)
	if err != nil {
		return fmt.Errorf("project.Repository.Create: %w", err)
	}
	return nil
}

func (r *PostgresProjectRepository) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	query := `SELECT id, workspace_id, name, description, status, created_by, created_at, updated_at FROM projects WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	p := &domain.Project{}
	err := row.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Description, &p.Status, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("project.Repository.GetByID: %w", err)
	}
	return p, nil
}

func (r *PostgresProjectRepository) ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]*domain.Project, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects WHERE workspace_id = $1`, workspaceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("project.Repository.ListByWorkspace count: %w", err)
	}

	query := `SELECT id, workspace_id, name, description, status, created_by, created_at, updated_at
		FROM projects WHERE workspace_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, query, workspaceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("project.Repository.ListByWorkspace: %w", err)
	}
	defer rows.Close()

	var projects []*domain.Project
	for rows.Next() {
		p := &domain.Project{}
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Description, &p.Status, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("project.Repository.ListByWorkspace scan: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, total, nil
}

func (r *PostgresProjectRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Project, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM projects p INNER JOIN project_members pm ON p.id = pm.project_id WHERE pm.user_id = $1`,
		userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("project.Repository.ListByUser count: %w", err)
	}

	query := `SELECT p.id, p.workspace_id, p.name, p.description, p.status, p.created_by, p.created_at, p.updated_at
		FROM projects p
		INNER JOIN project_members pm ON p.id = pm.project_id
		WHERE pm.user_id = $1
		ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("project.Repository.ListByUser: %w", err)
	}
	defer rows.Close()

	var projects []*domain.Project
	for rows.Next() {
		p := &domain.Project{}
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Description, &p.Status, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("project.Repository.ListByUser scan: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, total, nil
}

func (r *PostgresProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	query := `UPDATE projects SET name=$1, description=$2, status=$3, updated_at=$4 WHERE id=$5`
	_, err := r.pool.Exec(ctx, query, project.Name, project.Description, project.Status, project.UpdatedAt, project.ID)
	if err != nil {
		return fmt.Errorf("project.Repository.Update: %w", err)
	}
	return nil
}

func (r *PostgresProjectRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("project.Repository.Delete: %w", err)
	}
	return nil
}

// Workspace repository

type PostgresWorkspaceRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresWorkspaceRepository(pool *pgxpool.Pool) *PostgresWorkspaceRepository {
	return &PostgresWorkspaceRepository{pool: pool}
}

func (r *PostgresWorkspaceRepository) Create(ctx context.Context, ws *domain.Workspace) error {
	query := `INSERT INTO workspaces (id, name, owner_id, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, ws.ID, ws.Name, ws.OwnerID, ws.CreatedAt)
	if err != nil {
		return fmt.Errorf("workspace.Repository.Create: %w", err)
	}
	memberQuery := `INSERT INTO workspace_members (workspace_id, user_id, role, invited_at, joined_at) VALUES ($1, $2, 'admin', $3, $3)`
	_, err = r.pool.Exec(ctx, memberQuery, ws.ID, ws.OwnerID, ws.CreatedAt)
	if err != nil {
		return fmt.Errorf("workspace.Repository.Create add owner: %w", err)
	}
	return nil
}

func (r *PostgresWorkspaceRepository) GetByID(ctx context.Context, id string) (*domain.Workspace, error) {
	query := `SELECT id, name, owner_id, created_at FROM workspaces WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	ws := &domain.Workspace{}
	err := row.Scan(&ws.ID, &ws.Name, &ws.OwnerID, &ws.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWorkspaceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("workspace.Repository.GetByID: %w", err)
	}
	return ws, nil
}

func (r *PostgresWorkspaceRepository) ListByUser(ctx context.Context, userID string) ([]*domain.Workspace, error) {
	query := `SELECT w.id, w.name, w.owner_id, w.created_at FROM workspaces w
		INNER JOIN workspace_members wm ON w.id = wm.workspace_id
		WHERE wm.user_id = $1 ORDER BY w.created_at`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("workspace.Repository.ListByUser: %w", err)
	}
	defer rows.Close()

	var workspaces []*domain.Workspace
	for rows.Next() {
		ws := &domain.Workspace{}
		if err := rows.Scan(&ws.ID, &ws.Name, &ws.OwnerID, &ws.CreatedAt); err != nil {
			return nil, fmt.Errorf("workspace.Repository.ListByUser scan: %w", err)
		}
		workspaces = append(workspaces, ws)
	}
	return workspaces, nil
}

func (r *PostgresWorkspaceRepository) Update(ctx context.Context, ws *domain.Workspace) error {
	query := `UPDATE workspaces SET name = $1 WHERE id = $2 AND owner_id = $3`
	tag, err := r.pool.Exec(ctx, query, ws.Name, ws.ID, ws.OwnerID)
	if err != nil {
		return fmt.Errorf("workspace.Repository.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrWorkspaceNotFound
	}
	return nil
}

// Member repository

type PostgresMemberRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMemberRepository(pool *pgxpool.Pool) *PostgresMemberRepository {
	return &PostgresMemberRepository{pool: pool}
}

func (r *PostgresMemberRepository) Add(ctx context.Context, member *domain.Member) error {
	query := `INSERT INTO project_members (project_id, user_id, role, added_by, added_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, query, member.ProjectID, member.UserID, member.Role, nilIfEmpty(member.AddedBy), member.AddedAt)
	if err != nil {
		return fmt.Errorf("member.Repository.Add: %w", err)
	}
	return nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (r *PostgresMemberRepository) Remove(ctx context.Context, projectID, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`, projectID, userID)
	if err != nil {
		return fmt.Errorf("member.Repository.Remove: %w", err)
	}
	return nil
}

func (r *PostgresMemberRepository) ListByProject(ctx context.Context, projectID string) ([]*domain.Member, error) {
	query := `SELECT project_id, user_id, role FROM project_members WHERE project_id = $1`
	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("member.Repository.ListByProject: %w", err)
	}
	defer rows.Close()

	var members []*domain.Member
	for rows.Next() {
		m := &domain.Member{}
		if err := rows.Scan(&m.ProjectID, &m.UserID, &m.Role); err != nil {
			return nil, fmt.Errorf("member.Repository.ListByProject scan: %w", err)
		}
		members = append(members, m)
	}
	return members, nil
}

func (r *PostgresMemberRepository) ListByProjectWithUsers(ctx context.Context, projectID string) ([]*domain.MemberWithUser, error) {
	query := `SELECT pm.project_id, pm.user_id, pm.role, u.name, u.email
		FROM project_members pm
		INNER JOIN users u ON u.id = pm.user_id
		WHERE pm.project_id = $1
		ORDER BY pm.role, u.name`
	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("member.Repository.ListByProjectWithUsers: %w", err)
	}
	defer rows.Close()

	var members []*domain.MemberWithUser
	for rows.Next() {
		m := &domain.MemberWithUser{}
		if err := rows.Scan(&m.ProjectID, &m.UserID, &m.Role, &m.UserName, &m.UserEmail); err != nil {
			return nil, fmt.Errorf("member.Repository.ListByProjectWithUsers scan: %w", err)
		}
		members = append(members, m)
	}
	return members, nil
}

func (r *PostgresMemberRepository) GetRole(ctx context.Context, projectID, userID string) (domain.Role, error) {
	var role domain.Role
	err := r.pool.QueryRow(ctx, `SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2`, projectID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrMemberNotFound
	}
	if err != nil {
		return "", fmt.Errorf("member.Repository.GetRole: %w", err)
	}
	return role, nil
}
