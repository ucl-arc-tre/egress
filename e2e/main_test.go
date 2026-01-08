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
	baseUrl          = "http://localhost:8080"
	baseApiUrl       = baseUrl + "/v0"
	requestTimeout   = 1 * time.Second
	serviceUpTimeout = 1 * time.Minute

	s3Location = "s3://bucket1"
)

func init() {
	timeout := time.Now().Add(serviceUpTimeout)
	for {
		if time.Now().After(timeout) {
			panic("timed out waiting for ping")
		}
		resp, err := http.Get(baseUrl + "/ping")
		if err == nil && resp.StatusCode == http.StatusOK {
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func TestEndpointsResponseCode(t *testing.T) {
	projectId := "p0001"
	fileId := "f1234"

	client := &http.Client{Timeout: requestTimeout}
	tests := []struct {
		name   string
		method string
		url    string
		body   io.Reader

		expectedStatusCode int
	}{
		{
			name:   "GetFileList",
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/%s/files", baseApiUrl, projectId),
			body:   strings.NewReader(fmt.Sprintf(`{"file_location":"%s"}`, s3Location)),

			expectedStatusCode: http.StatusOK,
		},
		{
			name:   "ApproveFile",
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/%s/files/%s/approve", baseApiUrl, projectId, fileId),
			body:   strings.NewReader(`{"user_id":"user1"}`),

			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:   "ApproveFileInvalidJson",
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/%s/files/%s/approve", baseApiUrl, projectId, fileId),
			body:   strings.NewReader(`{"user_id}`),

			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "GetFile",
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/%s/files/%s", baseApiUrl, projectId, fileId),
			body:   strings.NewReader(fmt.Sprintf(`{"required_approvals":1,"files_location":"%s","max_file_size": 1}`, s3Location)),

			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:   "GetFileInvalidJson",
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/%s/files/%s", baseApiUrl, projectId, fileId),
			body:   strings.NewReader(`{"n}`),

			expectedStatusCode: http.StatusBadRequest,
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
			assert.Equal(t, tc.expectedStatusCode, res.StatusCode)
		})
	}
}
