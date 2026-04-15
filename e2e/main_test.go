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
	"github.com/stretchr/testify/require"
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
			body:   strings.NewReader(`{"user_id":"user1","destination":"trusted"}`),

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
			body:   makeRequestBodyF(`{"required_approvals":1,"destination":"trusted","files_location":"%s","max_file_size": 1}`, filesLocation),

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
	tests := []struct {
		name               string
		approveDestination string
		egressDestination  string
		expectedStatusCode int
	}{
		{
			name:               "ApprovedForTrusted-EgressToTrusted",
			approveDestination: "trusted",
			egressDestination:  "trusted",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "ApprovedForTrusted-EgressToWorld",
			approveDestination: "trusted",
			egressDestination:  "world",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			projectId := "pTestApprovalAndEgress"
			userId := "userTestApprovalAndEgress"

			// Upload a file to storage with unique key
			key := uuid.New()
			fileContent := fmt.Sprintf("hello %s", key.String())
			assert.NoError(t, storageProvider.PutFile(key.String(), fileContent))

			client := newHTTPClient()

			// List files - expecting file to have no approvals
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

			// Approve uploaded file for the given destination
			req = must(http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("%s/%s/files/%s/approve", baseApiUrl, projectId, fileId),
				makeRequestBodyF(`{"user_id": "%s", "destination": "%s"}`, userId, tc.approveDestination),
			))
			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth(username, password)
			res = must(client.Do(req))
			assert.Equal(t, http.StatusNoContent, res.StatusCode)

			// List files - expecting file to have one approval
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

			// Attempt to download file to the egress destination
			req = must(http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("%s/%s/files/%s", baseApiUrl, projectId, fileId),
				makeRequestBodyF(
					`{"required_approvals": 1,"destination": "%s", "files_location": "%s","max_file_size": 100}`,
					tc.egressDestination,
					filesLocation,
				),
			))
			req.SetBasicAuth(username, password)
			res = must(client.Do(req))
			assert.Equal(t, tc.expectedStatusCode, res.StatusCode)
			if tc.expectedStatusCode == http.StatusOK {
				content := must(io.ReadAll(res.Body))
				assert.Equal(t, fileContent, string(content))
			}
		})
	}
}

func TestApproveIdempotency(t *testing.T) {
	projectId := "p0004"
	userId := "user-" + uuid.New().String()
	destination := "trusted"

	// Upload a file to storage
	key := uuid.New()
	fileContent := fmt.Sprintf("hello %s", key.String())
	require.NoError(t, storageProvider.PutFile(key.String(), fileContent))

	// List files - expecting file to have no approvals
	files := listFiles(t, projectId)
	assert.True(t, len(files) > 0)

	uploadedFile, exists := files.FileByFilename(key.String())
	assert.True(t, exists)
	assert.Len(t, uploadedFile.Approvals, 0)

	fileId := uploadedFile.Id

	// First pass; only one approval
	approve(t, projectId, fileId, userId, destination)
	files = listFiles(t, projectId)
	approvedFile, exists := files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 1)
	assert.Equal(t, userId, approvedFile.Approvals[0].UserId)
	assert.Equal(t, destination, approvedFile.Approvals[0].Destination)

	// Second pass; still the same one approval
	approve(t, projectId, fileId, userId, destination)
	files = listFiles(t, projectId)
	approvedFile, _ = files.FileById(fileId)
	assert.Len(t, approvedFile.Approvals, 1)
	assert.Equal(t, userId, approvedFile.Approvals[0].UserId)
	assert.Equal(t, destination, approvedFile.Approvals[0].Destination)
}

func TestApproveSameUserMultipleDestinations(t *testing.T) {
	projectId := "p0008"
	userId := "user-" + uuid.New().String()
	destination1 := "trusted"
	destination2 := "world"

	// Upload a file to storage
	key := uuid.New()
	fileContent := fmt.Sprintf("hello %s", key.String())
	assert.NoError(t, storageProvider.PutFile(key.String(), fileContent))

	// List files to get file-id of uploaded file
	files := listFiles(t, projectId)
	assert.True(t, len(files) > 0)

	uploadedFile, exists := files.FileByFilename(key.String())
	assert.True(t, exists)

	fileId := uploadedFile.Id

	// Approve file for destination-1
	approve(t, projectId, fileId, userId, destination1)
	files = listFiles(t, projectId)
	approvedFile, exists := files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 1)
	assert.Equal(t, userId, approvedFile.Approvals[0].UserId)
	assert.Equal(t, destination1, approvedFile.Approvals[0].Destination)

	// Approve file for destination-2 by same user; so has 2 approvals
	approve(t, projectId, fileId, userId, destination2)
	files = listFiles(t, projectId)
	approvedFile, exists = files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 2)
	assert.Equal(t, userId, approvedFile.Approvals[1].UserId)
	assert.Equal(t, destination2, approvedFile.Approvals[1].Destination)

	// Approve for destination-1 by same user; still 2 approvals
	approve(t, projectId, fileId, userId, destination1)
	files = listFiles(t, projectId)
	approvedFile, exists = files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 2)
	assert.Equal(t, userId, approvedFile.Approvals[0].UserId)
	assert.Equal(t, destination1, approvedFile.Approvals[0].Destination)
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

func listFiles(t *testing.T, projectId string) PartialListFilesResponse {
	t.Helper()
	client := newHTTPClient()

	listUrl := fmt.Sprintf("%s/%s/files", baseApiUrl, projectId)
	req := must(http.NewRequest(
		http.MethodGet,
		listUrl,
		makeRequestBodyF(`{"files_location": "%s"}`, filesLocation),
	))
	req.SetBasicAuth(username, password)
	res := must(client.Do(req))
	require.Equal(t, http.StatusOK, res.StatusCode)

	files := PartialListFilesResponse{}
	assertNoError(json.NewDecoder(res.Body).Decode(&files))
	assertNoError(res.Body.Close())

	return files
}

func approve(
	t *testing.T,
	projectId string,
	fileId string,
	userId string,
	destination string,
) {
	t.Helper()
	client := newHTTPClient()

	approveUrl := fmt.Sprintf("%s/%s/files/%s/approve", baseApiUrl, projectId, fileId)
	req := must(http.NewRequest(
		http.MethodPut,
		approveUrl,
		makeRequestBodyF(`{"user_id": "%s", "destination": "%s"}`, userId, destination),
	))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)
	res := must(client.Do(req))
	assert.Equal(t, http.StatusNoContent, res.StatusCode)
}

func makeRequestBodyF(format string, objs ...any) io.Reader {
	return strings.NewReader(fmt.Sprintf(format, objs...))
}
