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

func (h *Handler) GetProjectIdFiles(ctx *gin.Context, projectId openapi.ProjectIdParam) {
	ctx.Status(http.StatusNotImplemented)
}

func (h *Handler) GetProjectIdFilesFileId(ctx *gin.Context, projectId openapi.ProjectIdParam, fileId openapi.FileIdParam) {
	ctx.Status(http.StatusNotImplemented)
}

func (h *Handler) PutProjectIdFilesFileIdApprove(ctx *gin.Context, projectId openapi.ProjectIdParam, fileId openapi.FileIdParam) {
	ctx.Status(http.StatusNotImplemented)
}
