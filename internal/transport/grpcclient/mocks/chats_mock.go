package mocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

type MockChatsClient struct {
	mock.Mock
}

func NewMockChatsClient(t *testing.T) *MockChatsClient {
	return &MockChatsClient{}
}

func (m *MockChatsClient) CreateServerChat(ctx context.Context, serverID, name, createdBy string) (string, error) {
	args := m.Called(ctx, serverID, name, createdBy)
	return args.String(0), args.Error(1)
}

func (m *MockChatsClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
