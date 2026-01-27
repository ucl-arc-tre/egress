package db

import (
	"fmt"

	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/db/inmemory"
	"github.com/ucl-arc-tre/egress/internal/db/rqlite"
)

type DBProvider string

const (
	DBProviderInMemory = DBProvider("inmemory")
	DBProviderRqlite   = DBProvider("rqlite")
)

func Provider(cfg config.DBConfigBundle) (Interface, error) {
	switch DBProvider(cfg.Provider) {
	case DBProviderInMemory:
		return inmemory.New(), nil

	case DBProviderRqlite:
		db, err := rqlite.New(cfg.BaseURL, cfg.Username, cfg.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to initialise rqlite: %w", err)
		}
		return db, nil

	default:
		return nil, fmt.Errorf("unsupported database provider %s", cfg.Provider)
	}
}
