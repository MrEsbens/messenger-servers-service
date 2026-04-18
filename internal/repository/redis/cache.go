package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// ModerationConfigKeyPattern — шаблон ключа для кэша конфига модерации
	// Пример: "srv:moderation:config:550e8400-e29b-41d4-a716-446655440000"
	ModerationConfigKeyPattern = "%smoderation:config:%s"
)

type CacheRepository interface {
	InvalidateModerationConfig(ctx context.Context, serverID uuid.UUID) error
}

type cacheRepo struct {
	client *redis.Client
	prefix string
}

func NewCacheRepo(client *redis.Client, prefix string) CacheRepository {
	return &cacheRepo{
		client: client,
		prefix: prefix,
	}
}

func (r *cacheRepo) InvalidateModerationConfig(ctx context.Context, serverID uuid.UUID) error {
	key := fmt.Sprintf(ModerationConfigKeyPattern, r.prefix, serverID.String())

	// DEL возвращает количество удалённых ключей (0 или 1)
	// Ошибку возвращаем только если реальная проблема с Redis
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}

	return nil
}
