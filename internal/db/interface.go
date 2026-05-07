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
	FileApprovals(projectId types.ProjectId) (types.ProjectApprovals, error)
	Migrate() error
	IsReady() bool
}
