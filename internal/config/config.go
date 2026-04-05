package config

import (
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	GRPC       GRPCConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	Moderation ModerationConfig
	External   ExternalServicesConfig
}

type GRPCConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL            string
	Schema         string
	MigrationsPath string
}

type RedisConfig struct {
	URL    string
	Prefix string
}

type ModerationConfig struct {
	ServiceEndpoint string
	Timeout         time.Duration

	CacheEnabled bool
	CacheTTL     time.Duration
}

type ExternalServicesConfig struct {
	IdentityServiceEndpoint string
	ChatsServiceEndpoint    string
	FileServiceEndpoint     string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	modTimeoutSec := getEnvInt("MODERATION_ML_TIMEOUT_SEC", 5)
	cacheTTLMin := getEnvInt("MODERATION_CACHE_TTL_MIN", 60)

	return &Config{
		GRPC: GRPCConfig{
			Port: getEnv("GRPC_PORT", "50054"),
		},
		Database: DatabaseConfig{
			URL:            getEnvRequired("DATABASE_URL"),
			Schema:         getEnv("DB_SCHEMA", "servers"),
			MigrationsPath: getEnv("MIGRATIONS_PATH", "internal/db/migrations"),
		},
		Redis: RedisConfig{
			URL:    getEnv("REDIS_URL", "redis://localhost:6379"),
			Prefix: getEnv("REDIS_KEY_PREFIX", "srv:"),
		},
		Moderation: ModerationConfig{
			ServiceEndpoint: getEnv("MODERATION_SERVICE_ENDPOINT", "http://localhost:8000"),
			Timeout:         time.Duration(modTimeoutSec) * time.Second,
			CacheEnabled:    getEnv("MODERATION_CACHE_ENABLED", "true") == "true",
			CacheTTL:        time.Duration(cacheTTLMin) * time.Minute,
		},
		External: ExternalServicesConfig{
			IdentityServiceEndpoint: getEnv("IDENTITY_SERVICE_ENDPOINT", ""),
			ChatsServiceEndpoint:    getEnv("CHATS_SERVICE_ENDPOINT", ""),
			FileServiceEndpoint:     getEnv("FILE_SERVICE_ENDPOINT", ""),
		},
	}, nil
}
