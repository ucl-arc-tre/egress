package router

import (
	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/db"
)

// Probe database server readiness
func isDBReady() func() bool {
	cfg := config.DBConfig()
	return func() bool {
		db, err := db.Provider(cfg)
		if err != nil {
			return false
		}
		return db.IsReady()
	}
}
