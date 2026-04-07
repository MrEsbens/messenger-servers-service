package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type ServerChat struct {
	ServerID  uuid.UUID
	ChatID    uuid.UUID
	Name      string
	Position  int
	CreatedAt time.Time
}

type ChatRepository interface {
	AddChat(ctx context.Context, serverID, chatID uuid.UUID, name string) error
	RemoveChat(ctx context.Context, serverID, chatID uuid.UUID) error
	GetByServer(ctx context.Context, serverID uuid.UUID) ([]*ServerChat, error)
	GetByChatID(ctx context.Context, chatID uuid.UUID) (*ServerChat, error)
}

type postgresChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) ChatRepository {
	return &postgresChatRepository{db: db}
}

func (r *postgresChatRepository) AddChat(ctx context.Context, serverID, chatID uuid.UUID, name string) error {
	query := `
		INSERT INTO server_chats (server_id, chat_id, name, position)
		VALUES ($1, $2, $3, 0)
	`
	_, err := r.db.ExecContext(ctx, query, serverID, chatID, name)
	return err
}

func (r *postgresChatRepository) RemoveChat(ctx context.Context, serverID, chatID uuid.UUID) error {
	query := `DELETE FROM server_chats WHERE server_id = $1 AND chat_id = $2`
	_, err := r.db.ExecContext(ctx, query, serverID, chatID)
	return err
}

func (r *postgresChatRepository) GetByServer(ctx context.Context, serverID uuid.UUID) ([]*ServerChat, error) {
	query := `
		SELECT server_id, chat_id, name, position, created_at
		FROM server_chats
		WHERE server_id = $1
		ORDER BY position ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []*ServerChat
	for rows.Next() {
		chat := &ServerChat{}
		err := rows.Scan(
			&chat.ServerID,
			&chat.ChatID,
			&chat.Name,
			&chat.Position,
			&chat.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}

	return chats, rows.Err()
}

func (r *postgresChatRepository) GetByChatID(ctx context.Context, chatID uuid.UUID) (*ServerChat, error) {
	query := `
		SELECT server_id, chat_id, name, position, created_at
		FROM server_chats
		WHERE chat_id = $1
	`

	chat := &ServerChat{}
	err := r.db.QueryRowContext(ctx, query, chatID).Scan(
		&chat.ServerID,
		&chat.ChatID,
		&chat.Name,
		&chat.Position,
		&chat.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return chat, nil
}
