package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MrEsbens/messenger-servers-service/internal/config"
	"github.com/MrEsbens/messenger-servers-service/internal/repository"
	"github.com/MrEsbens/messenger-servers-service/internal/service"
	"github.com/MrEsbens/messenger-servers-service/internal/transport/grpcclient"
	"github.com/MrEsbens/messenger-servers-service/internal/transport/grpcserver"
	_ "github.com/lib/pq"
)

type App struct {
	cfg            *config.Config
	db             *sql.DB
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

	// ─── Repositories ────────────────────────────────────────
	serverRepo := repository.NewServerRepository(db)
	configRepo := repository.NewConfigRepository(db)
	memberRepo := repository.NewMemberRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	moderationRepo := repository.NewModerationRepository(db)
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

	// Moderation Client — STUB пока
	moderationClient := grpcclient.NewStubModerationClient()
	log.Println("✅ Using stub moderation client (all messages allowed)")

	// ─── Service ─────────────────────────────────────────────
	serverService := service.NewServerService(
		serverRepo,
		configRepo,
		memberRepo,
		roleRepo,
		moderationRepo,
		chatRepo,
		identityClient,
		chatsClient,
		moderationClient,
	)
	log.Println("✅ Server service initialized")

	// ─── gRPC Server ─────────────────────────────────────────
	handler := grpcserver.NewHandler(serverService)
	grpcServer := grpcserver.NewServer(cfg.GRPC.Port, handler)

	return &App{
		cfg:            cfg,
		db:             db,
		server:         grpcServer,
		serverService:  serverService,
		identityClient: identityClient,
		chatsClient:    chatsClient,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	// Запускаем gRPC сервер в горутине
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.server.Start(ctx)
	}()

	// Ждём сигнала завершения или ошибки
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	log.Println("🛑 Shutting down Servers Service...")

	// Закрываем gRPC сервер
	a.server.Stop()

	// Закрываем внешние клиенты
	if a.identityClient != nil {
		_ = a.identityClient.Close()
	}
	if a.chatsClient != nil {
		_ = a.chatsClient.Close()
	}
	// moderationClient — stub, закрывать не нужно

	// Закрываем БД
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}

	log.Println("✅ Servers Service shut down successfully")
	return nil
}

// WaitForSignal возвращает context, который отменяется по сигналу OS
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
