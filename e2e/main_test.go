package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/config"
)

const (
	baseUrl          = "http://localhost:8080"
	username         = "egressuser"
	password         = "egressuser" /* pragma: allowlist secret */
	baseApiUrl       = baseUrl + config.BaseURL
	requestTimeout   = 1 * time.Second
	serviceUpTimeout = 2 * time.Minute
)

var (
	storageProvider = newStorageProviderFromEnv()
	filesLocation   = storageProvider.FilesLocation()
)

func init() {
	timeout := time.Now().Add(serviceUpTimeout)
	for {
		if time.Now().After(timeout) {
			panic("timed out waiting for service readiness")
		}
		if canPing() && canListFiles() {
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: requestTimeout}
}

func canPing() bool {
	res, err := http.Get(baseUrl + "/ping")
	return err == nil && res.StatusCode == http.StatusOK
}

func canListFiles() bool {
	url := fmt.Sprintf("%s/%s/files", baseApiUrl, "p001")
	body := strings.NewReader(fmt.Sprintf(`{"files_location":"%s"}`, filesLocation))
	req, err := http.NewRequest(http.MethodGet, url, body)
	if err != nil {
		return false
	}
	req.SetBasicAuth(username, password)
	resp, err := newHTTPClient().Do(req)
	return err == nil && resp.StatusCode == http.StatusOK
}

func TestEndpointResponseCodes(t *testing.T) {
	projectId := "p0001"
	fileId := "f1234"

	client := newHTTPClient()
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
			body:   makeRequestBodyF(`{"files_location":"%s"}`, filesLocation),

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
			body:   makeRequestBodyF(`{"required_approvals":1,"files_location":"%s","max_file_size": 1}`, filesLocation),

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
			req.SetBasicAuth(username, password)
			res, err := client.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatusCode, res.StatusCode)
		})
	}
}

func TestApprovalAndEgress(t *testing.T) {
	projectId := "pTestApprovalAndEgress"
	userId := "userTestApprovalAndEgress"

	key := uuid.New()
	fileContent := fmt.Sprintf("hello %s", key.String())

	assert.NoError(t, storageProvider.PutFile(key.String(), fileContent))

	client := newHTTPClient()

	// List files - expecting one with none approved
	req := must(http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/%s/files", baseApiUrl, projectId),
		makeRequestBodyF(`{"files_location": "%s"}`, filesLocation),
	))
	req.SetBasicAuth(username, password)
	res := must(client.Do(req))
	assert.Equal(t, http.StatusOK, res.StatusCode)

	partialListFilesResponse := PartialListFilesResponse{}
	assertNoError(json.NewDecoder(res.Body).Decode(&partialListFilesResponse))
	assertNoError(res.Body.Close())
	assert.True(t, len(partialListFilesResponse) > 0)
	partialListFileResponse, exists := partialListFilesResponse.FileByFilename(key.String())
	assert.True(t, exists)
	assert.Len(t, partialListFileResponse.Approvals, 0)

	// Retrieve the file ID from the list response
	fileId := partialListFileResponse.Id

	// Approve uploaded file
	req = must(http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/%s/files/%s/approve", baseApiUrl, projectId, fileId),
		makeRequestBodyF(`{"user_id": "%s"}`, userId),
	))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)
	res = must(client.Do(req))
	assert.Equal(t, http.StatusNoContent, res.StatusCode)

	// List files - expecting one approved
	req = must(http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/%s/files", baseApiUrl, projectId),
		makeRequestBodyF(`{"files_location": "%s"}`, filesLocation),
	))
	req.SetBasicAuth(username, password)
	res = must(client.Do(req))
	assertNoError(json.NewDecoder(res.Body).Decode(&partialListFilesResponse))
	assertNoError(res.Body.Close())
	partialListFileResponse, exists = partialListFilesResponse.FileByFilename(key.String())
	assert.True(t, exists)
	assert.Len(t, partialListFileResponse.Approvals, 1)

	// The one file can now be downloaded
	req = must(http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/%s/files/%s", baseApiUrl, projectId, fileId),
		makeRequestBodyF(
			`{"required_approvals": %d,"files_location": "%s","max_file_size": %d}`,
			1,
			filesLocation,
			100,
		),
	))
	req.SetBasicAuth(username, password)
	res = must(client.Do(req))
	assert.Equal(t, http.StatusOK, res.StatusCode)
	content := must(io.ReadAll(res.Body))
	assert.Equal(t, fileContent, string(content))
}

func TestApprovalIdempotency(t *testing.T) {
	userId := uuid.New().String()
	approve_url := fmt.Sprintf("%s/%s/files/%s/approve", baseApiUrl, "p0004", "f0004")

	client := newHTTPClient()

	// First pass
	req := must(http.NewRequest(
		http.MethodPut,
		approve_url,
		makeRequestBodyF(`{"user_id": "%s"}`, userId),
	))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)
	res := must(client.Do(req))
	assert.Equal(t, http.StatusNoContent, res.StatusCode)

	// Second pass, with same project-id, file-id, user-id
	req = must(http.NewRequest(
		http.MethodPut,
		approve_url,
		makeRequestBodyF(`{"user_id": "%s"}`, userId),
	))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)
	res = must(client.Do(req))
	assert.Equal(t, http.StatusNoContent, res.StatusCode)
}

func TestAuthFailureWithIncorrectUsername(t *testing.T) {
	client := newHTTPClient()
	req := must(http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/%s/files", baseApiUrl, "p0001"),
		makeRequestBodyF(`{"files_location": "%s"}`, filesLocation),
	))
	req.SetBasicAuth("badUsername", password)
	res := must(client.Do(req))
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestAuthFailureWithIncorrectPassword(t *testing.T) {
	client := newHTTPClient()
	req := must(http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/%s/files", baseApiUrl, "p0001"),
		makeRequestBodyF(`{"files_location": "%s"}`, filesLocation),
	))
	req.SetBasicAuth(username, "badPassword")
	res := must(client.Do(req))
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func makeRequestBodyF(format string, objs ...any) io.Reader {
	return strings.NewReader(fmt.Sprintf(format, objs...))
}
