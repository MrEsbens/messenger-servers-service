package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/MrEsbens/messenger-servers-service/internal/config"
	_ "github.com/lib/pq"
)

type Postgres struct {
	DB *sql.DB
}

func NewPostgres(cfg config.DatabaseConfig) *Postgres {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		log.Fatalf("❌ Failed to open database connection: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("SET search_path TO %s", cfg.Schema))
	if err != nil {
		log.Fatalf("❌ Failed to set search_path to %s: %v", cfg.Schema, err)
	}

	return &Postgres{DB: db}
}

func (p *Postgres) Close() error {
	if p.DB != nil {
		log.Println("🔌 Closing database connection...")
		return p.DB.Close()
	}
	return nil
}
