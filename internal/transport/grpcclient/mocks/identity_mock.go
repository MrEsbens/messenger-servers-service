package mocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

type MockIdentityClient struct {
	mock.Mock
}

func NewMockIdentityClient(t *testing.T) *MockIdentityClient {
	return &MockIdentityClient{}
}

func (m *MockIdentityClient) UserExists(ctx context.Context, userID string) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockIdentityClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
