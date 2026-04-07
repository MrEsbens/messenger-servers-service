package repository

import (
	"context"
	"database/sql"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type RoleRepository interface {
	Create(ctx context.Context, role *domain.ServerRole) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ServerRole, error)
	GetByServer(ctx context.Context, serverID uuid.UUID) ([]*domain.ServerRole, error)
	Update(ctx context.Context, role *domain.ServerRole) error
	Delete(ctx context.Context, id uuid.UUID) error
	AssignToMember(ctx context.Context, memberID, roleID uuid.UUID) error
	RemoveFromMember(ctx context.Context, memberID, roleID uuid.UUID) error
	GetMemberRoles(ctx context.Context, memberID uuid.UUID) ([]*domain.ServerRole, error)
}

type postgresRoleRepository struct {
	db *sql.DB
}

func NewRoleRepository(db *sql.DB) RoleRepository {
	return &postgresRoleRepository{db: db}
}

// ───────────────────────────────────────────────────────────
// Create — 🔧 pq.Array() автоматически конвертирует []string
// ───────────────────────────────────────────────────────────

func (r *postgresRoleRepository) Create(ctx context.Context, role *domain.ServerRole) error {
	query := `
		INSERT INTO server_roles (id, server_id, name, color, permissions, position, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		role.ID,
		role.ServerID,
		role.Name,
		role.Color,
		pq.Array(role.Permissions), // ✅ Автоматическая конвертация
		role.Position,
		role.IsDefault,
	)

	return err
}

// ───────────────────────────────────────────────────────────
// GetByID — 🔧 pq.Array() автоматически парсит TEXT[]
// ───────────────────────────────────────────────────────────

func (r *postgresRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ServerRole, error) {
	query := `
		SELECT id, server_id, name, color, permissions, position, is_default, created_at
		FROM server_roles
		WHERE id = $1
	`

	role := &domain.ServerRole{}
	var createdAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&role.ID,
		&role.ServerID,
		&role.Name,
		&role.Color,
		pq.Array(&role.Permissions), // ✅ Автоматический парсинг
		&role.Position,
		&role.IsDefault,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		role.CreatedAt = createdAt.Time
	}

	return role, nil
}

// ───────────────────────────────────────────────────────────
// GetByServer
// ───────────────────────────────────────────────────────────

func (r *postgresRoleRepository) GetByServer(ctx context.Context, serverID uuid.UUID) ([]*domain.ServerRole, error) {
	query := `
		SELECT id, server_id, name, color, permissions, position, is_default, created_at
		FROM server_roles
		WHERE server_id = $1
		ORDER BY position ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*domain.ServerRole
	for rows.Next() {
		role := &domain.ServerRole{}
		var createdAt sql.NullTime

		err := rows.Scan(
			&role.ID,
			&role.ServerID,
			&role.Name,
			&role.Color,
			pq.Array(&role.Permissions), // ✅
			&role.Position,
			&role.IsDefault,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		if createdAt.Valid {
			role.CreatedAt = createdAt.Time
		}

		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// ───────────────────────────────────────────────────────────
// Update — 🔧 Теперь без лишнего параметра и JSON
// ───────────────────────────────────────────────────────────

func (r *postgresRoleRepository) Update(ctx context.Context, role *domain.ServerRole) error {
	query := `
		UPDATE server_roles
		SET name = $2, color = $3, permissions = $4, position = $5
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		role.ID,
		role.Name,
		role.Color,
		pq.Array(role.Permissions), // ✅
		role.Position,
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

// ───────────────────────────────────────────────────────────
// Delete
// ───────────────────────────────────────────────────────────

func (r *postgresRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM server_roles WHERE id = $1 AND is_default = FALSE`
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

// ───────────────────────────────────────────────────────────
// AssignToMember
// ───────────────────────────────────────────────────────────

func (r *postgresRoleRepository) AssignToMember(ctx context.Context, memberID, roleID uuid.UUID) error {
	query := `INSERT INTO member_roles (member_id, role_id) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, query, memberID, roleID)
	return err
}

// ───────────────────────────────────────────────────────────
// RemoveFromMember
// ───────────────────────────────────────────────────────────

func (r *postgresRoleRepository) RemoveFromMember(ctx context.Context, memberID, roleID uuid.UUID) error {
	query := `DELETE FROM member_roles WHERE member_id = $1 AND role_id = $2`
	_, err := r.db.ExecContext(ctx, query, memberID, roleID)
	return err
}

// ───────────────────────────────────────────────────────────
// GetMemberRoles
// ───────────────────────────────────────────────────────────

func (r *postgresRoleRepository) GetMemberRoles(ctx context.Context, memberID uuid.UUID) ([]*domain.ServerRole, error) {
	query := `
		SELECT r.id, r.server_id, r.name, r.color, r.permissions, r.position, r.is_default, r.created_at
		FROM server_roles r
		JOIN member_roles mr ON r.id = mr.role_id
		WHERE mr.member_id = $1
		ORDER BY r.position ASC
	`

	rows, err := r.db.QueryContext(ctx, query, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*domain.ServerRole
	for rows.Next() {
		role := &domain.ServerRole{}
		var createdAt sql.NullTime

		err := rows.Scan(
			&role.ID,
			&role.ServerID,
			&role.Name,
			&role.Color,
			pq.Array(&role.Permissions), // ✅
			&role.Position,
			&role.IsDefault,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		if createdAt.Valid {
			role.CreatedAt = createdAt.Time
		}

		roles = append(roles, role)
	}

	return roles, rows.Err()
}
