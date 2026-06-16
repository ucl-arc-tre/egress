package handler

import (
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/db"
	"github.com/ucl-arc-tre/egress/internal/openapi"
	"github.com/ucl-arc-tre/egress/internal/storage"
	"github.com/ucl-arc-tre/egress/internal/types"
)

type Handler struct {
	db      db.Interface
	storage storage.Interface
}

func New() *Handler {
	db, err := db.Provider(config.DBConfig())
	if err != nil {
		panic(err)
	}
	if err := db.Migrate(); err != nil {
		panic(err)
	}
	storage, err := storage.Provider(config.StorageConfig())
	if err != nil {
		panic(err)
	}
	return &Handler{db: db, storage: storage}
}

func (h *Handler) GetProjectIdEvents(ctx *gin.Context, projectId openapi.ProjectIdParam) {
	projectEvents, err := h.db.FileEvents(types.ProjectId(projectId))
	if err != nil {
		setError(ctx, projectId, err, "Failed to get events")
		return
	}

	response := openapi.EventListResponse{}
	for fileId, events := range projectEvents {
		for _, e := range events {
			response = append(response, openapi.Event{
				FileId:      string(fileId),
				Datetime:    e.Time,
				UserId:      string(e.UserId),
				Action:      (*openapi.EventAction)(&e.Action),
				Destination: (*string)(&e.Destination),
				Comment:     &e.Comment,
			})
		}
	}
	sort.SliceStable(response, func(a, b int) bool {
		return response[a].Datetime.Before(response[b].Datetime)
	})

	ctx.JSON(http.StatusOK, response)
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

	location, err := storage.ParseLocation(data.FilesLocation)
	if err != nil {
		setError(ctx, projectId, err, "Failed to parse file location")
		return
	}

	filesMetadata, err := h.storage.List(ctx, *location)
	if err != nil {
		setError(ctx, projectId, err, "Failed to get list of files from storage")
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
	userId := optional(data.UserId)
	if !matchUserIdWithBearerSub(ctx, &userId) {
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
	destApprovals := fileApprovals.ForDestination(types.Destination(data.Destination))
	if numApprovals := len(destApprovals); numApprovals < data.RequiredApprovals {
		ctx.JSON(http.StatusBadRequest, openapi.BadRequest{
			Message: fmt.Sprintf("Required %d approvals for destination %s but only had %d",
				data.RequiredApprovals, string(data.Destination), numApprovals),
		})
		return
	}

	location, err := storage.ParseLocation(data.FilesLocation)
	if err != nil {
		setError(ctx, projectId, err, "Failed to parse file location")
		return
	}

	file, err := h.storage.Get(ctx, *location, types.FileId(fileId))
	if err != nil {
		setError(ctx, projectId, err, "Failed to get file from storage")
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

	comment := optional(data.Comment)
	err = h.db.DownloadFile(
		types.ProjectId(projectId),
		types.FileId(fileId),
		types.UserId(userId),
		types.Destination(data.Destination),
		comment,
	)
	if err != nil {
		setError(ctx, projectId, err, "Failed to write download file event")
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
	if !matchUserIdWithBearerSub(ctx, &data.UserId) {
		return
	}
	comment := optional(data.Comment)
	err := h.db.ApproveFile(
		types.ProjectId(projectId),
		types.FileId(fileId),
		types.UserId(data.UserId),
		types.Destination(data.Destination),
		comment,
	)
	if err != nil {
		setError(ctx, projectId, err, "Failed to approve file")
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handler) PutProjectIdFilesFileIdReject(ctx *gin.Context, projectId openapi.ProjectIdParam, fileId openapi.FileIdParam) {
	data := openapi.RejectFileRequest{}
	if err := ctx.BindJSON(&data); err != nil {
		log.Err(err).Any("projectId", projectId).Msg("Failed to bind reject file json")
		setBadRequest(ctx, "Failed to parse request body")
		return
	}
	if !matchUserIdWithBearerSub(ctx, &data.UserId) {
		return
	}
	comment := optional(data.Comment)
	err := h.db.RejectFile(
		types.ProjectId(projectId),
		types.FileId(fileId),
		types.UserId(data.UserId),
		types.Destination(data.Destination),
		comment,
	)
	if err != nil {
		setError(ctx, projectId, err, "Failed to reject file")
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handler) Ready(ctx *gin.Context) {
	if !h.db.IsReady() {
		ctx.Status(http.StatusServiceUnavailable)
		return
	}
	ctx.Status(http.StatusOK)
}

func (h *Handler) Ping(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
}

// Checks that the user_id matches the `sub` claim from the Bearer
// token (stored as "sub" in the Gin context) when Bearer auth is used.
// If user_id is "" (i.e. optional), then 'sub' is assigned to user_id.
// The check is skipped for Basic auth
func matchUserIdWithBearerSub(ctx *gin.Context, userId *string) bool {
	sub, exists := ctx.Get("sub")
	if !exists { // Exists only for Bearer auth
		return true
	}
	subStr, ok := sub.(string)
	if !ok {
		return false
	}
	if *userId == "" {
		*userId = subStr
		return true
	}
	if *userId == subStr {
		return true
	}
	setBadRequest(ctx, "user_id does not match Bearer token sub")
	return false
}

func optional(param *string) string {
	if param != nil {
		return *param
	}
	return ""
}
