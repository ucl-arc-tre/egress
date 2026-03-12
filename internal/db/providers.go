package db

import (
	"fmt"

	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/db/inmemory"
	"github.com/ucl-arc-tre/egress/internal/db/rqlite"
	"github.com/ucl-arc-tre/egress/internal/types"
)

func Provider(cfg config.DBConfigBundle) (Interface, error) {
	switch types.DBProvider(cfg.Provider) {
	case types.DBProviderInMemory:
		return inmemory.New(), nil

	case types.DBProviderRqlite:
		db, err := rqlite.New(cfg.Rqlite.BaseURL, cfg.Rqlite.Username, cfg.Rqlite.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to initialise rqlite: %w", err)
		}
		return db, nil
	}
	// An unsupported provider should have been failed by Helm
	// So, this is fallback
	return nil, fmt.Errorf("unsupported database provider %s", cfg.Provider)
}
