package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ucl-arc-tre/egress/internal/openapi"
)

func setInvalidObject(ctx *gin.Context, message string) {
	ctx.JSON(http.StatusBadRequest, openapi.BadRequest{
		Message: fmt.Sprintf("Invalid object. %s", message),
	})
}
