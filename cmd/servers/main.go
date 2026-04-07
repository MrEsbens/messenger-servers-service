package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/MrEsbens/messenger-servers-service/internal/app"
	"github.com/MrEsbens/messenger-servers-service/internal/config"
)

func main() {
	// ─── Load Config ─────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}
	log.Println("✅ Config loaded")

	// ─── Initialize App ──────────────────────────────────────
	ctx := app.WaitForSignal()
	appInstance, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("❌ Failed to initialize app: %v", err)
	}

	// ─── Run App ─────────────────────────────────────────────
	go func() {
		if err := appInstance.Run(ctx); err != nil {
			log.Printf("❌ App error: %v", err)
		}
	}()

	// ─── Wait for Shutdown ───────────────────────────────────
	<-ctx.Done()

	// ─── Graceful Shutdown ───────────────────────────────────
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := appInstance.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️  Shutdown error: %v", err)
		os.Exit(1)
	}

	log.Println("👋 Servers Service stopped")
}
