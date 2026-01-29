package rqlite

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	rqmig "github.com/golang-migrate/migrate/v4/database/rqlite"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	rq "github.com/rqlite/gorqlite"
)

//go:embed migrations/*.sql
var migrateFS embed.FS

func applyMigrations(conn *rq.Connection) error {
	fsdrv, err := iofs.New(migrateFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed read rqlite migrations: %w", err)
	}

	dbdrv, err := rqmig.WithInstance(conn, &rqmig.Config{})
	if err != nil {
		return fmt.Errorf("failed to acquire rqlite driver for migration: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", fsdrv, "rqlite", dbdrv)
	if err != nil {
		return fmt.Errorf("failed to setup rqlite migration: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to apply rqlite migrations: %w", err)
	}
	return nil
}
