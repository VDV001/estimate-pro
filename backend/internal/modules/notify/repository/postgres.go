package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VDV001/estimate-pro/backend/internal/modules/notify/domain"
)

// --- NotificationRepository ---

type PostgresNotificationRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresNotificationRepository(pool *pgxpool.Pool) *PostgresNotificationRepository {
	return &PostgresNotificationRepository{pool: pool}
}

func (r *PostgresNotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notifications (id, user_id, event_type, title, message, project_id, read, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		n.ID, n.UserID, n.EventType, n.Title, n.Message, nullIfEmpty(n.ProjectID), n.Read, n.CreatedAt)
	if err != nil {
		return fmt.Errorf("NotificationRepository.Create: %w", err)
	}
	return nil
}

func (r *PostgresNotificationRepository) CreateBatch(ctx context.Context, notifications []*domain.Notification) error {
	batch := &pgx.Batch{}
	for _, n := range notifications {
		batch.Queue(
			`INSERT INTO notifications (id, user_id, event_type, title, message, project_id, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			n.ID, n.UserID, n.EventType, n.Title, n.Message, nullIfEmpty(n.ProjectID), n.CreatedAt)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range notifications {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("NotificationRepository.CreateBatch: %w", err)
		}
	}
	return nil
}

func (r *PostgresNotificationRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("NotificationRepository.ListByUser: count: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, event_type, title, message, COALESCE(project_id::text, ''), read, created_at
		 FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("NotificationRepository.ListByUser: query: %w", err)
	}
	defer rows.Close()

	var result []*domain.Notification
	for rows.Next() {
		n := new(domain.Notification)
		if err := rows.Scan(&n.ID, &n.UserID, &n.EventType, &n.Title, &n.Message, &n.ProjectID, &n.Read, &n.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("NotificationRepository.ListByUser: scan: %w", err)
		}
		result = append(result, n)
	}
	return result, total, nil
}

func (r *PostgresNotificationRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = false`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("NotificationRepository.CountUnread: %w", err)
	}
	return count, nil
}

func (r *PostgresNotificationRepository) MarkRead(ctx context.Context, userID, notificationID string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`, notificationID, userID)
	if err != nil {
		return fmt.Errorf("NotificationRepository.MarkRead: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}

func (r *PostgresNotificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read = true WHERE user_id = $1 AND read = false`, userID)
	if err != nil {
		return fmt.Errorf("NotificationRepository.MarkAllRead: %w", err)
	}
	return nil
}

// --- PreferenceRepository ---

type PostgresPreferenceRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPreferenceRepository(pool *pgxpool.Pool) *PostgresPreferenceRepository {
	return &PostgresPreferenceRepository{pool: pool}
}

func (r *PostgresPreferenceRepository) Get(ctx context.Context, userID string) ([]*domain.Preference, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, channel, enabled FROM notification_preferences WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("PreferenceRepository.Get: %w", err)
	}
	defer rows.Close()

	var result []*domain.Preference
	for rows.Next() {
		p := new(domain.Preference)
		if err := rows.Scan(&p.UserID, &p.Channel, &p.Enabled); err != nil {
			return nil, fmt.Errorf("PreferenceRepository.Get: scan: %w", err)
		}
		result = append(result, p)
	}
	return result, nil
}

func (r *PostgresPreferenceRepository) Upsert(ctx context.Context, pref *domain.Preference) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notification_preferences (user_id, channel, enabled)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, channel) DO UPDATE SET enabled = EXCLUDED.enabled`,
		pref.UserID, pref.Channel, pref.Enabled)
	if err != nil {
		return fmt.Errorf("PreferenceRepository.Upsert: %w", err)
	}
	return nil
}

// --- DeliveryLogRepository ---

type PostgresDeliveryLogRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresDeliveryLogRepository(pool *pgxpool.Pool) *PostgresDeliveryLogRepository {
	return &PostgresDeliveryLogRepository{pool: pool}
}

func (r *PostgresDeliveryLogRepository) Create(ctx context.Context, log *domain.DeliveryLog) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notification_log (id, user_id, event_type, channel, sent_at, status)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		log.ID, log.UserID, log.EventType, log.Channel, log.SentAt, log.Status)
	if err != nil {
		return fmt.Errorf("DeliveryLogRepository.Create: %w", err)
	}
	return nil
}

// --- Adapters ---

// EmailLookup resolves user ID to email by querying the users table directly.
type EmailLookup struct {
	pool *pgxpool.Pool
}

func NewEmailLookup(pool *pgxpool.Pool) *EmailLookup {
	return &EmailLookup{pool: pool}
}

func (l *EmailLookup) GetEmail(ctx context.Context, userID string) (string, error) {
	var email string
	err := l.pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	if err != nil {
		return "", fmt.Errorf("EmailLookup.GetEmail: %w", err)
	}
	return email, nil
}

// TelegramChatLookup resolves user ID to Telegram chat ID.
type TelegramChatLookup struct {
	pool *pgxpool.Pool
}

func NewTelegramChatLookup(pool *pgxpool.Pool) *TelegramChatLookup {
	return &TelegramChatLookup{pool: pool}
}

func (l *TelegramChatLookup) GetTelegramChatID(ctx context.Context, userID string) (string, error) {
	var chatID sql.NullString
	err := l.pool.QueryRow(ctx, `SELECT telegram_chat_id FROM users WHERE id = $1`, userID).Scan(&chatID)
	if err != nil {
		return "", fmt.Errorf("TelegramChatLookup.GetTelegramChatID: %w", err)
	}
	if !chatID.Valid || chatID.String == "" {
		return "", fmt.Errorf("TelegramChatLookup.GetTelegramChatID: user %s has no telegram_chat_id", userID)
	}
	return chatID.String, nil
}

// MemberListerAdapter wraps a direct query to project_members to implement domain.MemberLister.
type MemberListerAdapter struct {
	pool *pgxpool.Pool
}

func NewMemberListerAdapter(pool *pgxpool.Pool) *MemberListerAdapter {
	return &MemberListerAdapter{pool: pool}
}

func (a *MemberListerAdapter) ListMemberUserIDs(ctx context.Context, projectID string) ([]string, error) {
	rows, err := a.pool.Query(ctx, `SELECT user_id FROM project_members WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, fmt.Errorf("MemberListerAdapter.ListMemberUserIDs: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("MemberListerAdapter.ListMemberUserIDs: scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// UserNameLookup resolves user ID to display name.
type UserNameLookup struct {
	pool *pgxpool.Pool
}

func NewUserNameLookup(pool *pgxpool.Pool) *UserNameLookup {
	return &UserNameLookup{pool: pool}
}

func (l *UserNameLookup) GetName(ctx context.Context, userID string) (string, error) {
	var name string
	err := l.pool.QueryRow(ctx, `SELECT name FROM users WHERE id = $1`, userID).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("UserNameLookup.GetName: %w", err)
	}
	return name, nil
}

// --- Helpers ---

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
