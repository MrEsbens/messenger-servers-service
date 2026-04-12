package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockRoleRepository struct {
	mock.Mock
}

func NewMockRoleRepository(t *testing.T) *MockRoleRepository {
	return &MockRoleRepository{}
}

func (m *MockRoleRepository) Create(ctx context.Context, role *domain.ServerRole) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ServerRole, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ServerRole), args.Error(1)
}

func (m *MockRoleRepository) GetByServer(ctx context.Context, serverID uuid.UUID) ([]*domain.ServerRole, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ServerRole), args.Error(1)
}

func (m *MockRoleRepository) Update(ctx context.Context, role *domain.ServerRole) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleRepository) AssignToMember(ctx context.Context, memberID, roleID uuid.UUID) error {
	args := m.Called(ctx, memberID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepository) RemoveFromMember(ctx context.Context, memberID, roleID uuid.UUID) error {
	args := m.Called(ctx, memberID, roleID)
	return args.Error(0)
}

func (m *MockRoleRepository) GetMemberRoles(ctx context.Context, memberID uuid.UUID) ([]*domain.ServerRole, error) {
	args := m.Called(ctx, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.ServerRole), args.Error(1)
}
