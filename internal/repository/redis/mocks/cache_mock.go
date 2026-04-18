package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) InvalidateModerationConfig(ctx context.Context, serverID uuid.UUID) error {
	ret := m.Called(ctx, serverID)

	if len(ret) == 0 {
		panic("no return value specified for InvalidateModerationConfig")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uuid.UUID) error); ok {
		r0 = rf(ctx, serverID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockCacheRepository creates a new instance of MockCacheRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockCacheRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockCacheRepository {
	mock := &MockCacheRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}