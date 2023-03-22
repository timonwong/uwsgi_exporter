package exporter

import (
	"fmt"
	"net/url"
	"os"
	"time"
)

type fileStatsReader struct {
	filename string
}

func init() {
	registerStatsReaderFunc("file", newFileStatsReader)
}

func newFileStatsReader(u *url.URL, timeout time.Duration) StatsReader {
	return &fileStatsReader{
		filename: u.Path,
	}
}

func (reader *fileStatsReader) Read() (*UwsgiStats, error) {
	f, err := os.Open(reader.filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open file: %w", err)
	}
	defer f.Close()

	uwsgiStats, err := parseUwsgiStatsFromIO(f)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON: %w", err)
	}

	return uwsgiStats, nil
}
