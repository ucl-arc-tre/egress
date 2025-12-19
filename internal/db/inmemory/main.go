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

func (db *DB) ApproveFile(projectId types.ProjectId, fileId types.FileId, userId types.UserId) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.state[projectId]; !exists {
		db.state[projectId] = types.ProjectApprovals{}
	}

	if _, exists := db.state[projectId][fileId]; !exists {
		db.state[projectId][fileId] = types.FileApprovals{}
	}

	db.state[projectId][fileId] = append(db.state[projectId][fileId], userId)
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
