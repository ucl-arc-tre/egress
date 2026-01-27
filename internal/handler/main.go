package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/db"
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
	db, err := db.Provider(config.DBConfig())
	if err != nil {
		panic(err)
	}
	return &Handler{db: db, s3: s3.New()}
}

func (h *Handler) GetProjectIdFiles(ctx *gin.Context, projectId openapi.ProjectIdParam) {
	data := openapi.ListFilesRequest{}
	if err := ctx.BindJSON(&data); err != nil {
		log.Err(err).Any("projectId", projectId).Msg("Failed to bind request json")
		setBadRequest(ctx, "Failed to parse request body")
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

	filesMetadata := []types.FileMetadata{}
	switch location.StorageBackendKind() {
	case types.StorageBackendKindS3:
		filesMetadata, err = h.s3.List(ctx, *location)
	default:
		err = types.NewErrInvalidObjectF("unspported storage backend kind")
	}
	if err != nil {
		setError(ctx, projectId, err, "Failed to get list files from storage backend")
		return
	}

	response := openapi.FileListResponse{}
	for _, fileMetadata := range filesMetadata {
		approvals := projectApprovals.FileApprovals(fileMetadata.Id)
		fileMetadata := openapi.MakeFileMetadata(fileMetadata, approvals)
		response = append(response, fileMetadata)
	}

	ctx.JSON(http.StatusOK, response)
}

func (h *Handler) GetProjectIdFilesFileId(ctx *gin.Context, projectId openapi.ProjectIdParam, fileId openapi.FileIdParam) {
	data := openapi.DownloadFileRequest{}
	if err := ctx.BindJSON(&data); err != nil {
		log.Err(err).Any("projectId", projectId).Any("fileId", fileId).Msg("Failed to bind download request json")
		setBadRequest(ctx, "Failed to parse request body")
		return
	}

	projectApprovals, err := h.db.FileApprovals(types.ProjectId(projectId))
	if err != nil {
		setError(ctx, projectId, err, "Failed to get approved files")
		return
	}
	fileApprovals, exists := projectApprovals[types.FileId(fileId)]
	if !exists {
		log.Debug().Any("fileId", fileId).Msg("File had no approvals")
		fileApprovals = types.FileApprovals{}
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

	var file *types.File
	switch location.StorageBackendKind() {
	case types.StorageBackendKindS3:
		file, err = h.s3.Get(ctx, *location, types.FileId(fileId))
	default:
		err = types.NewErrInvalidObjectF("unspported storage backend kind")
	}
	if err != nil {
		setError(ctx, projectId, err, "Failed to get list files from storage")
		return
	}
	defer func() {
		if err := file.Content.Close(); err != nil {
			log.Err(err).Msg("Failed to close stream")
		}
	}()
	if file.Size > int64(data.MaxFileSize) {
		ctx.JSON(http.StatusBadRequest, openapi.BadRequest{
			Message: fmt.Sprintf("Size [%d] was greater than max [%d]", file.Size, data.MaxFileSize),
		})
		return
	}

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Status(http.StatusOK)
	numBytes, err := io.Copy(ctx.Writer, file.Content)
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
		setBadRequest(ctx, "Failed to parse request body")
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
