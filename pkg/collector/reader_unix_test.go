package collector

import (
	"context"
	"reflect"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestUnixStatsReader_Read(t *testing.T) {
	t.Parallel()

	// Setup a local UDS server for testing
	ls, err := newLocalServer(t, "unix")
	assert.NoError(t, err)

	ch := make(chan error, 1)

	ls.buildup(justWriteHandler(sampleUwsgiStatsJSON, ch))

	uri := "unix://" + ls.Listener.Addr().String()
	reader, err := NewStatsReader(uri)
	assert.NoError(t, err)

	assert.Equal(t, reflect.TypeOf(&unixStatsReader{}).String(), reflect.TypeOf(reader).String())

	ctx, cancel := context.WithTimeout(context.Background(), someTimeout)
	defer cancel()

	uwsgiStats, err := reader.Read(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "2.0.12", uwsgiStats.Version)

	for err := range ch {
		assert.NoError(t, err)
	}
}
