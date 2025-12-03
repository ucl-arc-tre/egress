package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func New() *gin.Engine {
	router := gin.Default()
	router.Group("/ping").GET("", ping)
	router.Group("/ready").GET("", ready)
	return router
}

func ping(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func ready(ctx *gin.Context) {
	ctx.Status(http.StatusOK)
}
