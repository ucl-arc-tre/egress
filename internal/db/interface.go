package db

import "github.com/ucl-arc-tre/egress/internal/types"

type Interface interface {
	ApproveFile(projectId types.ProjectId, fileId types.FileId, userId types.UserId) error
	FileApprovals(projectId types.ProjectId) (types.ProjectApprovals, error)
	IsReady() bool
}
