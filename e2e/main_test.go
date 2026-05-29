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
	"github.com/ucl-arc-tre/egress/internal/openapi"
)

const (
	baseUrl          = "http://localhost:8080"
	username         = "egressuser"
	password         = "egressuser" /* pragma: allowlist secret */
	baseApiUrl       = baseUrl + config.BaseURL
	requestTimeout   = 1 * time.Second
	serviceUpTimeout = 2 * time.Minute
)

const (
	projectId          = "p001"
	destinationTrusted = "trusted"
	destinationPublic  = "world"
	commentApprove1    = "nice"
	commentApprove2    = "good"
	commentReject      = "bad"
	commentDownload    = "results"
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
	url := fmt.Sprintf("%s/%s/files", baseApiUrl, projectId)
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
	fileId := "f1234" // Non-existent file-id/etag

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
			body:   strings.NewReader(`{"user_id":"user1","destination":"trusted","comment":"ok"}`),

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
			name:   "GetFileNonExistent",
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
		{
			name:   "GetProjectEvents",
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/%s/events", baseApiUrl, projectId),
			body:   nil,

			expectedStatusCode: http.StatusOK,
		},
		{
			name:   "RejectFile",
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/%s/files/%s/reject", baseApiUrl, projectId, fileId),
			body:   strings.NewReader(`{"user_id":"user1","destination":"trusted","comment":"bad"}`),

			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:   "RejectFileInvalidJson",
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/%s/files/%s/reject", baseApiUrl, projectId, fileId),
			body:   strings.NewReader(`{"user_id}`),

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

func TestApproveAndEgress(t *testing.T) {
	tests := []struct {
		name               string
		approveDestination string
		egressDestination  string
		expectedStatusCode int
	}{
		{
			name:               "ApprovedForTrusted-EgressToTrusted",
			approveDestination: destinationTrusted,
			egressDestination:  destinationTrusted,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "ApprovedForTrusted-EgressToWorld",
			approveDestination: destinationTrusted,
			egressDestination:  destinationPublic,
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userId := "userTestApprovalAndEgress"

			// Upload a file to storage with unique key
			key := uuid.New()
			fileContent := fmt.Sprintf("hello %s", key.String())
			assert.NoError(t, storageProvider.PutFile(key.String(), fileContent))

			// List files - expecting file to have no approvals
			files := listFiles(t, projectId)
			assert.True(t, len(files) > 0)

			uploadedFile, exists := files.FileByFilename(key.String())
			assert.True(t, exists)
			assert.Len(t, uploadedFile.Approvals, 0)

			fileId := uploadedFile.Id

			// Approve uploaded file for the given destination
			approve(t, projectId, fileId, userId, tc.approveDestination, commentApprove1)

			// List files - expecting file to have one approval
			files = listFiles(t, projectId)
			uploadedFile, exists = files.FileByFilename(key.String())
			assert.True(t, exists)
			assert.Len(t, uploadedFile.Approvals, 1)

			// Attempt to download file to the egress destination
			content, status := download(t, projectId, fileId, userId, tc.egressDestination, commentDownload)
			assert.Equal(t, tc.expectedStatusCode, status)
			if status == http.StatusOK {
				assert.Equal(t, fileContent, content)
			}
		})
	}
}

func TestApproveThenList(t *testing.T) {
	userId := "user-" + uuid.New().String()

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

	// Approve file
	approve(t, projectId, fileId, userId, destinationTrusted, commentApprove1)

	// List file; expect 1 approval
	files = listFiles(t, projectId)
	approvedFile, exists := files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 1)
	assert.Equal(t, userId, approvedFile.Approvals[0].UserId)
	assert.Equal(t, destinationTrusted, approvedFile.Approvals[0].Destination)
	assert.Equal(t, commentApprove1, approvedFile.Approvals[0].Comment)
}

func TestMultipleApprovals(t *testing.T) {
	userId := "user-" + uuid.New().String()

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

	// Approve file
	approve(t, projectId, fileId, userId, destinationTrusted, commentApprove1)
	files = listFiles(t, projectId)
	approvedFile, exists := files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 1)

	// Approve file again with same {userId,destination}
	// Approve is not idempotent, but approvals are deduped on
	// {userId,desitnation}, so only 1 approval is returned
	approve(t, projectId, fileId, userId, destinationTrusted, commentApprove2)
	files = listFiles(t, projectId)
	approvedFile, _ = files.FileById(fileId)
	assert.Len(t, approvedFile.Approvals, 1)
	assert.Equal(t, userId, approvedFile.Approvals[0].UserId)
	assert.Equal(t, destinationTrusted, approvedFile.Approvals[0].Destination)
	assert.Equal(t, commentApprove1, approvedFile.Approvals[0].Comment)
}

func TestApproveToMultipleDestinations(t *testing.T) {
	userId := "user-" + uuid.New().String()

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
	approve(t, projectId, fileId, userId, destinationTrusted, commentApprove1)
	files = listFiles(t, projectId)
	approvedFile, exists := files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 1)
	assert.Equal(t, userId, approvedFile.Approvals[0].UserId)
	assert.Equal(t, destinationTrusted, approvedFile.Approvals[0].Destination)

	// Approve file for destination-2 by same user; so has 2 approvals
	approve(t, projectId, fileId, userId, destinationPublic, commentApprove1)
	files = listFiles(t, projectId)
	approvedFile, exists = files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 2)
	assert.Equal(t, userId, approvedFile.Approvals[1].UserId)
	assert.Equal(t, destinationPublic, approvedFile.Approvals[1].Destination)

	// Approve for destination-1 by same user
	// Approvals are deduped on {userId,desitnation}, so still 2 approvals
	approve(t, projectId, fileId, userId, destinationTrusted, commentApprove2)
	files = listFiles(t, projectId)
	approvedFile, exists = files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 2)
}

func TestRejectThenList(t *testing.T) {
	userId := "user-" + uuid.New().String()

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

	// Reject file
	reject(t, projectId, fileId, userId, destinationTrusted, commentReject)

	// List file; had no approvals, so no impact
	files = listFiles(t, projectId)
	approvedFile, exists := files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 0)

	// List events; expect a rejection event as latest
	events := listEvents(t, projectId)
	assert.GreaterOrEqual(t, len(events), 1)

	latest := events[len(events)-1]
	assert.Equal(t, fileId, latest.FileId)
	assert.Equal(t, userId, latest.UserId)
	assert.Equal(t, openapi.EventActionRejection, *latest.Action)
	assert.Equal(t, destinationTrusted, *latest.Destination)
	assert.Equal(t, commentReject, *latest.Comment)
}

func TestApproveThenReject(t *testing.T) {
	userId := "user-" + uuid.New().String()

	// Upload a file to storage
	key := uuid.New()
	fileContent := fmt.Sprintf("hello %s", key.String())
	assert.NoError(t, storageProvider.PutFile(key.String(), fileContent))

	// List files to get file-id of uploaded file
	files := listFiles(t, projectId)
	uploadedFile, exists := files.FileByFilename(key.String())
	assert.True(t, exists)

	fileId := uploadedFile.Id

	// Approve file
	approve(t, projectId, fileId, userId, destinationTrusted, commentApprove1)
	files = listFiles(t, projectId)
	approvedFile, exists := files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 1)

	// Now reject file; expect no approvals after this
	reject(t, projectId, fileId, userId, destinationTrusted, commentReject)
	files = listFiles(t, projectId)
	approvedFile, exists = files.FileById(fileId)
	assert.True(t, exists)
	assert.Len(t, approvedFile.Approvals, 0)
}

func TestEventsOfEgressActions(t *testing.T) {
	userId := "user-" + uuid.New().String()

	// Upload a file to storage
	key := uuid.New()
	fileContent := fmt.Sprintf("hello %s", key.String())
	assert.NoError(t, storageProvider.PutFile(key.String(), fileContent))

	// List files to get file-id of uploaded file
	files := listFiles(t, projectId)
	uploadedFile, exists := files.FileByFilename(key.String())
	assert.True(t, exists)

	fileId := uploadedFile.Id

	// Approve, reject, approve and download file
	approve(t, projectId, fileId, userId, destinationPublic, commentApprove1)
	reject(t, projectId, fileId, userId, destinationPublic, commentReject)
	approve(t, projectId, fileId, userId, destinationTrusted, commentApprove2)
	_, status := download(t, projectId, fileId, userId, destinationTrusted, commentDownload)
	assert.Equal(t, http.StatusOK, status)

	// Get the 4 latest events and verify that they are -
	// approve, reject, approve and download events, in that order
	events := listEvents(t, projectId)
	assert.GreaterOrEqual(t, len(events), 4)

	approve1 := events[len(events)-4]
	assert.Equal(t, openapi.EventActionApproval, *approve1.Action)
	assert.Equal(t, fileId, approve1.FileId)
	assert.Equal(t, userId, approve1.UserId)
	assert.Equal(t, commentApprove1, *approve1.Comment)

	reject := events[len(events)-3]
	assert.Equal(t, openapi.EventActionRejection, *reject.Action)
	assert.Equal(t, fileId, reject.FileId)
	assert.Equal(t, userId, reject.UserId)
	assert.Equal(t, commentReject, *reject.Comment)

	approve2 := events[len(events)-2]
	assert.Equal(t, openapi.EventActionApproval, *approve2.Action)
	assert.Equal(t, fileId, approve2.FileId)
	assert.Equal(t, userId, approve2.UserId)
	assert.Equal(t, commentApprove2, *approve2.Comment)

	download := events[len(events)-1]
	assert.Equal(t, openapi.EventActionDownload, *download.Action)
	assert.Equal(t, fileId, download.FileId)
	assert.Equal(t, userId, download.UserId)
	assert.Equal(t, commentDownload, *download.Comment)
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

func listEvents(t *testing.T, projectId string) openapi.EventListResponse {
	t.Helper()
	client := newHTTPClient()

	eventsUrl := fmt.Sprintf("%s/%s/events", baseApiUrl, projectId)
	req := must(http.NewRequest(
		http.MethodGet,
		eventsUrl,
		makeRequestBodyF(`{"files_location": "%s"}`, filesLocation),
	))
	req.SetBasicAuth(username, password)
	res := must(client.Do(req))
	require.Equal(t, http.StatusOK, res.StatusCode)

	events := openapi.EventListResponse{}
	assertNoError(json.NewDecoder(res.Body).Decode(&events))
	assertNoError(res.Body.Close())

	return events
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
	comment string,
) {
	t.Helper()
	client := newHTTPClient()

	approveUrl := fmt.Sprintf("%s/%s/files/%s/approve", baseApiUrl, projectId, fileId)
	req := must(http.NewRequest(
		http.MethodPut,
		approveUrl,
		makeRequestBodyF(`{"user_id": "%s", "destination": "%s", "comment": "%s"}`,
			userId, destination, comment),
	))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)
	res := must(client.Do(req))
	assert.Equal(t, http.StatusNoContent, res.StatusCode)
}

func reject(
	t *testing.T,
	projectId string,
	fileId string,
	userId string,
	destination string,
	comment string,
) {
	t.Helper()
	client := newHTTPClient()

	rejectUrl := fmt.Sprintf("%s/%s/files/%s/reject", baseApiUrl, projectId, fileId)
	req := must(http.NewRequest(
		http.MethodPut,
		rejectUrl,
		makeRequestBodyF(`{"user_id": "%s", "destination": "%s", "comment": "%s"}`,
			userId, destination, comment),
	))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)
	res := must(client.Do(req))
	assert.Equal(t, http.StatusNoContent, res.StatusCode)
}

func download(
	t *testing.T,
	projectId string,
	fileId string,
	userId string,
	destination string,
	comment string,
) (string, int) {
	t.Helper()
	client := newHTTPClient()

	downloadUrl := fmt.Sprintf("%s/%s/files/%s", baseApiUrl, projectId, fileId)
	req := must(http.NewRequest(
		http.MethodGet,
		downloadUrl,
		makeRequestBodyF(
			`{"required_approvals": 1,"max_file_size": 100,"files_location": "%s","destination": "%s","user_id": "%s","comment": "%s"}`,
			filesLocation,
			destination,
			userId,
			comment,
		),
	))
	req.SetBasicAuth(username, password)
	res := must(client.Do(req))

	var content []byte
	if res.StatusCode == http.StatusOK {
		content = must(io.ReadAll(res.Body))
	}
	return string(content), res.StatusCode
}

func makeRequestBodyF(format string, objs ...any) io.Reader {
	return strings.NewReader(fmt.Sprintf(format, objs...))
}
