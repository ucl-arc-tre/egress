package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/db"
	"github.com/ucl-arc-tre/egress/internal/db/inmemory"
	"github.com/ucl-arc-tre/egress/internal/openapi"
	"github.com/ucl-arc-tre/egress/internal/storage"
	"github.com/ucl-arc-tre/egress/internal/storage/s3"
	"github.com/ucl-arc-tre/egress/internal/types"
)

type Handler struct {
	db db.Interface
	s3 storage.Interface
}

func New() *Handler {
	return &Handler{db: inmemory.New(), s3: s3.New()}
}

func (h *Handler) GetProjectIdFiles(ctx *gin.Context, projectId openapi.ProjectIdParam) {
	data := openapi.ListFilesRequest{}
	if err := ctx.BindJSON(&data); err != nil {
		log.Err(err).Any("projectId", projectId).Msg("Failed to bind request json")
		setInvalidObject(ctx, "Failed to parse request body")
		return
	}

	projectApprovals, err := h.db.FileApprovals(types.ProjectId(projectId))
	if err != nil {
		setError(ctx, projectId, err, "Failed to get file approvals")
		return
	}

	location, err := storage.ParseLocation(data.FileLocation)
	if err != nil {
		setError(ctx, projectId, err, "Failed to parse file location")
		return
	}

	objectsMeta := []types.ObjectMeta{}
	switch location.StorageBackendKind() {
	case types.StorageBackendKindS3:
		objectsMeta, err = h.s3.List(ctx, *location)
	default:
		err = errors.New("unspported storage backend kind")
	}
	if err != nil {
		setError(ctx, projectId, err, "Failed to get list objects from storage backend")
		return
	}

	response := openapi.FileListResponse{}
	for _, objectMeta := range objectsMeta {
		approvals := projectApprovals.FileApprovals(objectMeta.Id)
		fileMetadata := openapi.MakeFileMetadata(objectMeta, approvals)
		response = append(response, fileMetadata)
	}

	ctx.JSON(http.StatusOK, response)
}

func (h *Handler) GetProjectIdFilesFileId(ctx *gin.Context, projectId openapi.ProjectIdParam, fileId openapi.FileIdParam) {
	data := openapi.DownloadFileRequest{}
	if err := ctx.BindJSON(&data); err != nil {
		log.Err(err).Any("projectId", projectId).Any("fileId", fileId).Msg("Failed to bind download request json")
		setInvalidObject(ctx, "Failed to parse request body")
		return
	}

	projectApprovals, err := h.db.FileApprovals(types.ProjectId(projectId))
	if err != nil {
		setError(ctx, projectId, err, "Failed to get approved files")
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

	location, err := storage.ParseLocation(data.FilesLocation)
	if err != nil {
		setError(ctx, projectId, err, "Failed to parse file location")
		return
	}

	var object *types.Object
	switch location.StorageBackendKind() {
	case types.StorageBackendKindS3:
		object, err = h.s3.Get(ctx, *location, types.FileId(fileId))
	default:
		err = errors.New("unspported storage backend kind")
	}
	if err != nil {
		setError(ctx, projectId, err, "Failed to get list objects from storage")
		return
	}
	defer func() {
		if err := object.Content.Close(); err != nil {
			log.Err(err).Msg("Failed to close stream")
		}
	}()
	if object.Size > int64(data.MaxFileSize) {
		ctx.JSON(http.StatusBadRequest, openapi.BadRequest{
			Message: fmt.Sprintf("Size [%d] was greater than max [%d]", object.Size, data.MaxFileSize),
		})
		return
	}

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Status(http.StatusOK)
	numBytes, err := io.Copy(ctx.Writer, object.Content)
	if err != nil {
		log.Err(err).
			Any("projectId", projectId).
			Int64("numBytes", numBytes).
			Msg("Failed to copy stream")
	}
}

func (h *Handler) PutProjectIdFilesFileIdApprove(ctx *gin.Context, projectId openapi.ProjectIdParam, fileId openapi.FileIdParam) {
	data := openapi.ApproveFileRequest{}
	if err := ctx.BindJSON(&data); err != nil {
		log.Err(err).Any("projectId", projectId).Msg("Failed to bind approve file json")
		setInvalidObject(ctx, "Failed to parse request body")
		return
	}
	err := h.db.ApproveFile(
		types.ProjectId(projectId),
		types.FileId(fileId),
		types.UserId(data.UserId),
	)
	if err != nil {
		setError(ctx, projectId, err, "Failed to approve file")
		return
	}
	ctx.Status(http.StatusNoContent)
}
