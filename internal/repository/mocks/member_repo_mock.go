package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockMemberRepository struct {
	mock.Mock
}

func NewMockMemberRepository(t *testing.T) *MockMemberRepository {
	return &MockMemberRepository{}
}

func (m *MockMemberRepository) Create(ctx context.Context, member *domain.ServerMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockMemberRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ServerMember, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ServerMember), args.Error(1)
}

func (m *MockMemberRepository) GetByServerAndUser(ctx context.Context, serverID, userID uuid.UUID) (*domain.ServerMember, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ServerMember), args.Error(1)
}

func (m *MockMemberRepository) GetByServer(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*domain.ServerMember, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ServerMember), args.Error(1)
}

func (m *MockMemberRepository) GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.ServerMember, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ServerMember), args.Error(1)
}

func (m *MockMemberRepository) Update(ctx context.Context, member *domain.ServerMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockMemberRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMemberRepository) Exists(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, serverID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMemberRepository) CountByServer(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Int(0), args.Error(1)
}
