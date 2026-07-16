package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/db"
)

type HealthHandler struct {
	db db.Interface
}

func NewHealth() *HealthHandler {
	db, err := db.Provider(config.DBConfig())
	if err != nil {
		panic(err)
	}
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Ready(ctx *gin.Context) { ready(ctx, h.db) }
func (h *HealthHandler) Ping(ctx *gin.Context)  { ping(ctx) }
