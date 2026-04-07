package repository

import (
	"context"
	"database/sql"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/google/uuid"
)

// MemberRepository — интерфейс для работы с участниками
type MemberRepository interface {
	Create(ctx context.Context, member *domain.ServerMember) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ServerMember, error)
	GetByServerAndUser(ctx context.Context, serverID, userID uuid.UUID) (*domain.ServerMember, error)
	GetByServer(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*domain.ServerMember, error)
	GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.ServerMember, error)
	Update(ctx context.Context, member *domain.ServerMember) error
	Delete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, serverID, userID uuid.UUID) (bool, error)
	CountByServer(ctx context.Context, serverID uuid.UUID) (int, error)
}

type postgresMemberRepository struct {
	db *sql.DB
}

func NewMemberRepository(db *sql.DB) MemberRepository {
	return &postgresMemberRepository{db: db}
}

func (r *postgresMemberRepository) Create(ctx context.Context, member *domain.ServerMember) error {
	query := `
		INSERT INTO server_members (id, server_id, user_id, joined_at, is_muted, is_banned)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		member.ID,
		member.ServerID,
		member.UserID,
		member.JoinedAt,
		member.IsMuted,
		member.IsBanned,
	)

	return err
}

func (r *postgresMemberRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ServerMember, error) {
	query := `
		SELECT id, server_id, user_id, joined_at, is_muted, is_banned, muted_until
		FROM server_members
		WHERE id = $1
	`

	member := &domain.ServerMember{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&member.ID,
		&member.ServerID,
		&member.UserID,
		&member.JoinedAt,
		&member.IsMuted,
		&member.IsBanned,
		&member.MutedUntil,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return member, nil
}

func (r *postgresMemberRepository) GetByServerAndUser(ctx context.Context, serverID, userID uuid.UUID) (*domain.ServerMember, error) {
	query := `
		SELECT id, server_id, user_id, joined_at, is_muted, is_banned, muted_until
		FROM server_members
		WHERE server_id = $1 AND user_id = $2
	`

	member := &domain.ServerMember{}
	err := r.db.QueryRowContext(ctx, query, serverID, userID).Scan(
		&member.ID,
		&member.ServerID,
		&member.UserID,
		&member.JoinedAt,
		&member.IsMuted,
		&member.IsBanned,
		&member.MutedUntil,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return member, nil
}

func (r *postgresMemberRepository) GetByServer(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*domain.ServerMember, error) {
	query := `
		SELECT id, server_id, user_id, joined_at, is_muted, is_banned, muted_until
		FROM server_members
		WHERE server_id = $1
		ORDER BY joined_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*domain.ServerMember
	for rows.Next() {
		member := &domain.ServerMember{}
		err := rows.Scan(
			&member.ID,
			&member.ServerID,
			&member.UserID,
			&member.JoinedAt,
			&member.IsMuted,
			&member.IsBanned,
			&member.MutedUntil,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, rows.Err()
}

func (r *postgresMemberRepository) GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.ServerMember, error) {
	query := `
		SELECT id, server_id, user_id, joined_at, is_muted, is_banned, muted_until
		FROM server_members
		WHERE user_id = $1
		ORDER BY joined_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*domain.ServerMember
	for rows.Next() {
		member := &domain.ServerMember{}
		err := rows.Scan(
			&member.ID,
			&member.ServerID,
			&member.UserID,
			&member.JoinedAt,
			&member.IsMuted,
			&member.IsBanned,
			&member.MutedUntil,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, rows.Err()
}

func (r *postgresMemberRepository) Update(ctx context.Context, member *domain.ServerMember) error {
	query := `
		UPDATE server_members
		SET is_muted = $2, is_banned = $3, muted_until = $4
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		member.ID,
		member.IsMuted,
		member.IsBanned,
		member.MutedUntil,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *postgresMemberRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM server_members WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *postgresMemberRepository) Exists(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM server_members WHERE server_id = $1 AND user_id = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, serverID, userID).Scan(&exists)
	return exists, err
}

func (r *postgresMemberRepository) CountByServer(ctx context.Context, serverID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM server_members WHERE server_id = $1 AND is_banned = FALSE`
	var count int
	err := r.db.QueryRowContext(ctx, query, serverID).Scan(&count)
	return count, err
}
