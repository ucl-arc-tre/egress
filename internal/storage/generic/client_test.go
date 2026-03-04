package generic

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ucl-arc-tre/egress/internal/types"
)

func TestLocationToServerURLValidHTTPS(t *testing.T) {
	u, _ := url.Parse("https://data.local/v0")

	result, err := locationToServerURL(types.LocationURI(*u))

	require.NoError(t, err)
	assert.Equal(t, "https://data.local/v0", result)
}

func TestLocationToServerURLValidHTTP(t *testing.T) {
	u, _ := url.Parse("http://data.local/v0")

	result, err := locationToServerURL(types.LocationURI(*u))

	require.NoError(t, err)
	assert.Equal(t, "http://data.local/v0", result)
}

func TestLocationToServerURLTrailingSlashStripped(t *testing.T) {
	u, _ := url.Parse("http://localhost/v0/")

	result, err := locationToServerURL(types.LocationURI(*u))

	require.NoError(t, err)
	assert.Equal(t, "http://localhost/v0", result)
}

func TestLocationToServerURLMissingHost(t *testing.T) {
	u, _ := url.Parse("https:///v0")

	_, err := locationToServerURL(types.LocationURI(*u))
	assert.Error(t, err)
}

func TestLocationToServerURLUnsupportedScheme(t *testing.T) {
	u, _ := url.Parse("s3://egress")

	_, err := locationToServerURL(types.LocationURI(*u))
	assert.Error(t, err)
}
