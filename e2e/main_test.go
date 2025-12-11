package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	baseUrl        = "http://localhost:8080/v0"
	requestTimeout = 1 * time.Second
)

func TestEndpoints(t *testing.T) {
	client := &http.Client{Timeout: requestTimeout}
	tests := []struct {
		name   string
		method string
		url    string
		body   io.Reader
	}{
		{
			name:   "GetFileList",
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/%s/files", baseUrl, "p0001"),
		},
		{
			name:   "ApproveFile",
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/%s/files/%s/approve", baseUrl, "p0001", "f1234"),
			body:   strings.NewReader(`{"user_id":"user1"}`),
		},
		{
			name:   "GetFile",
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/%s/files/%s", baseUrl, "p0001", "f1234"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.url, tc.body)
			assert.NoError(t, err)
			if tc.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			res, err := client.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusNotImplemented, res.StatusCode)
		})
	}
}
