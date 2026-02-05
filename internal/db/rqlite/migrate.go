package rqlite

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	rqmig "github.com/golang-migrate/migrate/v4/database/rqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrateFS embed.FS

func (db *DB) Migrate() error {
	fsdrv, err := iofs.New(migrateFS, "migrations")
	if err != nil {
		return fmt.Errorf("[rqlite] failed read migrations: %w", err)
	}

	dbdrv, err := rqmig.WithInstance(db.conn, &rqmig.Config{})
	if err != nil {
		return fmt.Errorf("[rqlite] failed to acquire driver for migration: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", fsdrv, "rqlite", dbdrv)
	if err != nil {
		return fmt.Errorf("[rqlite] failed to initialise migration: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange && err != migrate.ErrNilVersion {
		return fmt.Errorf("[rqlite] failed to apply migrations: %w", err)
	}
	return nil
}
