package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockConfigRepository struct {
	mock.Mock
}

func NewMockConfigRepository(t *testing.T) *MockConfigRepository {
	return &MockConfigRepository{}
}

func (m *MockConfigRepository) GetServerConfig(ctx context.Context, serverID uuid.UUID) (*domain.ServerConfig, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ServerConfig), args.Error(1)
}

func (m *MockConfigRepository) UpdateServerConfig(ctx context.Context, config *domain.ServerConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockConfigRepository) CreateServerConfig(ctx context.Context, config *domain.ServerConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockConfigRepository) GetModerationConfig(ctx context.Context, serverID uuid.UUID) (*domain.ModerationConfig, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ModerationConfig), args.Error(1)
}

func (m *MockConfigRepository) UpdateModerationConfig(ctx context.Context, config *domain.ModerationConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockConfigRepository) CreateModerationConfig(ctx context.Context, config *domain.ModerationConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}
