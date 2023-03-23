package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStatsReaderNil(t *testing.T) {
	a := assert.New(t)
	unknownUris := []string{
		"abc://xxx",
		"def://yyy",
		"socks://vvvv",
	}

	for _, uri := range unknownUris {
		reader, err := NewStatsReader(uri)
		if a.Error(err) {
			a.Nil(reader)
		}
	}
}

func TestNewStatsReader_Safe(t *testing.T) {
	testCases := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{
			name: "tcp",
			uri:  "tcp://xxx:123",
		},
		{
			name: "http",
			uri:  "http://10.1.1.2:1234",
		},
		{
			name: "https",
			uri:  "https://example.com:8443",
		},
		{
			name:    "file",
			uri:     "file:///tmp/uwsgi.json",
			wantErr: true,
		},
		{
			name:    "unix",
			uri:     "unix:///tmp/uwsgi.sock",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewStatsReader(tc.uri, WithRequireSafeScheme(true))
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
