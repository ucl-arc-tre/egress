package generic

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ucl-arc-tre/egress/internal/types"
)

//go:generate go tool oapi-codegen -generate types,client -package generic -o client.gen.go ../../../api/storage.yaml

// Gets a client for the given location
// Used primarily for mocking the client
type clientGetter interface {
	Get(location types.LocationURI) (ClientWithResponsesInterface, error)
}

type httpClientGetter struct {
	httpClient *http.Client
}

func (g *httpClientGetter) Get(location types.LocationURI) (ClientWithResponsesInterface, error) {
	return newClient(location, g.httpClient)
}

// Creates a generic storage client using the generated OpenAPI
// code and the provided http.Client
func newClient(location types.LocationURI, httpClient *http.Client) (*ClientWithResponses, error) {
	serverURL, err := locationToServerURL(location)
	if err != nil {
		return nil, types.NewErrServerF("[generic] invalid storage location: %w", err)
	}
	client, err := NewClientWithResponses(
		serverURL,
		WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, types.NewErrServerF("[generic] failed to create client: %w", err)
	}
	return client, nil
}

func locationToServerURL(location types.LocationURI) (string, error) {
	u := url.URL(location)
	if u.Host == "" {
		return "", fmt.Errorf("does not contain a host")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", fmt.Errorf("unsupported scheme %q (must be http or https)", u.Scheme)
	}
	// Strip trailing any slash to avoid double slashes in appended paths
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), nil
}
