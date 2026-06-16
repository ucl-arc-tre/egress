package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		approvals map[types.FileId]types.Approval

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
			body:               `{"files_location":"s3://bucket"}`,
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:               "invalid location",
			body:               `{"files_location":"://bucket"}`,
			s3client:           s3.MockClient{},
			expectedStatusCode: 520,
		},
		{
			name:               "unknown file location",
			body:               `{"files_location":"unknown://bucket"}`,
			s3client:           s3.MockClient{},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "empty location",
			body:               `{"files_location":""}`,
			s3client:           s3.MockClient{},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "ok",
			body: `{"files_location":"s3://bucket"}`,
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
			approvals: map[types.FileId]types.Approval{
				"etag1": {UserId: "user1", Destination: "trusted", Comment: "ok"},
				"etag2": {UserId: "user1", Destination: "trusted", Comment: "ok"},
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `[{"approvals":[{"comment":"ok","destination":"trusted","user_id":"user1"}],"file_name":"object1","id":"etag1","size":11}]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &Handler{
				storage: s3.NewMock(tc.s3client),
				db:      inmemory.New(),
			}
			for fileId, approval := range tc.approvals {
				err := handler.db.ApproveFile(
					types.ProjectId(projectId),
					fileId,
					approval.UserId,
					approval.Destination,
					approval.Comment)
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
		name       string
		body       string
		fileId     string
		authUserId string
		s3client   s3.MockClient
		approvals  map[types.FileId]types.Approval

		expectedStatusCode int
		expectedBody       string
	}{
		{
			name:               "invalid body",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Invalid object. Failed to parse request body"}`,
		},
		{
			name:               "auth user missmatch",
			authUserId:         "user1",
			body:               `{"files_location":"s:/bucket1","max_file_size":100,"destination":"trusted","required_approvals":0,"user_id":"badUser"}`,
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Invalid object. user_id does not match Bearer token sub"}`,
		},
		{
			name:               "bad location",
			body:               `{"files_location":"s:/bucket1","max_file_size":100,"destination":"trusted","required_approvals":0}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "no approvals",
			body: `{"files_location":"s3://bucket1","max_file_size":100,"destination":"trusted","required_approvals":1}`,
			s3client: s3.MockClient{
				Buckets: map[s3.MockBucketName]s3.MockBucket{
					"bucket1": bucket1,
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Required 1 approvals for destination trusted but only had 0"}`,
		},
		{
			name:   "bad download destination",
			body:   `{"files_location":"s3://bucket1","max_file_size":100,"destination":"trusted","required_approvals":1}`,
			fileId: fileId1,
			s3client: s3.MockClient{
				Buckets: map[s3.MockBucketName]s3.MockBucket{
					"bucket1": bucket1,
				},
			},
			approvals: map[types.FileId]types.Approval{
				types.FileId(fileId1): {
					UserId:      "user1",
					Destination: "world",
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Required 1 approvals for destination trusted but only had 0"}`,
		},
		{
			name:   "above max body size",
			body:   `{"files_location":"s3://bucket1","max_file_size":1,"destination":"trusted","required_approvals":0}`,
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
			body:   `{"files_location":"s3://bucket1","max_file_size":100,"destination":"trusted","required_approvals":1}`,
			fileId: fileId1,
			s3client: s3.MockClient{
				Buckets: map[s3.MockBucketName]s3.MockBucket{
					"bucket1": bucket1,
				},
			},
			approvals: map[types.FileId]types.Approval{
				types.FileId(fileId1): {
					UserId:      "user1",
					Destination: "trusted",
				},
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `hello world`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &Handler{
				storage: s3.NewMock(tc.s3client),
				db:      inmemory.New(),
			}
			for fileId, approval := range tc.approvals {
				err := handler.db.ApproveFile(types.ProjectId(projectId), fileId, approval.UserId, approval.Destination, "")
				assert.NoError(t, err)
			}
			writer := httptest.NewRecorder()
			ctx, router := gin.CreateTestContext(writer)
			router.GET("/", func(ctx *gin.Context) {
				if tc.authUserId != "" {
					ctx.Set("sub", tc.authUserId)
				}
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
		name       string
		body       string
		fileId     string
		authUserId string

		expectedStatusCode int
		expectedBody       string
		expectedApprovals  int
	}{
		{
			name:               "invalid body",
			fileId:             "etag1",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Invalid object. Failed to parse request body"}`,
			expectedApprovals:  0,
		},
		{
			name:               "auth user missmatch",
			fileId:             "etag1",
			authUserId:         "user1",
			body:               `{"user_id":"badUser","destination":"trusted","comment":"good"}`,
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Invalid object. user_id does not match Bearer token sub"}`,
			expectedApprovals:  0,
		},
		{
			name:               "ok",
			fileId:             "etag1",
			authUserId:         "user1",
			body:               `{"user_id":"user1","destination":"trusted","comment":"good"}`,
			expectedStatusCode: http.StatusNoContent,
			expectedBody:       ``,
			expectedApprovals:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &Handler{
				db: inmemory.New(),
			}
			writer := httptest.NewRecorder()
			ctx, router := gin.CreateTestContext(writer)
			router.PUT("/", func(ctx *gin.Context) {
				ctx.Set("sub", tc.authUserId)
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
			fileApprovals := approvals[types.FileId(tc.fileId)]
			assert.Len(t, fileApprovals, tc.expectedApprovals)
			if tc.expectedApprovals > 0 {
				assert.Equal(t, types.UserId("user1"), fileApprovals[0].UserId)
				assert.Equal(t, types.Destination("trusted"), fileApprovals[0].Destination)
				assert.Equal(t, "good", fileApprovals[0].Comment)
			}
		})
	}
}

func TestRejectFileId(t *testing.T) {
	testCases := []struct {
		name       string
		body       string
		fileId     string
		authUserId string

		expectedStatusCode int
		expectedBody       string
		expectedEvents     int
	}{
		{
			name:               "invalid body",
			fileId:             "etag1",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Invalid object. Failed to parse request body"}`,
			expectedEvents:     0,
		},
		{
			name:               "auth user missmatch",
			fileId:             "etag1",
			authUserId:         "user1",
			body:               `{"user_id":"badUser","destination":"trusted","comment":"bad"}`,
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Invalid object. user_id does not match Bearer token sub"}`,
			expectedEvents:     0,
		},
		{
			name:               "ok",
			fileId:             "etag1",
			authUserId:         "user1",
			body:               `{"user_id":"user1","destination":"trusted","comment":"bad"}`,
			expectedStatusCode: http.StatusNoContent,
			expectedBody:       ``,
			expectedEvents:     1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &Handler{
				db: inmemory.New(),
			}
			writer := httptest.NewRecorder()
			ctx, router := gin.CreateTestContext(writer)
			router.PUT("/", func(ctx *gin.Context) {
				ctx.Set("sub", tc.authUserId)
				handler.PutProjectIdFilesFileIdReject(ctx, projectId, tc.fileId)
			})
			ctx.Request, _ = http.NewRequest(http.MethodPut, "/", strings.NewReader(tc.body))
			router.ServeHTTP(writer, ctx.Request)

			assert.Equal(t, tc.expectedStatusCode, writer.Code)
			body, err := io.ReadAll(writer.Body)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, string(body))

			events, err := handler.db.FileEvents(projectId)
			assert.NoError(t, err)
			fileEvents := events[types.FileId(tc.fileId)]
			assert.Len(t, fileEvents, tc.expectedEvents)
			if tc.expectedEvents > 0 {
				assert.Equal(t, types.UserId("user1"), fileEvents[0].UserId)
				assert.Equal(t, types.Destination("trusted"), fileEvents[0].Destination)
				assert.Equal(t, "bad", fileEvents[0].Comment)
			}
		})
	}
}

func TestGetEvents(t *testing.T) {
	handler := &Handler{
		db: inmemory.New(),
	}
	sourceEvents := []struct {
		action      types.EventAction
		fileId      types.FileId
		userId      types.UserId
		destination types.Destination
		comment     string
		runner      func(types.ProjectId, types.FileId, types.UserId, types.Destination, string) error
	}{
		{
			action:      "Approval",
			fileId:      "file1",
			userId:      "user2",
			destination: "trusted",
			comment:     "ok",
			runner:      handler.db.ApproveFile,
		},
		{
			action:      "Rejection",
			fileId:      "file2",
			userId:      "user1",
			destination: "trusted",
			comment:     "bad",
			runner:      handler.db.RejectFile,
		},
		{
			action:      "Download",
			fileId:      "file1",
			userId:      "user1",
			destination: "trusted",
			comment:     "results",
			runner:      handler.db.DownloadFile,
		},
	}

	// Log the events
	for _, e := range sourceEvents {
		err := e.runner(types.ProjectId(projectId), e.fileId, e.userId, e.destination, e.comment)
		assert.NoError(t, err)
	}

	writer := httptest.NewRecorder()
	ctx, router := gin.CreateTestContext(writer)
	router.GET("/", func(ctx *gin.Context) {
		handler.GetProjectIdEvents(ctx, projectId)
	})
	ctx.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(writer, ctx.Request)

	assert.Equal(t, http.StatusOK, writer.Code)
	body, err := io.ReadAll(writer.Body)
	assert.NoError(t, err)

	// Unmarshal events from the response body
	var events []struct {
		Datetime    time.Time `json:"datetime"`
		Action      string    `json:"action"`
		FileId      string    `json:"file_id"`
		UserId      string    `json:"user_id"`
		Destination string    `json:"destination"`
		Comment     string    `json:"comment"`
	}
	assert.NoError(t, json.Unmarshal(body, &events))
	assert.Len(t, events, len(sourceEvents))

	// Verify content of the events
	for i, e := range events {
		assert.Equal(t, string(sourceEvents[i].action), e.Action)
		assert.Equal(t, string(sourceEvents[i].fileId), e.FileId)
		assert.Equal(t, string(sourceEvents[i].userId), e.UserId)
		assert.Equal(t, string(sourceEvents[i].destination), e.Destination)
		assert.Equal(t, sourceEvents[i].comment, e.Comment)
	}

	// Verify ordering of the events
	for i := 1; i < len(events); i++ {
		assert.True(t, events[i].Datetime.After(events[i-1].Datetime))
	}
}
