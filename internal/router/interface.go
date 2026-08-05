package router

import "github.com/gin-gonic/gin"

type HandlerInterface interface {
	Ping(ctx *gin.Context)
	Ready(ctx *gin.Context)
}
