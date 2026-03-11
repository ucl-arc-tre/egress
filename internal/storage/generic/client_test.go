package generic

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ucl-arc-tre/egress/internal/types"
)

func TestLocationToServerURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{
			name:      "Valid HTTPS",
			input:     "https://data.local/v0",
			expected:  "https://data.local/v0",
			expectErr: false,
		},
		{
			name:      "Valid HTTP",
			input:     "http://data.local/v0",
			expected:  "http://data.local/v0",
			expectErr: false,
		},
		{
			name:      "Trailing slash stripped",
			input:     "http://localhost/v0/",
			expected:  "http://localhost/v0",
			expectErr: false,
		},
		{
			name:      "Missing host",
			input:     "https:///v0",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "Unsupported scheme",
			input:     "s3://egress",
			expected:  "",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, _ := url.Parse(tc.input)

			result, err := locationToServerURL(types.LocationURI(*u))
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.String())
		})
	}
}
