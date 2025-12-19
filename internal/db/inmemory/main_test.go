package inmemory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/types"
)

func TestApproveThenList(t *testing.T) {
	db := New()
	projectId1 := types.ProjectId("project-1")
	fileId1 := types.FileId("file-1")
	userId1 := types.UserId("user-1")

	err := db.ApproveFile(projectId1, fileId1, userId1)
	assert.NoError(t, err)
	approvals, err := db.FileApprovals(projectId1)
	assert.NoError(t, err)
	assert.Len(t, approvals, 1)
	assert.Equal(t, userId1, approvals[fileId1][0])

	userId2 := types.UserId("user-2")
	assert.NoError(t, db.ApproveFile(projectId1, fileId1, userId2))
	approvals, err = db.FileApprovals(projectId1)
	assert.NoError(t, err)
	assert.Len(t, approvals[fileId1], 2)
}

func TestListNoApprovals(t *testing.T) {
	db := New()
	projectId1 := types.ProjectId("project-1")

	approvals, err := db.FileApprovals(projectId1)
	assert.NoError(t, err)
	assert.Len(t, approvals, 0)
}
