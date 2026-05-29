package inmemory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/types"
)

const (
	projectId       = types.ProjectId("project-1")
	fileId          = types.FileId("file-1")
	userId1         = types.UserId("user-1")
	userId2         = types.UserId("user-2")
	destTrusted     = types.Destination("trusted")
	destPublic      = types.Destination("world")
	commentApprove1 = "lovely"
	commentApprove2 = "good"
	commentReject   = "bad"
	commentDownload = "results"
)

func TestApproveThenList(t *testing.T) {
	db := New()

	err := db.ApproveFile(projectId, fileId, userId1, destTrusted, commentApprove1)
	assert.NoError(t, err)
	approvals, err := db.FileApprovals(projectId)
	assert.NoError(t, err)
	assert.Len(t, approvals, 1)
	assert.Equal(t, userId1, approvals[fileId][0].UserId)
	assert.Equal(t, destTrusted, approvals[fileId][0].Destination)

	assert.NoError(t, db.ApproveFile(projectId, fileId, userId2, destPublic, commentApprove2))
	approvals, err = db.FileApprovals(projectId)
	assert.NoError(t, err)
	assert.Len(t, approvals[fileId], 2)
}

func TestListNoApprovals(t *testing.T) {
	db := New()

	approvals, err := db.FileApprovals(projectId)
	assert.NoError(t, err)
	assert.Len(t, approvals, 0)
}

func TestMultipleApprovals(t *testing.T) {
	db := New()

	assert.NoError(t, db.ApproveFile(projectId, fileId, userId1, destTrusted, commentApprove1))
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId1, destTrusted, commentApprove2))

	// Approvals deduped on {userId,destination}, so only 1 approval returned
	approvals, err := db.FileApprovals(projectId)
	assert.NoError(t, err)
	assert.Len(t, approvals[fileId], 1)

	// Comment of first approval returned
	assert.Equal(t, userId1, approvals[fileId][0].UserId)
	assert.Equal(t, destTrusted, approvals[fileId][0].Destination)
	assert.Equal(t, commentApprove1, approvals[fileId][0].Comment)
}

func TestApproveToMultipleDestinations(t *testing.T) {
	db := New()

	assert.NoError(t, db.ApproveFile(projectId, fileId, userId1, destTrusted, commentApprove1))
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId1, destPublic, commentApprove2))

	// Should have two approvals for the two different destinations
	approvals, err := db.FileApprovals(projectId)
	assert.NoError(t, err)
	assert.Len(t, approvals[fileId], 2)

	destinations := []types.Destination{approvals[fileId][0].Destination, approvals[fileId][1].Destination}
	assert.Contains(t, destinations, destTrusted)
	assert.Contains(t, destinations, destPublic)

	comments := []string{approvals[fileId][0].Comment, approvals[fileId][1].Comment}
	assert.Equal(t, commentApprove1, comments[0])
	assert.Equal(t, commentApprove2, comments[1])
}

func TestMultipleApprovalsToMultipleDestinations(t *testing.T) {
	db := New()

	// Approvals for 2 different destinations
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId1, destTrusted, commentApprove1))
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId1, destPublic, commentApprove2))

	// Duplicate approvals for both destinations
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId1, destTrusted, commentApprove1))
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId1, destPublic, commentApprove2))

	// Approvals deduped on {userId,destination}, so only 2 approval returned
	approvals, err := db.FileApprovals(projectId)
	assert.NoError(t, err)
	assert.Len(t, approvals[fileId], 2)
}

func TestRejectThenList(t *testing.T) {
	db := New()

	assert.NoError(t, db.RejectFile(projectId, fileId, userId1, destTrusted, commentReject))

	events, err := db.FileEvents(projectId)
	assert.NoError(t, err)
	assert.Len(t, events[fileId], 1)
	assert.Equal(t, userId1, events[fileId][0].UserId)
	assert.Equal(t, commentReject, events[fileId][0].Comment)
	assert.Equal(t, types.EventActionRejection, events[fileId][0].Action)
}

func TestApproveThenReject(t *testing.T) {
	db := New()

	// Approve and then reject same file
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId1, destTrusted, commentApprove1))
	assert.NoError(t, db.RejectFile(projectId, fileId, userId1, destTrusted, commentReject))

	// Reject cancels prior approval, so no approvals
	approvals, err := db.FileApprovals(projectId)
	assert.NoError(t, err)
	assert.Len(t, approvals[fileId], 0)

	// However, there must be 2 events, approval and rejection, in that order
	events, err := db.FileEvents(projectId)
	assert.Len(t, events[fileId], 2)
	assert.NoError(t, err)
	assert.Equal(t, types.EventActionApproval, events[fileId][0].Action)
	assert.Equal(t, types.EventActionRejection, events[fileId][1].Action)
}

func TestDownloadThenList(t *testing.T) {
	db := New()

	assert.NoError(t, db.DownloadFile(projectId, fileId, userId1, destTrusted, commentDownload))

	events, err := db.FileEvents(projectId)
	assert.NoError(t, err)
	assert.Len(t, events[fileId], 1)
	assert.Equal(t, userId1, events[fileId][0].UserId)
	assert.Equal(t, destTrusted, events[fileId][0].Destination)
	assert.Equal(t, types.EventActionDownload, events[fileId][0].Action)
	assert.Equal(t, commentDownload, events[fileId][0].Comment)
}

func TestListEvents(t *testing.T) {
	db := New()

	// Add three events
	assert.NoError(t, db.RejectFile(projectId, fileId, userId1, destTrusted, commentApprove1))
	assert.NoError(t, db.ApproveFile(projectId, fileId, userId1, destTrusted, commentReject))
	assert.NoError(t, db.DownloadFile(projectId, fileId, userId1, destTrusted, commentDownload))

	events, err := db.FileEvents(projectId)
	assert.NoError(t, err)
	assert.Len(t, events[fileId], 3)
	assert.Equal(t, types.EventActionRejection, events[fileId][0].Action)
	assert.Equal(t, types.EventActionApproval, events[fileId][1].Action)
	assert.Equal(t, types.EventActionDownload, events[fileId][2].Action)
}
