package collector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnixStatsReader_Read(t *testing.T) {
	t.Parallel()

	a := assert.New(t)

	// Setup a local UDS server for testing
	ls, err := newLocalServer(t, "unix")
	a.NoError(err)

	ch := make(chan error, 1)

	ls.buildup(justWriteHandler(sampleUwsgiStatsJSON, ch))

	uri := "unix://" + ls.Listener.Addr().String()
	reader, err := NewStatsReader(uri)
	a.NoError(err)

	a.IsType(&unixStatsReader{}, reader)

	ctx, cancel := context.WithTimeout(context.Background(), someTimeout)
	defer cancel()

	uwsgiStats, err := reader.Read(ctx)
	a.NoError(err)

	a.Equal(uwsgiStats.Version, "2.0.12")

	for err := range ch {
		a.NoError(err)
	}
}
