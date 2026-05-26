package rqlite

import (
	"fmt"
	"net/url"
	"time"

	rq "github.com/rqlite/gorqlite"
	"github.com/ucl-arc-tre/egress/internal/types"
)

const (
	datetimeLegacyFormat = time.DateTime // To parse old events timestamped within rqlite
	datetimeSubsecFormat = time.DateTime + ".000"
)

type DB struct {
	conn *rq.Connection
}

func New(baseURL, username, password string) (*DB, error) {
	connURL, err := buildAuthURL(baseURL, username, password)
	if err != nil {
		return nil, fmt.Errorf("[rqlite] failed to build connection URL: %w", err)
	}

	conn, err := rq.Open(connURL)
	if err != nil {
		return nil, fmt.Errorf("[rqlite] failed to open connection: %w", err)
	}

	db := &DB{conn: conn}
	return db, nil
}

func (db *DB) ApproveFile(
	projectId types.ProjectId,
	fileId types.FileId,
	userId types.UserId,
	destination types.Destination,
	comment string,
) error {
	return db.insertEvent(types.EventActionApproval, projectId, fileId, userId, destination, comment)
}

func (db *DB) RejectFile(
	projectId types.ProjectId,
	fileId types.FileId,
	userId types.UserId,
	destination types.Destination,
	comment string,
) error {
	return db.insertEvent(types.EventActionRejection, projectId, fileId, userId, destination, comment)
}

func (db *DB) DownloadFile(
	projectId types.ProjectId,
	fileId types.FileId,
	userId types.UserId,
	destination types.Destination,
	comment string,
) error {
	return db.insertEvent(types.EventActionDownload, projectId, fileId, userId, destination, comment)
}

func (db *DB) FileApprovals(projectId types.ProjectId) (types.ProjectApprovals, error) {
	events, err := db.FileEvents(projectId)
	if err != nil {
		return nil, err
	}
	return events.ProjectApprovals(), nil
}

func (db *DB) FileEvents(projectId types.ProjectId) (types.ProjectEvents, error) {
	sqlFileEvents := `SELECT file_id, user_id, destination, action, comment, created_at FROM events WHERE project_id = ? ORDER BY id ASC`

	stmt := rq.ParameterizedStatement{
		Query:     sqlFileEvents,
		Arguments: []any{projectId},
	}

	qr, operr := db.conn.QueryOneParameterized(stmt)
	err := unifyErrors("[rqlite] failed to execute events query", operr, qr.Err)
	if err != nil {
		return nil, err
	}

	projectEvents := make(types.ProjectEvents)
	for qr.Next() {
		var fileId, userId, destination, action, comment, createdAt string
		if err := qr.Scan(&fileId, &userId, &destination, &action, &comment, &createdAt); err != nil {
			return nil, fmt.Errorf("[rqlite] failed to scan row: %w", err)
		}
		dt, err := parseDatetime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("[rqlite] failed to parse timestamp %q: %w", createdAt, err)
		}
		event := types.Event{
			Time:   dt,
			Action: types.EventAction(action),
			EventDetails: types.EventDetails{
				UserId:      types.UserId(userId),
				Destination: types.Destination(destination),
				Comment:     comment,
			},
		}
		fid := types.FileId(fileId)
		projectEvents[fid] = append(projectEvents[fid], event)
	}
	return projectEvents, nil
}

func (db *DB) IsReady() bool {
	sqlIsReady := `SELECT 1 FROM events LIMIT 1`

	qr, operr := db.conn.QueryOne(sqlIsReady)
	return operr == nil && qr.Err == nil
}

func (db *DB) insertEvent(
	action types.EventAction,
	projectId types.ProjectId,
	fileId types.FileId,
	userId types.UserId,
	destination types.Destination,
	comment string,
) error {
	sqlInsert := `INSERT INTO events (project_id, file_id, user_id, destination, action, comment, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`

	createdAt := time.Now().UTC().Format(datetimeSubsecFormat)
	stmt := rq.ParameterizedStatement{
		Query:     sqlInsert,
		Arguments: []any{projectId, fileId, userId, destination, action, comment, createdAt},
	}

	wr, operr := db.conn.WriteOneParameterized(stmt)
	return unifyErrors("[rqlite] failed to insert event", operr, wr.Err)
}

// Parse datetime strings while accommodating for the non-subsecond
// precision of the default values for 'created_at' column
func parseDatetime(s string) (time.Time, error) {
	if t, err := time.Parse(datetimeSubsecFormat, s); err == nil {
		return t, nil
	}
	return time.Parse(datetimeLegacyFormat, s)
}

func buildAuthURL(baseURL, username, password string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("[rqlite] invalid URL: %w", err)
	}

	u.User = url.UserPassword(username, password)
	return u.String(), nil
}

func unifyErrors(msg string, operr, dberr error) error {
	if operr != nil { // First check for connection errors..
		return types.NewErrServerF("%s: %w", msg, operr)
	}
	if dberr != nil { // ..then check for DB errors
		return types.NewErrServerF("%s: %w", msg, dberr)
	}
	return nil
}
