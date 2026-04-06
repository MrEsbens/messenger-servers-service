package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Server struct {
	ID          uuid.UUID
	Name        string
	Description *string
	OwnerID     uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

func NewServer(name string, ownerID uuid.UUID, description *string) (*Server, error) {
	if err := ValidateServerName(name); err != nil {
		return nil, err
	}

	if ownerID == uuid.Nil {
		return nil, errors.New("owner_id is required")
	}

	return &Server{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func ValidateServerName(name string) error {
	if name == "" {
		return errors.New("server name is required")
	}
	if len(name) < 2 {
		return errors.New("server name must be at least 2 characters")
	}
	if len(name) > 100 {
		return errors.New("server name must be at most 100 characters")
	}
	return nil
}

func (s *Server) Update(name *string, description *string) {
	if name != nil {
		s.Name = *name
	}
	if description != nil {
		s.Description = description
	}
	s.UpdatedAt = time.Now()
}

func (s *Server) SoftDelete() {
	now := time.Now()
	s.DeletedAt = &now
	s.UpdatedAt = now
}

func (s *Server) IsDeleted() bool {
	return s.DeletedAt != nil
}
