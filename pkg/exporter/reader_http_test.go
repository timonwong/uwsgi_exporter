package exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPStatsReader_Read(t *testing.T) {
	a := assert.New(t)

	s := newUwsgiStatsServer(sampleUwsgiStatsJSON)
	defer s.Close()

	uri := s.URL
	reader, err := NewStatsReader(uri, someTimeout)
	a.NoError(err)

	_, ok := reader.(*httpStatsReader)
	a.True(ok)

	uwsgiStats, err := reader.Read()
	a.NoError(err)

	a.Equal(uwsgiStats.Version, "2.0.12")
}
