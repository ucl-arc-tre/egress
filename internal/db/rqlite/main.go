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

func (db *DB) ApproveFile(projectId types.ProjectId, fileId types.FileId, userId types.UserId) error {
	sqlApproveFile := `INSERT OR IGNORE INTO file_approvals (project_id, file_id, user_id) VALUES (?, ?, ?)`

	stmt := rq.ParameterizedStatement{
		Query:     sqlApproveFile,
		Arguments: []any{projectId, fileId, userId},
	}

	wr, operr := db.conn.WriteOneParameterized(stmt)
	err := unifyErrors("[rqlite] failed to execute approve file insert", operr, wr.Err)

	return err
}

func (db *DB) FileApprovals(projectId types.ProjectId) (types.ProjectApprovals, error) {
	sqlFileApprovals := `SELECT file_id, user_id FROM file_approvals WHERE project_id = ? ORDER BY file_id`

	stmt := rq.ParameterizedStatement{
		Query:     sqlFileApprovals,
		Arguments: []any{projectId},
	}

	qr, operr := db.conn.QueryOneParameterized(stmt)
	err := unifyErrors("[rqlite] failed to execute approvals query", operr, qr.Err)
	if err != nil {
		return nil, err
	}

	// Make ProjectApprovals map from query results
	approvals := make(types.ProjectApprovals)
	for qr.Next() {
		var fileIdStr, userIdStr string
		if err := qr.Scan(&fileIdStr, &userIdStr); err != nil {
			return nil, fmt.Errorf("[rqlite] failed to scan row: %w", err)
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

func (db *DB) IsReady() bool {
	sqlIsReady := `SELECT 1 FROM file_approvals LIMIT 1`

	qr, operr := db.conn.QueryOne(sqlIsReady)

	return operr == nil && qr.Err == nil
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
