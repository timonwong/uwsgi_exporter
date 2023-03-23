package collector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileStatsReader_Read(t *testing.T) {
	a := assert.New(t)

	uri := "file://" + sampleUwsgiStatsFileName

	reader, err := NewStatsReader(uri)
	a.NoError(err)

	a.IsType(&fileStatsReader{}, reader)

	ctx, cancel := context.WithTimeout(context.Background(), someTimeout)
	defer cancel()

	uwsgiStats, err := reader.Read(ctx)
	a.NoError(err)

	a.Equal(uwsgiStats.Version, "2.0.12")
}
