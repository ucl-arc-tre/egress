package router

import (
	"github.com/gin-gonic/gin"
)

func New(h HandlerInterface) *gin.Engine {
	router := gin.Default()
	router.Group("/ping").GET("", h.Ping)
	router.Group("/ready").GET("", h.Ready)
	return router
}
