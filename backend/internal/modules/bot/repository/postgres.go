// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VDV001/estimate-pro/backend/internal/modules/bot/domain"
)

// --- PostgresSessionRepository ---

// PostgresSessionRepository implements domain.SessionRepository using PostgreSQL.
type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSessionRepository creates a new PostgresSessionRepository.
func NewPostgresSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

func (r *PostgresSessionRepository) Create(ctx context.Context, session *domain.BotSession) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO bot_sessions (id, chat_id, user_id, intent, state, step, expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		session.ID, session.ChatID, session.UserID, session.Intent,
		session.State, session.Step, session.ExpiresAt, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return fmt.Errorf("bot.Repository.Create: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) GetActiveByChatID(ctx context.Context, chatID string) (*domain.BotSession, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, chat_id, user_id, intent, state, step, expires_at, created_at, updated_at
		 FROM bot_sessions
		 WHERE chat_id = $1 AND expires_at > NOW()
		 ORDER BY created_at DESC LIMIT 1`, chatID)

	s := &domain.BotSession{}
	err := row.Scan(&s.ID, &s.ChatID, &s.UserID, &s.Intent,
		&s.State, &s.Step, &s.ExpiresAt, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("bot.Repository.GetActiveByChatID: %w", err)
	}
	return s, nil
}

func (r *PostgresSessionRepository) Update(ctx context.Context, session *domain.BotSession) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE bot_sessions SET state = $1, step = $2, updated_at = $3 WHERE id = $4`,
		session.State, session.Step, session.UpdatedAt, session.ID)
	if err != nil {
		return fmt.Errorf("bot.Repository.Update: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM bot_sessions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("bot.Repository.Delete: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM bot_sessions WHERE expires_at <= NOW()`)
	if err != nil {
		return fmt.Errorf("bot.Repository.DeleteExpired: %w", err)
	}
	return nil
}

// --- PostgresUserLinkRepository ---

// PostgresUserLinkRepository implements domain.UserLinkRepository using PostgreSQL.
type PostgresUserLinkRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserLinkRepository creates a new PostgresUserLinkRepository.
func NewPostgresUserLinkRepository(pool *pgxpool.Pool) *PostgresUserLinkRepository {
	return &PostgresUserLinkRepository{pool: pool}
}

func (r *PostgresUserLinkRepository) Link(ctx context.Context, link *domain.BotUserLink) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO bot_user_links (telegram_user_id, user_id, telegram_username, linked_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (telegram_user_id) DO UPDATE
		 SET user_id = EXCLUDED.user_id, telegram_username = EXCLUDED.telegram_username, linked_at = EXCLUDED.linked_at`,
		link.TelegramUserID, link.UserID, link.TelegramUsername, link.LinkedAt)
	if err != nil {
		return fmt.Errorf("bot.Repository.Link: %w", err)
	}
	return nil
}

func (r *PostgresUserLinkRepository) GetByTelegramUserID(ctx context.Context, telegramUserID int64) (*domain.BotUserLink, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT telegram_user_id, user_id, COALESCE(telegram_username, ''), linked_at
		 FROM bot_user_links WHERE telegram_user_id = $1`, telegramUserID)

	l := &domain.BotUserLink{}
	err := row.Scan(&l.TelegramUserID, &l.UserID, &l.TelegramUsername, &l.LinkedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotLinked
	}
	if err != nil {
		return nil, fmt.Errorf("bot.Repository.GetByTelegramUserID: %w", err)
	}
	return l, nil
}

func (r *PostgresUserLinkRepository) GetByUserID(ctx context.Context, userID string) (*domain.BotUserLink, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT telegram_user_id, user_id, COALESCE(telegram_username, ''), linked_at
		 FROM bot_user_links WHERE user_id = $1`, userID)

	l := &domain.BotUserLink{}
	err := row.Scan(&l.TelegramUserID, &l.UserID, &l.TelegramUsername, &l.LinkedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotLinked
	}
	if err != nil {
		return nil, fmt.Errorf("bot.Repository.GetByUserID: %w", err)
	}
	return l, nil
}

func (r *PostgresUserLinkRepository) Delete(ctx context.Context, telegramUserID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM bot_user_links WHERE telegram_user_id = $1`, telegramUserID)
	if err != nil {
		return fmt.Errorf("bot.Repository.Delete: %w", err)
	}
	return nil
}

// --- PostgresLLMConfigRepository ---

// PostgresLLMConfigRepository implements domain.LLMConfigRepository using PostgreSQL.
type PostgresLLMConfigRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresLLMConfigRepository creates a new PostgresLLMConfigRepository.
func NewPostgresLLMConfigRepository(pool *pgxpool.Pool) *PostgresLLMConfigRepository {
	return &PostgresLLMConfigRepository{pool: pool}
}

func (r *PostgresLLMConfigRepository) GetSystem(ctx context.Context) (*domain.LLMConfig, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, COALESCE(user_id::text, ''), provider, COALESCE(api_key, ''), model, COALESCE(base_url, ''), created_at, updated_at
		 FROM llm_configs WHERE user_id IS NULL`)

	c := &domain.LLMConfig{}
	err := row.Scan(&c.ID, &c.UserID, &c.Provider, &c.APIKey, &c.Model, &c.BaseURL, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNoLLMConfig
	}
	if err != nil {
		return nil, fmt.Errorf("bot.Repository.GetSystem: %w", err)
	}
	return c, nil
}

func (r *PostgresLLMConfigRepository) GetByUserID(ctx context.Context, userID string) (*domain.LLMConfig, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, COALESCE(user_id::text, ''), provider, COALESCE(api_key, ''), model, COALESCE(base_url, ''), created_at, updated_at
		 FROM llm_configs WHERE user_id = $1`, userID)

	c := &domain.LLMConfig{}
	err := row.Scan(&c.ID, &c.UserID, &c.Provider, &c.APIKey, &c.Model, &c.BaseURL, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNoLLMConfig
	}
	if err != nil {
		return nil, fmt.Errorf("bot.Repository.GetByUserID: %w", err)
	}
	return c, nil
}

func (r *PostgresLLMConfigRepository) Upsert(ctx context.Context, cfg *domain.LLMConfig) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO llm_configs (id, user_id, provider, api_key, model, base_url, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (user_id) DO UPDATE
		 SET provider = EXCLUDED.provider, api_key = EXCLUDED.api_key, model = EXCLUDED.model,
		     base_url = EXCLUDED.base_url, updated_at = EXCLUDED.updated_at`,
		cfg.ID, nilIfEmpty(cfg.UserID), cfg.Provider, nilIfEmpty(cfg.APIKey), cfg.Model, nilIfEmpty(cfg.BaseURL), cfg.CreatedAt, cfg.UpdatedAt)
	if err != nil {
		return fmt.Errorf("bot.Repository.Upsert: %w", err)
	}
	return nil
}

// --- Helpers ---

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// MemoryRepository

type PostgresMemoryRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMemoryRepository(pool *pgxpool.Pool) *PostgresMemoryRepository {
	return &PostgresMemoryRepository{pool: pool}
}

func (r *PostgresMemoryRepository) Save(ctx context.Context, entry *domain.MemoryEntry) error {
	query := `INSERT INTO bot_memory (id, user_id, chat_id, role, content, intent, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, query, entry.ID, entry.UserID, entry.ChatID, entry.Role, entry.Content, nilIfEmpty(entry.Intent), entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("bot.MemoryRepository.Save: %w", err)
	}
	return nil
}

func (r *PostgresMemoryRepository) GetRecent(ctx context.Context, userID string, limit int) ([]*domain.MemoryEntry, error) {
	query := `SELECT id, user_id, chat_id, role, content, COALESCE(intent, ''), created_at
		FROM bot_memory WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`
	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("bot.MemoryRepository.GetRecent: %w", err)
	}
	defer rows.Close()

	var entries []*domain.MemoryEntry
	for rows.Next() {
		e := &domain.MemoryEntry{}
		if err := rows.Scan(&e.ID, &e.UserID, &e.ChatID, &e.Role, &e.Content, &e.Intent, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("bot.MemoryRepository.GetRecent scan: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("bot.MemoryRepository.GetRecent iteration: %w", err)
	}

	// Reverse to chronological order.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

func (r *PostgresMemoryRepository) DeleteOld(ctx context.Context, userID string, keepLast int) error {
	query := `DELETE FROM bot_memory WHERE user_id = $1 AND id NOT IN (
		SELECT id FROM bot_memory WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2
	)`
	_, err := r.pool.Exec(ctx, query, userID, keepLast)
	if err != nil {
		return fmt.Errorf("bot.MemoryRepository.DeleteOld: %w", err)
	}
	return nil
}

// UserPrefsRepository

type PostgresUserPrefsRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserPrefsRepository(pool *pgxpool.Pool) *PostgresUserPrefsRepository {
	return &PostgresUserPrefsRepository{pool: pool}
}

func (r *PostgresUserPrefsRepository) Get(ctx context.Context, userID string) (*domain.UserPrefs, error) {
	query := `SELECT user_id, style, language, notes, updated_at FROM bot_user_prefs WHERE user_id = $1`
	row := r.pool.QueryRow(ctx, query, userID)
	p := &domain.UserPrefs{}
	err := row.Scan(&p.UserID, &p.Style, &p.Language, &p.Notes, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return &domain.UserPrefs{UserID: userID, Style: domain.StyleCasual, Language: "ru"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("bot.UserPrefsRepository.Get: %w", err)
	}
	return p, nil
}

func (r *PostgresUserPrefsRepository) Upsert(ctx context.Context, prefs *domain.UserPrefs) error {
	query := `INSERT INTO bot_user_prefs (user_id, style, language, notes, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id) DO UPDATE SET style = $2, language = $3, notes = $4, updated_at = NOW()`
	_, err := r.pool.Exec(ctx, query, prefs.UserID, prefs.Style, prefs.Language, prefs.Notes)
	if err != nil {
		return fmt.Errorf("bot.UserPrefsRepository.Upsert: %w", err)
	}
	return nil
}
