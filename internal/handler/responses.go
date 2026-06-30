package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/openapi"
	"github.com/ucl-arc-tre/egress/internal/types"
)

func setBadRequest(ctx *gin.Context, projectId string, err error, message string) {
	log.Err(err).Str("projectId", projectId).Msg(message)

	ctx.JSON(http.StatusBadRequest, openapi.BadRequest{
		Message: message,
	})
}

func setError(ctx *gin.Context, projectId string, err error, message string) {
	var statusCode int

	if errors.Is(err, types.ErrServer) {
		statusCode = http.StatusInternalServerError
	} else if errors.Is(err, types.ErrInvalidObject) {
		statusCode = http.StatusBadRequest
	} else if errors.Is(err, types.ErrNotFound) {
		statusCode = http.StatusNotFound
	} else {
		statusCode = 520
		err = fmt.Errorf("unknown error: %v", err)
	}
	log.Err(err).Str("projectId", projectId).Msg(message)

	ctx.JSON(statusCode, openapi.ErrorResponse{
		Message: message,
	})
}
