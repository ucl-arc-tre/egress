package rqlite

import (
	"fmt"
	"net/url"

	rq "github.com/rqlite/gorqlite"
	"github.com/ucl-arc-tre/egress/internal/types"
)

type DB struct {
	conn *rq.Connection
}

func New(baseURL, username, password string) (*DB, error) {
	connURL, err := buildURLWithAuth(baseURL, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to build connection URL: %w", err)
	}

	conn, err := rq.Open(connURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open rqlite connection: %w", err)
	}

	db := &DB{conn: conn}
	// Initialize schema
	// TODO: Need to do this as part of an external DB migration approach
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	return db, nil
}

func (db *DB) ApproveFile(projectId types.ProjectId, fileId types.FileId, userId types.UserId) error {
	sql := `INSERT INTO file_approvals (project_id, file_id, user_id) VALUES (?, ?, ?)`

	stmt := rq.ParameterizedStatement{
		Query:     sql,
		Arguments: []any{string(projectId), string(fileId), string(userId)},
	}

	wr, err := db.conn.WriteOneParameterized(stmt)
	if err != nil {
		return fmt.Errorf("failed to execute insert: %w", err)
	}
	if wr.Err != nil {
		return fmt.Errorf("failed to execute insert: %w", wr.Err)
	}
	return nil
}

func (db *DB) FileApprovals(projectId types.ProjectId) (types.ProjectApprovals, error) {
	sql := `SELECT file_id, user_id FROM file_approvals WHERE project_id = ? ORDER BY file_id, id`

	stmt := rq.ParameterizedStatement{
		Query:     sql,
		Arguments: []any{string(projectId)},
	}

	qr, err := db.conn.QueryOneParameterized(stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	if qr.Err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", qr.Err)
	}

	// Make ProjectApprovals map from query results
	approvals := make(types.ProjectApprovals)
	for qr.Next() {
		var fileIdStr, userIdStr string
		if err := qr.Scan(&fileIdStr, &userIdStr); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		fileId := types.FileId(fileIdStr)
		userId := types.UserId(userIdStr)

		// Append userId to file's approval list
		if _, exists := approvals[fileId]; !exists {
			approvals[fileId] = types.FileApprovals{}
		}
		approvals[fileId] = append(approvals[fileId], userId)
	}
	return approvals, nil
}

func buildURLWithAuth(baseURL, username, password string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid rqlite URL: %w", err)
	}
	if username == "" || password == "" {
		return "", fmt.Errorf("incomplete rqlite credentials")
	}
	u.User = url.UserPassword(username, password)
	return u.String(), nil
}

func (db *DB) initSchema() error {
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS file_approvals (
    	id INTEGER PRIMARY KEY AUTOINCREMENT,
    	project_id TEXT NOT NULL,
    	file_id TEXT NOT NULL,
    	user_id TEXT NOT NULL,
    	created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`
	createIndexSQL := `CREATE INDEX IF NOT EXISTS idx_project_file ON file_approvals(project_id, file_id);`

	// Execute CREATE TABLE
	wr, err := db.conn.WriteOne(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	if wr.Err != nil {
		return fmt.Errorf("failed to create table: %w", wr.Err)
	}

	// Execute CREATE INDEX
	wr, err = db.conn.WriteOne(createIndexSQL)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	if wr.Err != nil {
		return fmt.Errorf("failed to create index: %w", wr.Err)
	}

	return nil
}
