package collector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnixStatsReader_Read(t *testing.T) {
	a := assert.New(t)

	// Setup a local UDS server for testing
	ls, err := newLocalServer("unix")
	a.NoError(err)

	defer ls.teardown()
	ch := make(chan error, 1)

	ls.buildup(justwriteHandler(sampleUwsgiStatsJSON, ch))

	uri := "unix://" + ls.Listener.Addr().String()
	reader, err := NewStatsReader(uri)
	a.NoError(err)

	_, ok := reader.(*unixStatsReader)
	a.True(ok)

	ctx, cancel := context.WithTimeout(context.Background(), someTimeout)
	defer cancel()

	uwsgiStats, err := reader.Read(ctx)
	a.NoError(err)

	a.Equal(uwsgiStats.Version, "2.0.12")

	for err := range ch {
		a.NoError(err)
	}
}
