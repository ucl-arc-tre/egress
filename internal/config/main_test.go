package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerAddressSetPort(t *testing.T) {
	t.Setenv("PORT", "1234")
	assert.Equal(t, ":1234", ServerAddress())
}

func TestServerAddressDefault(t *testing.T) {
	assert.Equal(t, ":8080", ServerAddress())
}
