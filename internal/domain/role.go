package domain

import (
	"time"

	"github.com/google/uuid"
)

type ServerRole struct {
	ID          uuid.UUID
	ServerID    uuid.UUID
	Name        string
	Color       string          // #RRGGBB
	Permissions []string        // Список прав: ["manage_channels", "kick_members", ...]
	Position    int             // Порядок в иерархии (0 = lowest)
	IsDefault   bool            // Роль @everyone
	CreatedAt   time.Time
}

func NewServerRole(serverID uuid.UUID, name string, isDefault bool) (*ServerRole, error) {
	if serverID == uuid.Nil {
		return nil, ErrRoleNotFound
	}
	if name == "" {
		return nil, ErrRoleNotFound
	}

	return &ServerRole{
		ID:          uuid.New(),
		ServerID:    serverID,
		Name:        name,
		Color:       "#99AAB5", // Default Discord-like color
		Permissions: []string{},
		Position:    0,
		IsDefault:   isDefault,
		CreatedAt:   time.Now(),
	}, nil
}

func (r *ServerRole) Update(name *string, color *string, permissions []string, position *int) {
	if name != nil {
		r.Name = *name
	}
	if color != nil {
		r.Color = *color
	}
	if permissions != nil {
		r.Permissions = permissions
	}
	if position != nil {
		r.Position = *position
	}
}

func (r *ServerRole) HasPermission(permission string) bool {
	for _, p := range r.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (r *ServerRole) AddPermission(permission string) {
	if !r.HasPermission(permission) {
		r.Permissions = append(r.Permissions, permission)
	}
}

func (r *ServerRole) RemovePermission(permission string) {
	for i, p := range r.Permissions {
		if p == permission {
			r.Permissions = append(r.Permissions[:i], r.Permissions[i+1:]...)
			return
		}
	}
}