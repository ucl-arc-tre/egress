package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

const (
	baseUrl          = "http://localhost:8080"
	baseApiUrl       = baseUrl + config.BaseURL
	requestTimeout   = 1 * time.Second
	serviceUpTimeout = 2 * time.Minute
)

var (
	s3Location = fmt.Sprintf("s3://%s", bucketName)
)

func init() {
	timeout := time.Now().Add(serviceUpTimeout)
	for {
		if time.Now().After(timeout) {
			panic("timed out waiting for ping")
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
	body := strings.NewReader(fmt.Sprintf(`{"file_location":"%s"}`, s3Location))
	req, err := http.NewRequest("GET", url, body)
	if err != nil {
		return false
	}
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
			body:   makeRequestBodyF(`{"file_location":"%s"}`, s3Location),

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
			body:   makeRequestBodyF(`{"required_approvals":1,"files_location":"%s","max_file_size": 1}`, s3Location),

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

func TestApprovalAndEgressS3(t *testing.T) {
	projectId := "pTestApprovalAndEgressS3"
	userId := "userTestApprovalAndEgressS3"

	key := uuid.New()
	fileContent := fmt.Sprintf("hello %s", key.String())

	s3Client := newS3Client()
	putObjectOut, err := s3Client.PutObject(context.Background(), &awsS3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key.String()),
		Body:   strings.NewReader(fileContent),
	})
	assert.NoError(t, err)
	assert.NotNil(t, putObjectOut.ETag)
	fileId := stripQuotes(*putObjectOut.ETag)

	client := newHTTPClient()

	// List files - expecting one with none approved
	req := must(http.NewRequest(
		"GET",
		fmt.Sprintf("%s/%s/files", baseApiUrl, projectId),
		makeRequestBodyF(`{"file_location": "%s"}`, s3Location),
	))
	res := must(client.Do(req))
	assert.Equal(t, http.StatusOK, res.StatusCode)

	partialListFilesResponse := PartialListFilesResponse{}
	assertNoError(json.NewDecoder(res.Body).Decode(&partialListFilesResponse))
	assertNoError(res.Body.Close())
	assert.NoError(t, err)
	assert.True(t, len(partialListFilesResponse) > 0)
	partialListFileResponse, exists := partialListFilesResponse.FileByFilename(key.String())
	assert.True(t, exists)
	assert.Len(t, partialListFileResponse.Approvals, 0)

	// Approve uploaded file
	req = must(http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/%s/files/%s/approve", baseApiUrl, projectId, fileId),
		makeRequestBodyF(`{"user_id": "%s"}`, userId),
	))
	req.Header.Set("Content-Type", "application/json")
	res = must(client.Do(req))
	assert.Equal(t, http.StatusNoContent, res.StatusCode)

	// List files - expecting one approved
	req = must(http.NewRequest(
		"GET",
		fmt.Sprintf("%s/%s/files", baseApiUrl, projectId),
		makeRequestBodyF(`{"file_location": "%s"}`, s3Location),
	))
	res = must(client.Do(req))
	assertNoError(json.NewDecoder(res.Body).Decode(&partialListFilesResponse))
	assertNoError(res.Body.Close())
	partialListFileResponse, exists = partialListFilesResponse.FileByFilename(key.String())
	assert.True(t, exists)
	assert.Len(t, partialListFileResponse.Approvals, 1)

	// The one file can now be downloaded
	req = must(http.NewRequest(
		"GET",
		fmt.Sprintf("%s/%s/files/%s", baseApiUrl, projectId, fileId),
		makeRequestBodyF(
			`{"required_approvals": %d,"files_location": "%s","max_file_size": %d}`,
			1,
			s3Location,
			100,
		),
	))
	res = must(client.Do(req))
	assert.Equal(t, http.StatusOK, res.StatusCode)
	content := must(io.ReadAll(res.Body))
	assert.Equal(t, fileContent, string(content))
}

func stripQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, "")
}

func makeRequestBodyF(format string, objs ...any) io.Reader {
	return strings.NewReader(fmt.Sprintf(format, objs...))
}
