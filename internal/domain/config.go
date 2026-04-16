package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type ServerConfig struct {
	ID                      uuid.UUID
	ServerID                uuid.UUID
	MaxMembers              int
	MaxChannels             int
	DefaultNotificationMode string // all, mentions, none
	ModerationEnabled       bool   // 🔴 Главный переключатель модерации
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func NewServerConfig(serverID uuid.UUID) *ServerConfig {
	return &ServerConfig{
		ID:                      uuid.New(),
		ServerID:                serverID,
		MaxMembers:              500,
		MaxChannels:             500,
		DefaultNotificationMode: "all",
		ModerationEnabled:       false, // По умолчанию модерация выключена
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}
}

func (c *ServerConfig) Update(
	maxMembers *int,
	maxChannels *int,
	defaultNotificationMode *string,
	moderationEnabled *bool,
) {
	if maxMembers != nil {
		c.MaxMembers = *maxMembers
	}
	if maxChannels != nil {
		c.MaxChannels = *maxChannels
	}
	if defaultNotificationMode != nil {
		c.DefaultNotificationMode = *defaultNotificationMode
	}
	if moderationEnabled != nil {
		c.ModerationEnabled = *moderationEnabled
	}
	c.UpdatedAt = time.Now()
}

func (c *ServerConfig) Validate() error {
	if c.ServerID == uuid.Nil {
		return errors.New("server_id is required")
	}
	if c.MaxMembers < 1 || c.MaxMembers > 10000 {
		return errors.New("max_members must be between 1 and 10000")
	}
	if c.MaxChannels < 1 || c.MaxChannels > 1000 {
		return errors.New("max_channels must be between 1 and 1000")
	}
	return nil
}

type ModerationAction string

const (
	ActionNone   ModerationAction = "none"   // Игнорировать
	ActionWarn   ModerationAction = "warn"   // Предупреждение
	ActionMute   ModerationAction = "mute"   // Мут
	ActionBan    ModerationAction = "ban"    // Бан
	ActionDelete ModerationAction = "delete" // Удалить сообщение
)

func (a ModerationAction) IsValid() bool {
	switch a {
	case ActionNone, ActionWarn, ActionMute, ActionBan, ActionDelete:
		return true
	}
	return false
}

type ModerationConfig struct {
	ID                     uuid.UUID
	ServerID               uuid.UUID
	ProfanityFilterAction  ModerationAction // Мат/оскорбления
	ToxicityFilterAction   ModerationAction // Токсичность (ML)
	NsfwTextFilterAction   ModerationAction // 18+ текст (ML)
	HateSpeechFilterAction ModerationAction // Хейт-спич (ML)
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func NewModerationConfig(serverID uuid.UUID) *ModerationConfig {
	return &ModerationConfig{
		ID:                     uuid.New(),
		ServerID:               serverID,
		ProfanityFilterAction:  ActionNone,
		ToxicityFilterAction:   ActionNone,
		NsfwTextFilterAction:   ActionNone,
		HateSpeechFilterAction: ActionNone,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}
}

func (c *ModerationConfig) Update(
	profanityAction *ModerationAction,
	toxicityAction *ModerationAction,
	nsfwAction *ModerationAction,
	hateSpeechAction *ModerationAction,
) {
	if profanityAction != nil {
		c.ProfanityFilterAction = *profanityAction
	}
	if toxicityAction != nil {
		c.ToxicityFilterAction = *toxicityAction
	}
	if nsfwAction != nil {
		c.NsfwTextFilterAction = *nsfwAction
	}
	if hateSpeechAction != nil {
		c.HateSpeechFilterAction = *hateSpeechAction
	}
	c.UpdatedAt = time.Now()
}

func (c *ModerationConfig) GetActionForFilter(filterType string) ModerationAction {
	switch filterType {
	case "profanity":
		return c.ProfanityFilterAction
	case "toxicity":
		return c.ToxicityFilterAction
	case "nsfw_text":
		return c.NsfwTextFilterAction
	case "hate_speech":
		return c.HateSpeechFilterAction
	default:
		return ActionNone
	}
}

func (c *ModerationConfig) IsEnabled() bool {
	return c.ProfanityFilterAction != ActionNone ||
		c.ToxicityFilterAction != ActionNone ||
		c.NsfwTextFilterAction != ActionNone ||
		c.HateSpeechFilterAction != ActionNone
}

func (c *ModerationConfig) Validate() error {
	if c.ServerID == uuid.Nil {
		return errors.New("server_id is required")
	}

	actions := []ModerationAction{
		c.ProfanityFilterAction,
		c.ToxicityFilterAction,
		c.NsfwTextFilterAction,
		c.HateSpeechFilterAction,
	}

	for _, action := range actions {
		if !action.IsValid() {
			return errors.New("invalid moderation action")
		}
	}

	return nil
}
