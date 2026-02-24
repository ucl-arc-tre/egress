package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer creates a temporary directory, writes the given files into it,
// and returns a Server rooted at that directory.
func newTestServer(t *testing.T, files map[string]string) *Server {
	t.Helper()
	dir := t.TempDir()
	for key, content := range files {
		path := filepath.Join(dir, key)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return New(dir)
}

// etag returns the ETag for a file in the server's directory.
func etag(t *testing.T, s *Server, key string) string {
	t.Helper()
	info, err := os.Stat(filepath.Join(s.path, key))
	require.NoError(t, err)
	return computeETag(key, info)
}

func TestGetFiles(t *testing.T) {
	files := map[string]string{
		"a/foo.txt": "hello",
		"a/bar.txt": "world",
		"b/baz.txt": "other",
	}
	prefix := "a/"

	testCases := []struct {
		name               string
		prefix             *string
		expectedStatusCode int
		expectedCount      int
	}{
		{
			name:               "no prefix returns all files",
			prefix:             nil,
			expectedStatusCode: http.StatusOK,
			expectedCount:      3,
		},
		{
			name:               "prefix filters files",
			prefix:             &prefix,
			expectedStatusCode: http.StatusOK,
			expectedCount:      2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestServer(t, files)

			url := "/files"
			if tc.prefix != nil {
				url += "?prefix=" + *tc.prefix
			}

			writer := httptest.NewRecorder()
			_, router := gin.CreateTestContext(writer)
			RegisterHandlers(router, s)

			req, _ := http.NewRequest(http.MethodGet, url, nil)
			router.ServeHTTP(writer, req)

			assert.Equal(t, tc.expectedStatusCode, writer.Code)
			var resp ListFilesResponse
			require.NoError(t, json.NewDecoder(writer.Body).Decode(&resp))
			assert.Len(t, resp.Files, tc.expectedCount)
			assert.Equal(t, tc.expectedCount, *resp.FileCount)
		})
	}
}

func TestGetFile(t *testing.T) {
	files := map[string]string{
		"data.txt": "hello world",
	}

	testCases := []struct {
		name               string
		key                string
		ifMatch            func(s *Server) string
		expectedStatusCode int
		expectedBody       string
	}{
		{
			name:               "ok",
			key:                "data.txt",
			ifMatch:            func(s *Server) string { return etag(t, s, "data.txt") },
			expectedStatusCode: http.StatusOK,
			expectedBody:       "hello world",
		},
		{
			name:               "not found",
			key:                "missing.txt",
			ifMatch:            func(s *Server) string { return `"doesnotmatter"` },
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "etag mismatch",
			key:                "data.txt",
			ifMatch:            func(s *Server) string { return `"wrongetag"` },
			expectedStatusCode: http.StatusPreconditionFailed,
		},
		{
			name:               "path traversal",
			key:                "../secret.txt",
			ifMatch:            func(s *Server) string { return `"x"` },
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestServer(t, files)

			writer := httptest.NewRecorder()
			_, router := gin.CreateTestContext(writer)
			RegisterHandlers(router, s)

			req, _ := http.NewRequest(http.MethodGet, "/file?key="+tc.key, nil)
			req.Header.Set("If-Match", tc.ifMatch(s))
			router.ServeHTTP(writer, req)

			assert.Equal(t, tc.expectedStatusCode, writer.Code)
			if tc.expectedBody != "" {
				body, err := io.ReadAll(writer.Body)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedBody, string(body))
			}
		})
	}
}
