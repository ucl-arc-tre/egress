package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	baseUrl        = "http://localhost:8080/v0"
	requestTimeout = 1 * time.Second
)

func TestHello(t *testing.T) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%v/hello", baseUrl), nil)
	assert.NoError(t, err)

	client := &http.Client{Timeout: requestTimeout}
	response, err := client.Do(request)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
}
