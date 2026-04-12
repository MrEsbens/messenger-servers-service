package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockServerRepository struct {
	mock.Mock
}

func NewMockServerRepository(t *testing.T) *MockServerRepository {
	return &MockServerRepository{}
}

func (m *MockServerRepository) Create(ctx context.Context, server *domain.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Server), args.Error(1)
}

func (m *MockServerRepository) GetByOwnerID(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.Server, error) {
	args := m.Called(ctx, ownerID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Server), args.Error(1)
}

func (m *MockServerRepository) Update(ctx context.Context, server *domain.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}
