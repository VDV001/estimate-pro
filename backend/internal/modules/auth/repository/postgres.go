package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VDV001/estimate-pro/backend/internal/modules/auth/domain"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (id, email, password_hash, name, avatar_url, preferred_locale, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash, user.Name, user.AvatarURL, user.PreferredLocale, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("auth.Repository.Create: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, avatar_url, preferred_locale, COALESCE(telegram_chat_id, ''), COALESCE(notification_email, ''), created_at, updated_at FROM users WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)
	u := &domain.User{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AvatarURL, &u.PreferredLocale, &u.TelegramChatID, &u.NotificationEmail, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.GetByID: %w", err)
	}
	return u, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, avatar_url, preferred_locale, COALESCE(telegram_chat_id, ''), COALESCE(notification_email, ''), created_at, updated_at FROM users WHERE email = $1`
	row := r.pool.QueryRow(ctx, query, email)
	u := &domain.User{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AvatarURL, &u.PreferredLocale, &u.TelegramChatID, &u.NotificationEmail, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.GetByEmail: %w", err)
	}
	return u, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `UPDATE users SET email=$1, name=$2, avatar_url=$3, preferred_locale=$4, telegram_chat_id=$5, notification_email=$6, updated_at=$7 WHERE id=$8`
	_, err := r.pool.Exec(ctx, query, user.Email, user.Name, user.AvatarURL, user.PreferredLocale, nilIfEmpty(user.TelegramChatID), nilIfEmpty(user.NotificationEmail), user.UpdatedAt, user.ID)
	if err != nil {
		return fmt.Errorf("auth.Repository.Update: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) Search(ctx context.Context, query string, excludeUserID string, limit int) ([]*domain.UserSearchResult, error) {
	escaped := strings.NewReplacer("%", "\\%", "_", "\\_", "\\", "\\\\").Replace(query)
	pattern := "%" + escaped + "%"
	sql := `SELECT id, email, name, COALESCE(avatar_url, '') FROM users
		WHERE id != $1 AND (email ILIKE $2 OR name ILIKE $2)
		ORDER BY name LIMIT $3`
	rows, err := r.pool.Query(ctx, sql, excludeUserID, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.Search: %w", err)
	}
	defer rows.Close()

	var results []*domain.UserSearchResult
	for rows.Next() {
		u := &domain.UserSearchResult{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL); err != nil {
			return nil, fmt.Errorf("auth.Repository.Search scan: %w", err)
		}
		results = append(results, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auth.Repository.Search iteration: %w", err)
	}
	return results, nil
}

func (r *PostgresUserRepository) ListColleagues(ctx context.Context, userID string, limit int) ([]*domain.UserSearchResult, error) {
	sql := `SELECT DISTINCT u.id, u.email, u.name, COALESCE(u.avatar_url, '')
		FROM users u
		INNER JOIN project_members pm1 ON u.id = pm1.user_id
		INNER JOIN project_members pm2 ON pm1.project_id = pm2.project_id
		WHERE pm2.user_id = $1 AND u.id != $1
		ORDER BY u.name LIMIT $2`
	rows, err := r.pool.Query(ctx, sql, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.ListColleagues: %w", err)
	}
	defer rows.Close()

	var results []*domain.UserSearchResult
	for rows.Next() {
		u := &domain.UserSearchResult{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL); err != nil {
			return nil, fmt.Errorf("auth.Repository.ListColleagues scan: %w", err)
		}
		results = append(results, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auth.Repository.ListColleagues iteration: %w", err)
	}
	return results, nil
}

func (r *PostgresUserRepository) ListRecentlyAdded(ctx context.Context, addedByUserID string, limit int) ([]*domain.UserSearchResult, error) {
	sql := `SELECT DISTINCT ON (u.id) u.id, u.email, u.name, COALESCE(u.avatar_url, '')
		FROM users u
		INNER JOIN project_members pm ON u.id = pm.user_id
		WHERE pm.added_by = $1 AND u.id != $1
		ORDER BY u.id, pm.added_at DESC
		LIMIT $2`
	rows, err := r.pool.Query(ctx, sql, addedByUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.ListRecentlyAdded: %w", err)
	}
	defer rows.Close()

	var results []*domain.UserSearchResult
	for rows.Next() {
		u := &domain.UserSearchResult{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL); err != nil {
			return nil, fmt.Errorf("auth.Repository.ListRecentlyAdded scan: %w", err)
		}
		results = append(results, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auth.Repository.ListRecentlyAdded iteration: %w", err)
	}
	return results, nil
}
