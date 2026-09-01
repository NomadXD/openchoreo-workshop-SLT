package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps the Postgres connection pool and all chat-gateway queries.
type Store struct {
	pool *pgxpool.Pool
}

const schema = `
CREATE TABLE IF NOT EXISTS conversations (
	id text PRIMARY KEY,
	subject_role text,
	subject_id text,
	created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS messages (
	id bigserial PRIMARY KEY,
	conversation_id text REFERENCES conversations(id),
	role text NOT NULL,
	content text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_log (
	id bigserial PRIMARY KEY,
	actor_role text,
	actor_id text,
	target_customer_id text,
	action text,
	details jsonb,
	created_at timestamptz NOT NULL DEFAULT now()
);
`

// newStore connects to Postgres and runs the idempotent migration.
func newStore(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}
	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("running migration: %w", err)
	}
	return nil
}

func (s *Store) Close() {
	s.pool.Close()
}

// ensureConversation creates the conversation row if it doesn't already
// exist. Safe to call repeatedly for the same id.
func (s *Store) ensureConversation(ctx context.Context, id, subjectRole, subjectID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO conversations (id, subject_role, subject_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING`,
		id, subjectRole, subjectID, time.Now())
	return err
}

// insertMessage persists one turn of a conversation.
func (s *Store) insertMessage(ctx context.Context, conversationID, role, content string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO messages (conversation_id, role, content, created_at)
		VALUES ($1, $2, $3, $4)`,
		conversationID, role, content, time.Now())
	return err
}

// HistoryItem is one prior turn of a conversation, as sent to chat-agent.
type HistoryItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// recentHistory returns up to `limit` most recent messages for a
// conversation, ordered oldest-first.
func (s *Store) recentHistory(ctx context.Context, conversationID string, limit int) ([]HistoryItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT role, content FROM (
			SELECT role, content, id FROM messages
			WHERE conversation_id = $1
			ORDER BY id DESC
			LIMIT $2
		) recent
		ORDER BY id ASC`,
		conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []HistoryItem
	for rows.Next() {
		var item HistoryItem
		if err := rows.Scan(&item.Role, &item.Content); err != nil {
			return nil, err
		}
		history = append(history, item)
	}
	return history, rows.Err()
}

// MessageRecord is a single stored message, as returned by the REST API.
type MessageRecord struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// listMessages returns every message for a conversation, ordered
// oldest-first.
func (s *Store) listMessages(ctx context.Context, conversationID string) ([]MessageRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT role, content, created_at FROM messages
		WHERE conversation_id = $1
		ORDER BY id ASC`,
		conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]MessageRecord, 0)
	for rows.Next() {
		var m MessageRecord
		if err := rows.Scan(&m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// insertAudit records an audited tool call.
func (s *Store) insertAudit(ctx context.Context, actorRole, actorID, targetCustomerID, action string, details any) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshalling audit details: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO audit_log (actor_role, actor_id, target_customer_id, action, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		actorRole, actorID, targetCustomerID, action, detailsJSON, time.Now())
	return err
}
