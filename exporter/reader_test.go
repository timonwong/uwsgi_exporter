package exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStatsReaderNil(t *testing.T) {
	a := assert.New(t)
	unknownUris := []string{
		"abc://xxx",
		"def://yyy",
		"socks://vvvv",
	}

	for _, uri := range unknownUris {
		reader, err := NewStatsReader(uri, someTimeout)
		if a.Error(err) {
			a.Nil(reader)
		}
	}
}
