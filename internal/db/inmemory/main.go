package inmemory

import (
	"sync"
	"time"

	"github.com/ucl-arc-tre/egress/internal/types"
)

func New() *DB {
	return &DB{state: map[types.ProjectId]types.ProjectEvents{}}
}

type DB struct {
	mu    sync.RWMutex
	state map[types.ProjectId]types.ProjectEvents
}

func (db *DB) ApproveFile(
	projectId types.ProjectId,
	fileId types.FileId,
	userId types.UserId,
	destination types.Destination,
	comment string,
) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.appendEvent(types.EventActionApproval, projectId, fileId, userId, destination, comment)
	return nil
}

func (db *DB) RejectFile(
	projectId types.ProjectId,
	fileId types.FileId,
	userId types.UserId,
	destination types.Destination,
	comment string,
) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.appendEvent(types.EventActionRejection, projectId, fileId, userId, destination, comment)
	return nil
}

func (db *DB) DownloadFile(
	projectId types.ProjectId,
	fileId types.FileId,
	userId types.UserId,
	destination types.Destination,
	comment string,
) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	db.appendEvent(types.EventActionDownload, projectId, fileId, userId, destination, comment)
	return nil
}

func (db *DB) FileApprovals(
	projectId types.ProjectId,
) (types.ProjectApprovals, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	events, exists := db.state[projectId]
	if !exists {
		return types.ProjectApprovals{}, nil
	}
	return events.ProjectApprovals(), nil
}

func (db *DB) FileEvents(
	projectId types.ProjectId,
) (types.ProjectEvents, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	events, exists := db.state[projectId]
	if !exists {
		return types.ProjectEvents{}, nil
	}
	// Events should already be in choronological order as they
	// are appended to FileEvents. Hence, sorting not required
	return events, nil
}

func (db *DB) Migrate() error {
	// NO-OP for inmemory database
	return nil
}

func (db *DB) IsReady() bool {
	return true
}

// Timestamp and append event to the in-memory store
func (db *DB) appendEvent(
	action types.EventAction,
	projectId types.ProjectId,
	fileId types.FileId,
	userId types.UserId,
	destination types.Destination,
	comment string,
) {
	if _, exists := db.state[projectId]; !exists {
		db.state[projectId] = types.ProjectEvents{}
	}
	if _, exists := db.state[projectId][fileId]; !exists {
		db.state[projectId][fileId] = types.FileEvents{}
	}
	event := types.Event{
		Time:   time.Now(),
		Action: action,
		EventDetails: types.EventDetails{
			UserId:      userId,
			Destination: destination,
			Comment:     comment,
		},
	}
	event.Time = time.Now()
	db.state[projectId][fileId] = append(db.state[projectId][fileId], event)
}
