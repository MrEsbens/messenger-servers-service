package repository

import (
	"context"
	"database/sql"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/google/uuid"
)

// ConfigRepository — интерфейс для работы с конфигами
type ConfigRepository interface {
	// Server Config
	GetServerConfig(ctx context.Context, serverID uuid.UUID) (*domain.ServerConfig, error)
	UpdateServerConfig(ctx context.Context, config *domain.ServerConfig) error
	CreateServerConfig(ctx context.Context, config *domain.ServerConfig) error

	// Moderation Config
	GetModerationConfig(ctx context.Context, serverID uuid.UUID) (*domain.ModerationConfig, error)
	UpdateModerationConfig(ctx context.Context, config *domain.ModerationConfig) error
	CreateModerationConfig(ctx context.Context, config *domain.ModerationConfig) error
}

type postgresConfigRepository struct {
	db *sql.DB
}

func NewConfigRepository(db *sql.DB) ConfigRepository {
	return &postgresConfigRepository{db: db}
}

// ───────────────────────────────────────────────────────────
// Server Config
// ───────────────────────────────────────────────────────────

func (r *postgresConfigRepository) GetServerConfig(ctx context.Context, serverID uuid.UUID) (*domain.ServerConfig, error) {
	query := `
		SELECT id, server_id, max_members, max_channels, default_notification_mode, moderation_enabled, created_at, updated_at
		FROM server_configs
		WHERE server_id = $1
	`

	config := &domain.ServerConfig{}
	err := r.db.QueryRowContext(ctx, query, serverID).Scan(
		&config.ID,
		&config.ServerID,
		&config.MaxMembers,
		&config.MaxChannels,
		&config.DefaultNotificationMode,
		&config.ModerationEnabled,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (r *postgresConfigRepository) UpdateServerConfig(ctx context.Context, config *domain.ServerConfig) error {
	query := `
		UPDATE server_configs
		SET max_members = $2, max_channels = $3, default_notification_mode = $4, moderation_enabled = $5, updated_at = $6
		WHERE server_id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		config.ServerID,
		config.MaxMembers,
		config.MaxChannels,
		config.DefaultNotificationMode,
		config.ModerationEnabled,
		config.UpdatedAt,
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

func (r *postgresConfigRepository) CreateServerConfig(ctx context.Context, config *domain.ServerConfig) error {
	query := `
		INSERT INTO server_configs (id, server_id, max_members, max_channels, default_notification_mode, moderation_enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID,
		config.ServerID,
		config.MaxMembers,
		config.MaxChannels,
		config.DefaultNotificationMode,
		config.ModerationEnabled,
	)

	return err
}

// ───────────────────────────────────────────────────────────
// Moderation Config
// ───────────────────────────────────────────────────────────

func (r *postgresConfigRepository) GetModerationConfig(ctx context.Context, serverID uuid.UUID) (*domain.ModerationConfig, error) {
	query := `
		SELECT id, server_id, profanity_filter_action, toxicity_filter_action, nsfw_text_filter_action, 
		    hate_speech_filter_action, created_at, updated_at
		FROM moderation_configs
		WHERE server_id = $1
	`

	config := &domain.ModerationConfig{}
	err := r.db.QueryRowContext(ctx, query, serverID).Scan(
		&config.ID,
		&config.ServerID,
		&config.ProfanityFilterAction,
		&config.ToxicityFilterAction,
		&config.NsfwTextFilterAction,
		&config.HateSpeechFilterAction,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (r *postgresConfigRepository) UpdateModerationConfig(ctx context.Context, config *domain.ModerationConfig) error {
	query := `
		UPDATE moderation_configs
		SET profanity_filter_action = $2, toxicity_filter_action = $3, nsfw_text_filter_action = $4,
		    hate_speech_filter_action = $5, updated_at = $6
		WHERE server_id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		config.ServerID,
		config.ProfanityFilterAction,
		config.ToxicityFilterAction,
		config.NsfwTextFilterAction,
		config.HateSpeechFilterAction,
		config.UpdatedAt,
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

func (r *postgresConfigRepository) CreateModerationConfig(ctx context.Context, config *domain.ModerationConfig) error {
	query := `
		INSERT INTO moderation_configs (id, server_id, profanity_filter_action, toxicity_filter_action, 
		                                nsfw_text_filter_action, hate_speech_filter_action)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID,
		config.ServerID,
		config.ProfanityFilterAction,
		config.ToxicityFilterAction,
		config.NsfwTextFilterAction,
		config.HateSpeechFilterAction,
	)

	return err
}
