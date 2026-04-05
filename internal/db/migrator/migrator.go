package migrator

import (
    "fmt"
    "log"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

type Migrator struct {
    dbURL string
    path  string
}

func New(dbURL, path string) *Migrator {
    return &Migrator{dbURL: dbURL, path: path}
}

func (m *Migrator) Up() {
    mig, err := migrate.New(fmt.Sprintf("file://%s", m.path), m.dbURL)
    if err != nil {
        log.Fatalf("failed to init migrate: %v", err)
    }
    if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
        log.Fatalf("failed to apply up migrations: %v", err)
    }
    log.Println("Migrations applied successfully")
}

func (m *Migrator) Down(steps int) {
    mig, err := migrate.New(fmt.Sprintf("file://%s", m.path), m.dbURL)
    if err != nil {
        log.Fatalf("failed to init migrate: %v", err)
    }
    if err := mig.Steps(-steps); err != nil && err != migrate.ErrNoChange {
        log.Fatalf("failed to apply down migrations: %v", err)
    }
    log.Println("Down migrations applied successfully")
}

func (m *Migrator) Force(version int) {
    mig, err := migrate.New(fmt.Sprintf("file://%s", m.path), m.dbURL)
    if err != nil {
        log.Fatalf("failed to init migrate: %v", err)
    }
    if err := mig.Force(version); err != nil {
        log.Fatalf("failed to force version: %v", err)
    }
    log.Printf("Forced migration version to %d\n", version)
}
