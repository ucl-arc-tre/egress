package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/db/inmemory"
)

func TestInMemoryProvider(t *testing.T) {
	cfg := config.DBConfigBundle{
		Provider: string(DBProviderInMemory),
	}
	db, err := Provider(cfg)
	assert.NoError(t, err)
	assert.IsType(t, &inmemory.DB{}, db)
}

func TestUnsupportedProvider(t *testing.T) {
	cfg := config.DBConfigBundle{
		Provider: "blah",
	}
	db, err := Provider(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
}
