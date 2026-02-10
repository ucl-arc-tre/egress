package router

import (
	"github.com/gin-gonic/gin"
	"github.com/ucl-arc-tre/egress/internal/handler"
)

func New(h *handler.Handler) *gin.Engine {
	router := gin.Default()
	router.Group("/ping").GET("", h.Ping)
	router.Group("/ready").GET("", h.Ready)
	return router
}
