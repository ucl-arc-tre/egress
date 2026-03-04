package generic

import (
	"context"
	"errors"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ucl-arc-tre/egress/internal/types"
)

var (
	file1 = mockFile{
		Key:            "project-1/data.csv",
		ETag:           `"abc123"`,
		LastModifiedAt: time.Date(2026, 3, 4, 16, 4, 0, 0, time.UTC),
		Content:        "id,result\n1,4.16\n",
	}
	file2 = mockFile{
		Key:            "project-1/report.txt",
		ETag:           `"def456"`,
		LastModifiedAt: time.Date(2026, 3, 4, 8, 22, 0, 0, time.UTC),
		Content:        "Hello, World!\n\x00\xff",
	}
	location types.LocationURI
)

func TestMain(t *testing.T) {
	u, err := url.Parse("https://data.local/v0")
	require.NoError(t, err)
	location = types.LocationURI(*u)
}

func TestListReturnsAllFiles(t *testing.T) {
	ms := newWithMock(&mockClient{
		Files: []mockFile{file1, file2},
	})
	files, err := ms.List(context.Background(), location)

	require.NoError(t, err)
	require.Len(t, files, 2)

	assert.Equal(t, file1.Key, files[0].Name)
	assert.Equal(t, types.FileId("abc123"), files[0].Id)
	assert.Equal(t, int64(len(file1.Content)), files[0].Size)
	assert.Equal(t, file1.LastModifiedAt, files[0].LastModifiedAt)

	assert.Equal(t, file2.Key, files[1].Name)
	assert.Equal(t, types.FileId("def456"), files[1].Id)
	assert.Equal(t, int64(len(file2.Content)), files[1].Size)
}

func TestListEmptyLocation(t *testing.T) {
	ms := newWithMock(&mockClient{
		Files: []mockFile{},
	})
	files, err := ms.List(context.Background(), location)

	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestListETagQuotesAreStripped(t *testing.T) {
	f := mockFile{Key: "a.data", ETag: `"ab12"`, Content: "hello"}
	ms := newWithMock(&mockClient{
		Files: []mockFile{f},
	})
	files, err := ms.List(context.Background(), location)

	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, types.FileId("ab12"), files[0].Id)
}

func TestListPropagatesClientError(t *testing.T) {
	ms := newWithMock(&mockClient{
		ForceListErr: errors.New("network error"),
	})
	_, err := ms.List(context.Background(), location)

	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrServer)
}

func TestGetFileContentByFileId(t *testing.T) {
	ms := newWithMock(&mockClient{
		Files: []mockFile{file1, file2},
	})
	f, err := ms.Get(context.Background(), location, types.FileId("abc123"))

	require.NoError(t, err)
	require.NotNil(t, f)
	defer f.Content.Close() // nolint:errcheck

	data, err := io.ReadAll(f.Content)
	require.NoError(t, err)
	assert.Equal(t, file1.Content, string(data))
	assert.Equal(t, int64(len(file1.Content)), f.Size)
}

func TestGetFileIdNotFound(t *testing.T) {
	ms := newWithMock(&mockClient{
		Files: []mockFile{file1},
	})
	_, err := ms.Get(context.Background(), location, types.FileId("nonexistent"))

	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrNotFound)
}

func TestGetPropagatesClientListError(t *testing.T) {
	ms := newWithMock(&mockClient{
		ForceListErr: errors.New("network error"),
	})
	_, err := ms.Get(context.Background(), location, types.FileId("abc123"))

	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrServer)
}

func TestGetPropagatesClientError(t *testing.T) {
	ms := newWithMock(&mockClient{
		Files:       []mockFile{file1},
		ForceGetErr: errors.New("network error"),
	})
	_, err := ms.Get(context.Background(), location, types.FileId("abc123"))

	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrServer)
}

func TestStripQuotes(t *testing.T) {
	assert.Equal(t, "abc", stripQuotes(`abc`))
	assert.Equal(t, "abc", stripQuotes(`"abc"`))
	assert.Equal(t, "", stripQuotes(`""`))
	assert.Equal(t, "", stripQuotes(``))
}
