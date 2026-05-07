package db

import "github.com/ucl-arc-tre/egress/internal/types"

type Interface interface {
	ApproveFile(
		projectId types.ProjectId,
		fileId types.FileId,
		userId types.UserId,
		destination types.Destination,
		comment string,
	) error
	RejectFile(
		projectId types.ProjectId,
		fileId types.FileId,
		userId types.UserId,
		destination types.Destination,
		comment string,
	) error
	DownloadFile(
		projectId types.ProjectId,
		fileId types.FileId,
		userId types.UserId,
		destination types.Destination,
		comment string,
	) error
	FileApprovals(projectId types.ProjectId) (types.ProjectApprovals, error)
	FileEvents(projectId types.ProjectId) (types.ProjectEvents, error)

	Migrate() error
	IsReady() bool
}
