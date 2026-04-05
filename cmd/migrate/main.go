package main

import (
	"flag"
	"log"

	"github.com/MrEsbens/messenger-servers-service/internal/config"
	"github.com/MrEsbens/messenger-servers-service/internal/db/migrator"
)

func main() {
	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	cmd := flag.String("cmd", "up", "up | down | force | buckets")
	steps := flag.Int("steps", 1, "steps for down")
	version := flag.Int("version", 1, "version for force")
	flag.Parse()

	m := migrator.New(cfg.Database.URL, cfg.Database.MigrationsPath)

	switch *cmd {
	case "up":
		m.Up()
	case "down":
		m.Down(*steps)
	case "force":
		m.Force(*version)
	default:
		log.Fatalf("unknown command: %s", *cmd)
	}
}
