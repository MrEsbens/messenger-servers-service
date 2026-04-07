package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type ServerRepository interface {
	Create(ctx context.Context, server *domain.Server) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Server, error)
	GetByOwnerID(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.Server, error)
	Update(ctx context.Context, server *domain.Server) error
	Delete(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type postgresServerRepository struct {
	db *sql.DB
}

func NewServerRepository(db *sql.DB) ServerRepository {
	return &postgresServerRepository{db: db}
}

func (r *postgresServerRepository) Create(ctx context.Context, server *domain.Server) error {
	query := `
		INSERT INTO servers (id, name, description, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		server.ID,
		server.Name,
		server.Description,
		server.OwnerID,
		server.CreatedAt,
		server.UpdatedAt,
	)

	if err != nil {
		return err
	}

	if err := r.createDefaultConfigs(ctx, server.ID); err != nil {
		return err
	}

	return nil
}

func (r *postgresServerRepository) createDefaultConfigs(ctx context.Context, serverID uuid.UUID) error {
	// Server Config
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO server_configs (server_id, max_members, max_channels, default_notification_mode, moderation_enabled)
		VALUES ($1, 500, 500, 'all', false)
	`, serverID)
	if err != nil {
		return err
	}

	// Moderation Config
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO moderation_configs (server_id)
		VALUES ($1)
	`, serverID)
	if err != nil {
		return err
	}

	return nil
}

func (r *postgresServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Server, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at, deleted_at
		FROM servers
		WHERE id = $1 AND deleted_at IS NULL
	`

	server := &domain.Server{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&server.ID,
		&server.Name,
		&server.Description,
		&server.OwnerID,
		&server.CreatedAt,
		&server.UpdatedAt,
		&server.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (r *postgresServerRepository) GetByOwnerID(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.Server, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at, deleted_at
		FROM servers
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, ownerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*domain.Server
	for rows.Next() {
		server := &domain.Server{}
		err := rows.Scan(
			&server.ID,
			&server.Name,
			&server.Description,
			&server.OwnerID,
			&server.CreatedAt,
			&server.UpdatedAt,
			&server.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, rows.Err()
}

func (r *postgresServerRepository) Update(ctx context.Context, server *domain.Server) error {
	query := `
		UPDATE servers
		SET name = $2, description = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		server.ID,
		server.Name,
		server.Description,
		server.UpdatedAt,
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

func (r *postgresServerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM servers WHERE id = $1`
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

func (r *postgresServerRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE servers
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
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

func (r *postgresServerRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM servers WHERE id = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}
