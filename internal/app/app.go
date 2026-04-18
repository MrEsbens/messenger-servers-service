// internal/app/app.go

package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	redislib "github.com/redis/go-redis/v9"

	"github.com/MrEsbens/messenger-servers-service/internal/config"
	"github.com/MrEsbens/messenger-servers-service/internal/repository"
	repo_redis "github.com/MrEsbens/messenger-servers-service/internal/repository/redis"
	"github.com/MrEsbens/messenger-servers-service/internal/service"
	"github.com/MrEsbens/messenger-servers-service/internal/transport/grpcclient"
	"github.com/MrEsbens/messenger-servers-service/internal/transport/grpcserver"
	_ "github.com/lib/pq"
)

type App struct {
	cfg            *config.Config
	db             *sql.DB
	redisClient    *redislib.Client
	server         *grpcserver.Server
	serverService  service.ServerServiceInterface
	identityClient grpcclient.IdentityClientInterface
	chatsClient    grpcclient.ChatsClientInterface
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	log.Println("🔧 Initializing Servers Service...")

	// ─── Database ────────────────────────────────────────────
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	log.Println("✅ Connected to database")

	var redisClient *redislib.Client
	var cacheRepo repo_redis.CacheRepository

	if cfg.Redis.URL != "" {
		opt, err := redislib.ParseURL(cfg.Redis.URL)
		if err != nil {
			log.Printf("⚠️  Failed to parse Redis URL: %v", err)
		} else {
			redisClient = redislib.NewClient(opt)

			if err := redisClient.Ping(ctx).Err(); err != nil {
				log.Printf("⚠️  Redis connection failed: %v", err)
			} else {
				log.Printf("✅ Connected to Redis at %s", cfg.Redis.URL)
			}

			cacheRepo = repo_redis.NewCacheRepo(redisClient, cfg.Redis.Prefix)
			log.Println("✅ Cache repository initialized")
		}
	} else {
		log.Println("⚠️  Redis URL not configured, caching disabled")
	}

	// ─── Repositories ────────────────────────────────────────
	serverRepo := repository.NewServerRepository(db)
	configRepo := repository.NewConfigRepository(db)
	memberRepo := repository.NewMemberRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	chatRepo := repository.NewChatRepository(db)
	log.Println("✅ Repositories initialized")

	// ─── External Services ───────────────────────────────────

	// Identity Client
	var identityClient grpcclient.IdentityClientInterface
	if cfg.External.IdentityServiceEndpoint != "" {
		client, err := grpcclient.NewIdentityClient(cfg.External.IdentityServiceEndpoint)
		if err != nil {
			log.Printf("⚠️  Failed to create identity client: %v", err)
		} else {
			identityClient = client
			log.Printf("✅ Connected to Identity Service at %s", cfg.External.IdentityServiceEndpoint)
		}
	}

	// Chats Client
	var chatsClient grpcclient.ChatsClientInterface
	if cfg.External.ChatsServiceEndpoint != "" {
		client, err := grpcclient.NewChatsClient(cfg.External.ChatsServiceEndpoint)
		if err != nil {
			log.Printf("⚠️  Failed to create chats client: %v", err)
		} else {
			chatsClient = client
			log.Printf("✅ Connected to Chats Service at %s", cfg.External.ChatsServiceEndpoint)
		}
	}

	// ─── Service ─────────────────────────────────────────────
	serverService := service.NewServerService(
		serverRepo,
		configRepo,
		memberRepo,
		roleRepo,
		chatRepo,
		identityClient,
		chatsClient,
		cacheRepo,
	)
	log.Println("✅ Server service initialized")

	// ─── gRPC Server ─────────────────────────────────────────
	handler := grpcserver.NewHandler(serverService)
	grpcServer := grpcserver.NewServer(cfg.GRPC.Port, handler)

	return &App{
		cfg:            cfg,
		db:             db,
		redisClient:    redisClient,
		server:         grpcServer,
		serverService:  serverService,
		identityClient: identityClient,
		chatsClient:    chatsClient,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.server.Start(ctx)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	log.Println("🛑 Shutting down Servers Service...")

	a.server.Stop()

	// Закрываем Redis клиент (через алиас)
	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			log.Printf("⚠️  Failed to close Redis client: %v", err)
		}
	}

	if a.identityClient != nil {
		_ = a.identityClient.Close()
	}
	if a.chatsClient != nil {
		_ = a.chatsClient.Close()
	}

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}

	log.Println("✅ Servers Service shut down successfully")
	return nil
}

func WaitForSignal() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("📶 Received signal: %v", sig)
		cancel()
	}()

	return ctx
}