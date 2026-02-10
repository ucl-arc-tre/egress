package rqlite

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/types"
)

func TestAuthUrls(t *testing.T) {
	connURL, err := buildAuthURL("http://dbserver.local", "foo", "bar")

	assert.NoError(t, err)
	assert.Equal(t, "http://foo:bar@dbserver.local", connURL) // pragma: allowlist secret
}

func TestAuthUrlsBadURL(t *testing.T) {
	_, err := buildAuthURL(":dbserver.local", "foo", "bar")

	assert.Error(t, err)
}

func TestUnifyErrorsWithOpError(t *testing.T) {
	err := unifyErrors("failed", errors.New("operation error"), nil)

	assert.Error(t, err)
	assert.Equal(t, types.ErrServer, errors.Unwrap(err))
}

func TestUnifyErrorsWithDBError(t *testing.T) {
	err := unifyErrors("failed", nil, errors.New("database error"))

	assert.Error(t, err)
	assert.Equal(t, types.ErrServer, errors.Unwrap(err))
}
