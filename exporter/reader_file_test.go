package exporter

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileStatsReader_Read(t *testing.T) {
	a := assert.New(t)

	filename, err := filepath.Abs("../testdata/sample.json")
	if err != nil {
		panic(err)
	}

	uri := "file://" + filename
	duration := 5 * time.Second

	reader, err := NewStatsReader(uri, duration)
	a.NoError(err)

	uwsgiStats, err := reader.Read()
	a.NoError(err)

	a.Equal(uwsgiStats.Version, "2.0.12")
}
