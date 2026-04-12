package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-servers-service/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockChatRepository struct {
	mock.Mock
}

func NewMockChatRepository(t *testing.T) *MockChatRepository {
	return &MockChatRepository{}
}

func (m *MockChatRepository) AddChat(ctx context.Context, serverID, chatID uuid.UUID, name string) error {
	args := m.Called(ctx, serverID, chatID, name)
	return args.Error(0)
}

func (m *MockChatRepository) RemoveChat(ctx context.Context, serverID, chatID uuid.UUID) error {
	args := m.Called(ctx, serverID, chatID)
	return args.Error(0)
}

func (m *MockChatRepository) GetByServer(ctx context.Context, serverID uuid.UUID) ([]*repository.ServerChat, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.ServerChat), args.Error(1)
}

func (m *MockChatRepository) GetByChatID(ctx context.Context, chatID uuid.UUID) (*repository.ServerChat, error) {
	args := m.Called(ctx, chatID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.ServerChat), args.Error(1)
}
