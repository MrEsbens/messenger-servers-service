package mocks

import (
	"context"
	"testing"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockModerationClient struct {
	mock.Mock
}

func NewMockModerationClient(t *testing.T) *MockModerationClient {
	return &MockModerationClient{}
}

func (m *MockModerationClient) CheckText(ctx context.Context, text string, config *domain.ModerationConfig) (*domain.ModerationResult, error) {
	args := m.Called(ctx, text, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ModerationResult), args.Error(1)
}

func (m *MockModerationClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
