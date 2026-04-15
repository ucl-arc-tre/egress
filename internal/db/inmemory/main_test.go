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
	destination1 := types.Destination("destination-1")

	err := db.ApproveFile(projectId1, fileId1, userId1, destination1)
	assert.NoError(t, err)
	approvals, err := db.FileApprovals(projectId1)
	assert.NoError(t, err)
	assert.Len(t, approvals, 1)
	assert.Equal(t, userId1, approvals[fileId1][0].UserId)
	assert.Equal(t, destination1, approvals[fileId1][0].Destination)

	userId2 := types.UserId("user-2")
	destination2 := types.Destination("destination-2")
	assert.NoError(t, db.ApproveFile(projectId1, fileId1, userId2, destination2))
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

func TestApproveIdempotency(t *testing.T) {
	db := New()
	projectId := types.ProjectId("project-1")
	fileId := types.FileId("file-1")
	userId := types.UserId("user-1")
	destination := types.Destination("destination-1")

	assert.NoError(t, db.ApproveFile(projectId, fileId, userId, destination))
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId, destination))

	// Should have only one approval
	approvals, err := db.FileApprovals(projectId)
	assert.NoError(t, err)
	assert.Len(t, approvals[fileId], 1)
	assert.Equal(t, userId, approvals[fileId][0].UserId)
	assert.Equal(t, destination, approvals[fileId][0].Destination)
}

func TestApproveSameUserMultipleDestinations(t *testing.T) {
	db := New()
	projectId := types.ProjectId("project-1")
	fileId := types.FileId("file-1")
	userId := types.UserId("user-1")
	destination1 := types.Destination("destination-1")
	destination2 := types.Destination("destination-2")

	assert.NoError(t, db.ApproveFile(projectId, fileId, userId, destination1))
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId, destination2))

	// Should have two approvals for the two different destinations
	approvals, err := db.FileApprovals(projectId)
	assert.NoError(t, err)
	assert.Len(t, approvals[fileId], 2)

	destinations := []types.Destination{approvals[fileId][0].Destination, approvals[fileId][1].Destination}
	assert.Contains(t, destinations, destination1)
	assert.Contains(t, destinations, destination2)
}

func TestApproveMultipleDestinationsIdempotency(t *testing.T) {
	db := New()
	projectId := types.ProjectId("project-1")
	fileId := types.FileId("file-1")
	userId := types.UserId("user-1")
	destination1 := types.Destination("destination-1")
	destination2 := types.Destination("destination-2")

	assert.NoError(t, db.ApproveFile(projectId, fileId, userId, destination1))
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId, destination2))

	// Duplicate approvals for both destinations by same user
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId, destination1))
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId, destination2))

	// Should have only two approvals
	approvals, err := db.FileApprovals(projectId)
	assert.NoError(t, err)
	assert.Len(t, approvals[fileId], 2)
}
