package router

import (
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/config"
)

// Probe for database server readiness
func isDBReady() func() bool {
	db := config.DBConfig()
	return func() bool {
		client := http.Client{}
		req, err := http.NewRequest(http.MethodGet, db.BaseURL+db.ReadinessProbe, nil)
		if err != nil {
			return false
		}
		req.SetBasicAuth(db.Username, db.Password)
		res, err := client.Do(req)
		if err != nil || res.StatusCode != http.StatusOK {
			log.Err(err).Any("HTTP", res.StatusCode).Msg("Failed to probe database readiness")
			return false
		}
		return true
	}
}
