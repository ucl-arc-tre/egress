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

// newTestHandler creates a temporary directory, writes the given files into it,
// and returns a Handler rooted at that directory.
func newTestHandler(t *testing.T, files map[string]string) *Handler {
	t.Helper()
	dir := t.TempDir()
	for key, content := range files {
		path := filepath.Join(dir, key)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return New(dir)
}

func etag(t *testing.T, s *Handler, fileKey string) string {
	t.Helper()
	info, err := os.Stat(filepath.Join(s.path, fileKey))
	require.NoError(t, err)
	return computeETag(fileKey, info)
}

func TestGetFiles(t *testing.T) {
	files := map[string]string{
		"a/foo.txt":      "hello",
		"a/bar.txt":      "world",
		"a/b/nested.txt": "nested content",
		"b/baz.txt":      "other",
	}
	prefix := "a/"
	traversalPrefix := "../"
	absolutePrefix := "/etc"
	maliciousPrefix := "a/../../../etc/"

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
			expectedCount:      4,
		},
		{
			name:               "prefix filters files, including nested",
			prefix:             &prefix,
			expectedStatusCode: http.StatusOK,
			expectedCount:      3,
		},
		{
			name:               "don't allow path traversal prefix",
			prefix:             &traversalPrefix,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "don't allow absolute path prefix",
			prefix:             &absolutePrefix,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "don't allow malicious path prefix",
			prefix:             &maliciousPrefix,
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(t, files)

			url := "/files"
			if tc.prefix != nil {
				url += "?prefix=" + *tc.prefix
			}

			writer := httptest.NewRecorder()
			_, router := gin.CreateTestContext(writer)
			RegisterHandlers(router, h)

			req, _ := http.NewRequest(http.MethodGet, url, nil)
			router.ServeHTTP(writer, req)

			assert.Equal(t, tc.expectedStatusCode, writer.Code)
			if writer.Code == http.StatusOK {
				var resp ListFilesResponse
				require.NoError(t, json.NewDecoder(writer.Body).Decode(&resp))
				assert.Len(t, resp.Files, tc.expectedCount)
				assert.Equal(t, tc.expectedCount, *resp.FileCount)
			}
		})
	}
}

func TestGetFile(t *testing.T) {
	files := map[string]string{
		"data.txt":      "hello world",
		"subdir/nested": "nested content",
	}

	testCases := []struct {
		name               string
		key                string
		ifMatch            func(s *Handler) string
		expectedStatusCode int
		expectedBody       string
	}{
		{
			name:               "ok",
			key:                "data.txt",
			ifMatch:            func(s *Handler) string { return etag(t, s, "data.txt") },
			expectedStatusCode: http.StatusOK,
			expectedBody:       "hello world",
		},
		{
			name:               "nested file",
			key:                "subdir/nested",
			ifMatch:            func(s *Handler) string { return etag(t, s, "subdir/nested") },
			expectedStatusCode: http.StatusOK,
			expectedBody:       "nested content",
		},
		{
			name:               "not found",
			key:                "missing.txt",
			ifMatch:            func(s *Handler) string { return `"doesnotmatter"` },
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "etag mismatch",
			key:                "data.txt",
			ifMatch:            func(s *Handler) string { return `"wrongetag"` },
			expectedStatusCode: http.StatusPreconditionFailed,
		},
		{
			name:               "path traversal",
			key:                "../secret.txt",
			ifMatch:            func(s *Handler) string { return `"x"` },
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "absolute path",
			key:                "/etc/passwd",
			ifMatch:            func(s *Handler) string { return `"x"` },
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "empty key",
			key:                "",
			ifMatch:            func(s *Handler) string { return `"x"` },
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "directory key",
			key:                "subdir",
			ifMatch:            func(s *Handler) string { return etag(t, s, "subdir/nested") },
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "unquoted If-Match",
			key:                "data.txt",
			ifMatch:            func(s *Handler) string { return "unquoted" },
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(t, files)

			writer := httptest.NewRecorder()
			_, router := gin.CreateTestContext(writer)
			RegisterHandlers(router, h)

			req, _ := http.NewRequest(http.MethodGet, "/file?key="+tc.key, nil)
			req.Header.Set("If-Match", tc.ifMatch(h))
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
