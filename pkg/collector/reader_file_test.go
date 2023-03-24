package collector

import (
	"context"
	"reflect"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestFileStatsReader_Read(t *testing.T) {
	t.Parallel()

	uri := "file://" + sampleUwsgiStatsFileName

	reader, err := NewStatsReader(uri)
	assert.NoError(t, err)

	assert.Equal(t, reflect.TypeOf(&fileStatsReader{}).String(), reflect.TypeOf(reader).String())

	ctx, cancel := context.WithTimeout(context.Background(), someTimeout)
	defer cancel()

	uwsgiStats, err := reader.Read(ctx)
	assert.NoError(t, err)

	assert.Equal(t, uwsgiStats.Version, "2.0.12")
}
