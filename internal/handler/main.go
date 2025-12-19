package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/db"
	"github.com/ucl-arc-tre/egress/internal/db/inmemory"
	"github.com/ucl-arc-tre/egress/internal/openapi"
	"github.com/ucl-arc-tre/egress/internal/types"
)

type Handler struct {
	db db.Interface
}

func New() *Handler {
	return &Handler{db: inmemory.New()}
}

func (h *Handler) GetProjectIdFiles(ctx *gin.Context, projectId openapi.ProjectIdParam) {
	ctx.Status(http.StatusNotImplemented)
}

func (h *Handler) GetProjectIdFilesFileId(ctx *gin.Context, projectId openapi.ProjectIdParam, fileId openapi.FileIdParam) {
	data := openapi.DownloadFileRequest{}
	if err := ctx.BindJSON(&data); err != nil {
		log.Err(err).Any("projectId", projectId).Any("fileId", fileId).Msg("Failed to bind download request json")
		ctx.JSON(http.StatusBadRequest, openapi.BadRequest{
			Message: "Invalid object",
		})
		return
	}

	projectApprovals, err := h.db.FileApprovals(types.ProjectId(projectId))
	if err != nil {
		log.Err(err).Any("projectId", projectId).Msg("Failed to get approved files")
		ctx.Status(http.StatusInternalServerError)
		return
	}
	fileApprovals, exists := projectApprovals[types.FileId(fileId)]
	if !exists {
		ctx.JSON(http.StatusNotFound, openapi.NotFound{})
		return
	}
	if numApprovals := len(fileApprovals); numApprovals < data.RequiredApprovals {
		ctx.JSON(http.StatusBadRequest, openapi.BadRequest{
			Message: fmt.Sprintf("Required %d approvals but only had %d", data.RequiredApprovals, numApprovals),
		})
		return
	}

	// todo - stream file

	ctx.Status(http.StatusNotImplemented)
}

func (h *Handler) PutProjectIdFilesFileIdApprove(ctx *gin.Context, projectId openapi.ProjectIdParam, fileId openapi.FileIdParam) {
	data := openapi.ApproveFileRequest{}
	if err := ctx.BindJSON(&data); err != nil {
		log.Err(err).Any("projectId", projectId).Msg("Failed to bind approve file json")
		ctx.JSON(http.StatusBadRequest, openapi.BadRequest{
			Message: "Invalid object",
		})
		return
	}
	err := h.db.ApproveFile(
		types.ProjectId(projectId),
		types.FileId(fileId),
		types.UserId(data.UserId),
	)
	if err != nil {
		log.Err(err).Any("projectId", projectId).Msg("Failed to approve file")
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.Status(http.StatusNoContent)
}
