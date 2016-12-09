package exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPStatsReader_Read(t *testing.T) {
	a := assert.New(t)

	// Setup a local TCP server for testing
	ls, err := newLocalServer("tcp")
	a.NoError(err)

	defer ls.teardown()
	ch := make(chan error, 1)

	ls.buildup(justwriteHandler(sampleUwsgiStatsJSON, ch))

	uri := "tcp://" + ls.Listener.Addr().String()
	reader, err := NewStatsReader(uri, someTimeout)
	a.NoError(err)

	_, ok := reader.(*tcpStatsReader)
	a.True(ok)

	uwsgiStats, err := reader.Read()
	a.NoError(err)

	a.Equal(uwsgiStats.Version, "2.0.12")

	for err := range ch {
		a.NoError(err)
	}
}
