package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ucl-arc-tre/egress/internal/openapi"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) GetHello(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, openapi.Hello{Message: "world"})
}
