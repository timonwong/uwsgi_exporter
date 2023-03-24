package collector

import (
	"context"
	"reflect"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestHTTPStatsReader_Read(t *testing.T) {
	t.Parallel()

	s := newUwsgiStatsServer(sampleUwsgiStatsJSON)
	defer s.Close()

	uri := s.URL
	reader, err := NewStatsReader(uri)
	assert.NoError(t, err)

	assert.Equal(t, reflect.TypeOf(&httpStatsReader{}).String(), reflect.TypeOf(reader).String())

	ctx, cancel := context.WithTimeout(context.Background(), someTimeout)
	defer cancel()

	uwsgiStats, err := reader.Read(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "2.0.12", uwsgiStats.Version)
}
