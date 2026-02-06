package rqlite

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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
	operr := errors.New("operation error")
	err := unifyErrors("failed", operr, nil)

	assert.Error(t, err)
	assert.Equal(t, operr, errors.Unwrap(err))
}

func TestUnifyErrorsWithDBError(t *testing.T) {
	dberr := errors.New("database error")
	err := unifyErrors("failed", nil, dberr)

	assert.Error(t, err)
	assert.Equal(t, dberr, errors.Unwrap(err))
}
