package rqlite

import (
	"errors"
	"testing"
	"time"

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

func TestParseDatetimeSubsecFormat(t *testing.T) {
	dt, err := parseDatetime("2026-04-24 10:44:48.442")

	assert.NoError(t, err)
	assert.Equal(t, time.Date(2026, 4, 24, 10, 44, 48, 442_000_000, time.UTC), dt)
}

func TestParseDatetimeLegacyFormat(t *testing.T) {
	dt, err := parseDatetime("2025-12-04 22:08:04")

	assert.NoError(t, err)
	assert.Equal(t, time.Date(2025, 12, 4, 22, 8, 4, 0, time.UTC), dt)
}
