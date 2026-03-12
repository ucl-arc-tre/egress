package generic

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ucl-arc-tre/egress/internal/types"
)

//go:generate go tool oapi-codegen -generate types,client -package generic -o client.gen.go ../../../api/storage.yaml

// Gets a storage API client for the given location
// Facilitates mocking the client
type apiClientGetter interface {
	Get(location types.LocationURI) (ClientWithResponsesInterface, error)
}

type httpAPIClientGetter struct {
	http *http.Client
}

func (g *httpAPIClientGetter) Get(location types.LocationURI) (ClientWithResponsesInterface, error) {
	return newAPIClient(location, g.http)
}

// Creates a storage API client using the generated
// code and the provided http.Client
func newAPIClient(location types.LocationURI, http *http.Client) (*ClientWithResponses, error) {
	serverURL, err := locationToServerURL(location)
	if err != nil {
		return nil, types.NewErrServerF("[generic] invalid storage location: %w", err)
	}
	apiClient, err := NewClientWithResponses(
		serverURL.String(),
		WithHTTPClient(http),
	)
	if err != nil {
		return nil, types.NewErrServerF("[generic] failed to create API client: %w", err)
	}
	return apiClient, nil
}

func locationToServerURL(location types.LocationURI) (url.URL, error) {
	u := url.URL(location)
	if u.Host == "" {
		return url.URL{}, fmt.Errorf("does not contain a host")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return url.URL{}, fmt.Errorf("unsupported scheme %q (must be http or https)", u.Scheme)
	}
	// Strip any trailing slash to avoid double slashes in appended paths
	u.Path = strings.TrimRight(u.Path, "/")
	return u, nil
}
