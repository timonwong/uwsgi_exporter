package collector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPStatsReader_Read(t *testing.T) {
	t.Parallel()

	a := assert.New(t)

	s := newUwsgiStatsServer(sampleUwsgiStatsJSON)
	defer s.Close()

	uri := s.URL
	reader, err := NewStatsReader(uri)
	a.NoError(err)

	a.IsType(&httpStatsReader{}, reader)

	ctx, cancel := context.WithTimeout(context.Background(), someTimeout)
	defer cancel()

	uwsgiStats, err := reader.Read(ctx)
	a.NoError(err)

	a.Equal(uwsgiStats.Version, "2.0.12")
}
