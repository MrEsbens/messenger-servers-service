package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockModerationRepository struct {
	mock.Mock
}

func NewMockModerationRepository(t *testing.T) *MockModerationRepository {
	return &MockModerationRepository{}
}

func (m *MockModerationRepository) Create(ctx context.Context, violation *domain.ModerationViolation) error {
	args := m.Called(ctx, violation)
	return args.Error(0)
}

func (m *MockModerationRepository) GetByServer(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*domain.ModerationViolation, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ModerationViolation), args.Error(1)
}

func (m *MockModerationRepository) GetByUser(ctx context.Context, serverID, userID uuid.UUID, limit, offset int) ([]*domain.ModerationViolation, error) {
	args := m.Called(ctx, serverID, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ModerationViolation), args.Error(1)
}

func (m *MockModerationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ModerationViolation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ModerationViolation), args.Error(1)
}
