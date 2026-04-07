package grpcclient

import (
	"context"

	"github.com/MrEsbens/messenger-servers-service/internal/domain"
)

// ModerationClientInterface — интерфейс для Moderation Service
// Сейчас — заглушка, потом заменим на реальный gRPC-клиент
type ModerationClientInterface interface {
	CheckText(ctx context.Context, text string, config *domain.ModerationConfig) (*domain.ModerationResult, error)
	Close() error
}

// StubModerationClient — заглушка для разработки
// Все сообщения пропускает, нарушений не находит
type StubModerationClient struct{}

// NewStubModerationClient создаёт stub-клиент
func NewStubModerationClient() *StubModerationClient {
	return &StubModerationClient{}
}

func (c *StubModerationClient) CheckText(ctx context.Context, text string, config *domain.ModerationConfig) (*domain.ModerationResult, error) {
	// 🔧 Stub: все сообщения разрешены (fail-open)
	// Потом здесь будет реальный gRPC-вызов к Moderation Service
	return &domain.ModerationResult{
		Allowed:    true,
		Violations: []domain.Violation{},
		Fallback:   false,
	}, nil
}

func (c *StubModerationClient) Close() error {
	return nil
}

// ───────────────────────────────────────────────────────────
// Заготовка для будущего реального клиента (чтобы не забыть)
// ───────────────────────────────────────────────────────────

/*
// RealModerationClient — реальный gRPC-клиент (будет потом)
type RealModerationClient struct {
	conn   *grpc.ClientConn
	client moderationv1.ModerationServiceClient
}

func NewRealModerationClient(endpoint string) (*RealModerationClient, error) {
	// ... gRPC подключение ...
}

func (c *RealModerationClient) CheckText(ctx context.Context, text string, config *domain.ModerationConfig) (*domain.ModerationResult, error) {
	// ... gRPC вызов к Moderation Service ...
}

func (c *RealModerationClient) Close() error {
	// ... закрытие соединения ...
}
*/
