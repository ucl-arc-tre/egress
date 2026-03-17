package server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubETagGenerator returns a deterministic ETag based only on the key
type stubETagGenerator struct{}

func (g stubETagGenerator) MakeETag(key string, _ fs.FileInfo) (string, error) {
	return fmt.Sprintf(`"stub-%s"`, key), nil
}

func newTestHandlerWithOpts(t *testing.T, files map[string]string, opts ...Option) *Handler {
	t.Helper()
	dir := t.TempDir()
	for key, content := range files {
		path := filepath.Join(dir, key)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return New(dir, opts...)
}

func TestCustomETagGenerator_GetFiles(t *testing.T) {
	files := map[string]string{
		"a.txt": "hello",
		"b.txt": "world",
	}
	h := newTestHandlerWithOpts(t, files, WithETagGenerator(stubETagGenerator{}))

	writer := httptest.NewRecorder()
	_, router := gin.CreateTestContext(writer)
	RegisterHandlers(router, h)

	req, _ := http.NewRequest(http.MethodGet, "/files", nil)
	router.ServeHTTP(writer, req)

	require.Equal(t, http.StatusOK, writer.Code)

	var resp ListFilesResponse
	require.NoError(t, json.NewDecoder(writer.Body).Decode(&resp))
	require.Len(t, resp.Files, 2)

	keys := make([]string, 0, len(resp.Files))
	for _, f := range resp.Files {
		keys = append(keys, f.Key)
		expected := fmt.Sprintf(`"stub-%s"`, f.Key)
		assert.Equal(t, expected, f.Etag, "GetFiles should use the custom ETagGenerator for key %s", f.Key)
	}
	assert.ElementsMatch(t, []string{"a.txt", "b.txt"}, keys)
}

func TestCustomETagGenerator_GetFile(t *testing.T) {
	files := map[string]string{
		"data.txt": "some content",
	}
	h := newTestHandlerWithOpts(t, files, WithETagGenerator(stubETagGenerator{}))

	expectedETag := `"stub-data.txt"`

	writer := httptest.NewRecorder()
	_, router := gin.CreateTestContext(writer)
	RegisterHandlers(router, h)

	req, _ := http.NewRequest(http.MethodGet, "/file?key=data.txt", nil)
	req.Header.Set("If-Match", expectedETag)
	router.ServeHTTP(writer, req)

	require.Equal(t, http.StatusOK, writer.Code)

	body, err := io.ReadAll(writer.Body)
	require.NoError(t, err)
	assert.Equal(t, "some content", string(body))
	assert.Equal(t, expectedETag, writer.Header().Get("ETag"), "GetFile response should contain the custom ETag header")
}

func TestCustomETagGenerator_GetFileRejectsDefaultETag(t *testing.T) {
	files := map[string]string{
		"data.txt": "some content",
	}
	h := newTestHandlerWithOpts(t, files, WithETagGenerator(stubETagGenerator{}))

	// Compute the ETag that DefaultETagGenerator would produce — this should
	// NOT match because we configured the stub generator.
	info, err := os.Stat(filepath.Join(h.rootDirPath, "data.txt"))
	require.NoError(t, err)
	defaultETag, err := DefaultETagGenerator{}.MakeETag("data.txt", info)
	require.NoError(t, err)

	writer := httptest.NewRecorder()
	_, router := gin.CreateTestContext(writer)
	RegisterHandlers(router, h)

	req, _ := http.NewRequest(http.MethodGet, "/file?key=data.txt", nil)
	req.Header.Set("If-Match", defaultETag)
	router.ServeHTTP(writer, req)

	assert.Equal(t, http.StatusPreconditionFailed, writer.Code,
		"GetFile should reject the default ETag when a custom ETagGenerator is configured")
}

func TestDefaultETagGeneratorUsedWhenNoOptionProvided(t *testing.T) {
	files := map[string]string{
		"file.txt": "content",
	}
	h := newTestHandlerWithOpts(t, files) // no WithETagGenerator option

	info, err := os.Stat(filepath.Join(h.rootDirPath, "file.txt"))
	require.NoError(t, err)
	expectedETag, err := DefaultETagGenerator{}.MakeETag("file.txt", info)
	require.NoError(t, err)

	writer := httptest.NewRecorder()
	_, router := gin.CreateTestContext(writer)
	RegisterHandlers(router, h)

	req, _ := http.NewRequest(http.MethodGet, "/file?key=file.txt", nil)
	req.Header.Set("If-Match", expectedETag)
	router.ServeHTTP(writer, req)

	assert.Equal(t, http.StatusOK, writer.Code,
		"Handler should use DefaultETagGenerator when no custom option is provided")
}
