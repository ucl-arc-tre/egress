package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/db/inmemory"
	"github.com/ucl-arc-tre/egress/internal/storage/generic"
	"github.com/ucl-arc-tre/egress/internal/types"
)

func TestGetFilesGeneric(t *testing.T) {
	testCases := []struct {
		name          string
		body          string
		genericClient generic.MockClient
		approvals     map[types.FileId]types.UserId

		expectedStatusCode int
		expectedBody       string
	}{
		{
			name:               "invalid body",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Invalid object. Failed to parse request body"}`,
		},
		{
			name: "list error",
			body: `{"files_location":"http://storage.local"}`,
			genericClient: generic.MockClient{
				ForceListErr: errors.New("server error"),
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:               "invalid location",
			body:               `{"files_location":"://bucket"}`,
			genericClient:      generic.MockClient{},
			expectedStatusCode: 520,
		},
		{
			name: "ok",
			body: `{"files_location":"http://storage.local"}`,
			genericClient: generic.MockClient{
				Files: []generic.MockFile{
					{
						Key:     "file1",
						ETag:    `"abc100"`,
						Content: "hello world",
					},
				},
			},
			approvals: map[types.FileId]types.UserId{
				"abc100": "user1",
				"abc200": "user1",
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `[{"approvals":["user1"],"file_name":"file1","id":"abc100","size":11}]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &Handler{
				storage: generic.NewWithMock(&tc.genericClient),
				db:      inmemory.New(),
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

func TestGetFileIdGeneric(t *testing.T) {
	fileId1 := "abc100"
	genericFile1 := generic.MockFile{
		Key:     "file1",
		ETag:    fmt.Sprintf(`"%s"`, fileId1),
		Content: "hello world",
	}
	testCases := []struct {
		name          string
		body          string
		fileId        string
		genericClient generic.MockClient
		approvals     map[types.FileId]types.UserId

		expectedStatusCode int
		expectedBody       string
	}{
		{
			name:               "invalid body",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Invalid object. Failed to parse request body"}`,
		},
		{
			name:   "list error on get",
			body:   `{"files_location":"http://storage.local","max_file_size":100,"required_approvals":0}`,
			fileId: fileId1,
			genericClient: generic.MockClient{
				ForceListErr: errors.New("server error"),
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "no approvals",
			body: `{"files_location":"http://storage.local","max_file_size":100,"required_approvals":1}`,
			genericClient: generic.MockClient{
				Files: []generic.MockFile{genericFile1},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Required 1 approvals but only had 0"}`,
		},
		{
			name:   "above max body size",
			body:   `{"files_location":"http://storage.local","max_file_size":1,"required_approvals":0}`,
			fileId: fileId1,
			genericClient: generic.MockClient{
				Files: []generic.MockFile{genericFile1},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"message":"Size [11] was greater than max [1]"}`,
		},
		{
			name:   "ok",
			body:   `{"files_location":"http://storage.local","max_file_size":100,"required_approvals":1}`,
			fileId: fileId1,
			genericClient: generic.MockClient{
				Files: []generic.MockFile{genericFile1},
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
				storage: generic.NewWithMock(&tc.genericClient),
				db:      inmemory.New(),
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
