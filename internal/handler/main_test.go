package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/db/inmemory"
	"github.com/ucl-arc-tre/egress/internal/storage/s3"
	"github.com/ucl-arc-tre/egress/internal/types"
)

const (
	projectId = "p123"
)

func TestGetFiles(t *testing.T) {
	testCases := []struct {
		name      string
		body      string
		s3client  s3.MockClient
		approvals map[types.FileId]types.UserId

		expectedStatusCode int
		expectedBody       string
	}{
		{
			name:               "invalid body",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Invalid object. Failed to parse request body"}`,
		},
		{
			name:               "no s3 client",
			body:               `{"file_location":"s3://bucket"}`,
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:               "invalid location",
			body:               `{"file_location":"://bucket"}`,
			s3client:           s3.MockClient{},
			expectedStatusCode: 520,
		},
		{
			name:               "unknown file location",
			body:               `{"file_location":"unknown://bucket"}`,
			s3client:           s3.MockClient{},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "ok",
			body: `{"file_location":"s3://bucket"}`,
			s3client: s3.MockClient{
				Buckets: map[s3.MockBucketName]s3.MockBucket{
					"bucket": {
						Objects: []s3.MockObject{
							{
								Key:     "object1",
								Etag:    `"etag1"`,
								Content: "hello world",
							},
						},
					},
				},
			},
			approvals: map[types.FileId]types.UserId{
				"etag1": "user1",
				"etag2": "user1",
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `[{"approvals":["user1"],"file_name":"object1","id":"etag1","size":11}]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &Handler{
				s3: s3.NewMock(tc.s3client),
				db: inmemory.New(),
			}
			for fileId, userId := range tc.approvals {
				err := handler.db.ApproveFile(types.ProjectId(projectId), fileId, userId)
				assert.NoError(t, err)
			}
			writer := httptest.NewRecorder()
			ctx, router := gin.CreateTestContext(writer)
			router.GET("/", func(ctx *gin.Context) {
				handler.GetProjectIdFiles(ctx, projectId)
			})
			ctx.Request, _ = http.NewRequest(http.MethodGet, "/", strings.NewReader(tc.body))
			router.ServeHTTP(writer, ctx.Request)

			assert.Equal(t, tc.expectedStatusCode, writer.Code)
			body, err := io.ReadAll(writer.Body)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, string(body))
		})
	}
}

func TestGetFileId(t *testing.T) {
	fileId1 := "etag1"
	bucket1 := s3.MockBucket{
		Objects: []s3.MockObject{
			{
				Key:     "object1",
				Etag:    fmt.Sprintf(`"%s"`, fileId1),
				Content: "hello world",
			},
		},
	}
	testCases := []struct {
		name      string
		body      string
		fileId    string
		s3client  s3.MockClient
		approvals map[types.FileId]types.UserId

		expectedStatusCode int
		expectedBody       string
	}{
		{
			name:               "invalid body",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Invalid object. Failed to parse request body"}`,
		},
		{
			name:               "bad location",
			body:               `{"files_location":"s:/bucket1","max_file_size":100,"required_approvals":0}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "no approvals",
			body: `{"files_location":"s3://bucket1","max_file_size":100,"required_approvals":1}`,
			s3client: s3.MockClient{
				Buckets: map[s3.MockBucketName]s3.MockBucket{
					"bucket1": bucket1,
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Required 1 approvals but only had 0"}`,
		},
		{
			name:   "above max body size",
			body:   `{"files_location":"s3://bucket1","max_file_size":1,"required_approvals":0}`,
			fileId: fileId1,
			s3client: s3.MockClient{
				Buckets: map[s3.MockBucketName]s3.MockBucket{
					"bucket1": bucket1,
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Size [11] was greater than max [1]"}`,
		},
		{
			name:   "ok",
			body:   `{"files_location":"s3://bucket1","max_file_size":100,"required_approvals":1}`,
			fileId: fileId1,
			s3client: s3.MockClient{
				Buckets: map[s3.MockBucketName]s3.MockBucket{
					"bucket1": bucket1,
				},
			},
			approvals: map[types.FileId]types.UserId{
				types.FileId(fileId1): "user1",
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `hello world`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &Handler{
				s3: s3.NewMock(tc.s3client),
				db: inmemory.New(),
			}
			for fileId, userId := range tc.approvals {
				err := handler.db.ApproveFile(types.ProjectId(projectId), fileId, userId)
				assert.NoError(t, err)
			}
			writer := httptest.NewRecorder()
			ctx, router := gin.CreateTestContext(writer)
			router.GET("/", func(ctx *gin.Context) {
				handler.GetProjectIdFilesFileId(ctx, projectId, tc.fileId)
			})
			ctx.Request, _ = http.NewRequest(http.MethodGet, "/", strings.NewReader(tc.body))
			router.ServeHTTP(writer, ctx.Request)

			assert.Equal(t, tc.expectedStatusCode, writer.Code)
			body, err := io.ReadAll(writer.Body)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, string(body))
		})
	}
}

func TestApproveFileId(t *testing.T) {
	testCases := []struct {
		name   string
		body   string
		fileId string

		expectedStatusCode int
		expectedBody       string
		expectedApprovals  int
	}{
		{
			name:               "invalid body",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Invalid object. Failed to parse request body"}`,
			expectedApprovals:  0,
		},
		{
			name:               "ok",
			body:               `{"user_id":"user1"}`,
			expectedStatusCode: http.StatusNoContent,
			expectedBody:       ``,
			expectedApprovals:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &Handler{db: inmemory.New()}
			writer := httptest.NewRecorder()
			ctx, router := gin.CreateTestContext(writer)
			router.PUT("/", func(ctx *gin.Context) {
				handler.PutProjectIdFilesFileIdApprove(ctx, projectId, tc.fileId)
			})
			ctx.Request, _ = http.NewRequest(http.MethodPut, "/", strings.NewReader(tc.body))
			router.ServeHTTP(writer, ctx.Request)

			assert.Equal(t, tc.expectedStatusCode, writer.Code)
			body, err := io.ReadAll(writer.Body)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, string(body))

			approvals, err := handler.db.FileApprovals(projectId)
			assert.NoError(t, err)
			assert.Len(t, approvals[types.FileId(tc.fileId)], tc.expectedApprovals)
		})
	}
}
