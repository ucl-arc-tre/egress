package inmemory

import (
	"sync"

	"github.com/ucl-arc-tre/egress/internal/types"
)

func New() *DB {
	return &DB{state: map[types.ProjectId]types.ProjectApprovals{}}
}

type DB struct {
	mu    sync.RWMutex
	state map[types.ProjectId]types.ProjectApprovals
}

func (db *DB) ApproveFile(
	projectId types.ProjectId,
	fileId types.FileId,
	userId types.UserId,
	destination types.Destination,
) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.state[projectId]; !exists {
		db.state[projectId] = types.ProjectApprovals{}
	}

	if _, exists := db.state[projectId][fileId]; !exists {
		db.state[projectId][fileId] = types.FileApprovals{}
	}

	// Enforce uniqueness of {file_id, user_id, destination} within a project
	// A duplicate call is treated as idempotent
	for _, existing := range db.state[projectId][fileId] {
		if existing.UserId == userId && existing.Destination == destination {
			return nil
		}
	}

	approval := types.Approval{
		UserId:      userId,
		Destination: destination,
	}
	db.state[projectId][fileId] = append(db.state[projectId][fileId], approval)
	return nil
}

func (db *DB) FileApprovals(projectId types.ProjectId) (types.ProjectApprovals, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	projectApprovals, exists := db.state[projectId]
	if !exists {
		return types.ProjectApprovals{}, nil
	}
	return projectApprovals, nil
}

func (db *DB) Migrate() error {
	// NO-OP for inmemory database
	return nil
}

func (db *DB) IsReady() bool {
	return true
}
