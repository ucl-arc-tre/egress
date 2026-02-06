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
		db, err := rqlite.New(cfg.Rqlite.BaseURL, cfg.Rqlite.Username, cfg.Rqlite.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to initialise rqlite: %w", err)
		}
		return db, nil
	}
	// An unsupported provider shoudld have been failed by Helm
	// So, this is fallback
	return nil, fmt.Errorf("unsupported database provider %s", cfg.Provider)
}
