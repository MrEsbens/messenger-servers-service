package repository

import (
	"context"
	"database/sql"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/google/uuid"
)

// ModerationRepository — интерфейс для журнала нарушений
type ModerationRepository interface {
	Create(ctx context.Context, violation *domain.ModerationViolation) error
	GetByServer(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*domain.ModerationViolation, error)
	GetByUser(ctx context.Context, serverID, userID uuid.UUID, limit, offset int) ([]*domain.ModerationViolation, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ModerationViolation, error)
}

type postgresModerationRepository struct {
	db *sql.DB
}

func NewModerationRepository(db *sql.DB) ModerationRepository {
	return &postgresModerationRepository{db: db}
}

func (r *postgresModerationRepository) Create(ctx context.Context, violation *domain.ModerationViolation) error {
	query := `
		INSERT INTO moderation_violations (id, server_id, user_id, message_id, message_content, violation_type, action_taken)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		violation.ID,
		violation.ServerID,
		violation.UserID,
		violation.MessageID,
		violation.MessageContent,
		violation.ViolationType,
		violation.ActionTaken,
	)

	return err
}

func (r *postgresModerationRepository) GetByServer(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*domain.ModerationViolation, error) {
	query := `
		SELECT id, server_id, user_id, message_id, message_content, violation_type, action_taken, created_at
		FROM moderation_violations
		WHERE server_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var violations []*domain.ModerationViolation
	for rows.Next() {
		v := &domain.ModerationViolation{}
		var messageID sql.NullString
		var messageContent sql.NullString
		err := rows.Scan(
			&v.ID,
			&v.ServerID,
			&v.UserID,
			&messageID,
			&messageContent,
			&v.ViolationType,
			&v.ActionTaken,
			&v.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if messageID.Valid {
			id, _ := uuid.Parse(messageID.String)
			v.MessageID = &id
		}
		if messageContent.Valid {
			v.MessageContent = &messageContent.String
		}
		violations = append(violations, v)
	}

	return violations, rows.Err()
}

func (r *postgresModerationRepository) GetByUser(ctx context.Context, serverID, userID uuid.UUID, limit, offset int) ([]*domain.ModerationViolation, error) {
	query := `
		SELECT id, server_id, user_id, message_id, message_content, violation_type, action_taken, created_at
		FROM moderation_violations
		WHERE server_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var violations []*domain.ModerationViolation
	for rows.Next() {
		v := &domain.ModerationViolation{}
		var messageID sql.NullString
		var messageContent sql.NullString
		err := rows.Scan(
			&v.ID,
			&v.ServerID,
			&v.UserID,
			&messageID,
			&messageContent,
			&v.ViolationType,
			&v.ActionTaken,
			&v.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if messageID.Valid {
			id, _ := uuid.Parse(messageID.String)
			v.MessageID = &id
		}
		if messageContent.Valid {
			v.MessageContent = &messageContent.String
		}
		violations = append(violations, v)
	}

	return violations, rows.Err()
}

func (r *postgresModerationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ModerationViolation, error) {
	query := `
		SELECT id, server_id, user_id, message_id, message_content, violation_type, action_taken, created_at
		FROM moderation_violations
		WHERE id = $1
	`

	v := &domain.ModerationViolation{}
	var messageID sql.NullString
	var messageContent sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&v.ID,
		&v.ServerID,
		&v.UserID,
		&messageID,
		&messageContent,
		&v.ViolationType,
		&v.ActionTaken,
		&v.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if messageID.Valid {
		id, _ := uuid.Parse(messageID.String)
		v.MessageID = &id
	}
	if messageContent.Valid {
		v.MessageContent = &messageContent.String
	}

	return v, nil
}
