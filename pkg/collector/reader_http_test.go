package collector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPStatsReader_Read(t *testing.T) {
	a := assert.New(t)

	s := newUwsgiStatsServer(sampleUwsgiStatsJSON)
	defer s.Close()

	uri := s.URL
	reader, err := NewStatsReader(uri)
	a.NoError(err)

	_, ok := reader.(*httpStatsReader)
	a.True(ok)

	ctx, cancel := context.WithTimeout(context.Background(), someTimeout)
	defer cancel()

	uwsgiStats, err := reader.Read(ctx)
	a.NoError(err)

	a.Equal(uwsgiStats.Version, "2.0.12")
}
