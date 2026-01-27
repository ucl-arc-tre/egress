package rqlite

import (
	"fmt"
	"net/url"

	rq "github.com/rqlite/gorqlite"
	"github.com/ucl-arc-tre/egress/internal/types"
)

const (
	sqlApproveFile   = `INSERT INTO file_approvals (project_id, file_id, user_id) VALUES (?, ?, ?)`
	sqlFileApprovals = `SELECT file_id, user_id FROM file_approvals WHERE project_id = ? ORDER BY file_id, id`
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

	// Assume schema has been created prior
	db := &DB{conn: conn}
	return db, nil
}

func (db *DB) ApproveFile(projectId types.ProjectId, fileId types.FileId, userId types.UserId) error {
	stmt := rq.ParameterizedStatement{
		Query:     sqlApproveFile,
		Arguments: []any{string(projectId), string(fileId), string(userId)},
	}

	wr, operr := db.conn.WriteOneParameterized(stmt)
	err := unifyErrors("failed to execute insert", wr.Err, operr)

	return err
}

func (db *DB) FileApprovals(projectId types.ProjectId) (types.ProjectApprovals, error) {
	stmt := rq.ParameterizedStatement{
		Query:     sqlFileApprovals,
		Arguments: []any{string(projectId)},
	}

	qr, operr := db.conn.QueryOneParameterized(stmt)
	err := unifyErrors("failed to execute query", qr.Err, operr)
	if err != nil {
		return nil, err
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
		return "", fmt.Errorf("insufficient rqlite credentials")
	}
	u.User = url.UserPassword(username, password)
	return u.String(), nil
}

func unifyErrors(msg string, dberr, operr error) error {
	f := "%s: %w"
	if operr != nil { // First check for API call errors..
		return fmt.Errorf(f, msg, operr)
	}
	if dberr != nil { // ..then check for DB errors
		return fmt.Errorf(f, msg, dberr)
	}
	return nil
}
