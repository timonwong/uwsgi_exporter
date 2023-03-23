package collector

import (
	"context"
	"fmt"
	"net/url"
	"os"
)

type fileStatsReader struct {
	filename string
}

func init() {
	registerStatsReaderFunc("file", newFileStatsReader)
}

func newFileStatsReader(u *url.URL) StatsReader {
	return &fileStatsReader{
		filename: u.Path,
	}
}

func (r *fileStatsReader) Read(_ context.Context) (*UwsgiStats, error) {
	f, err := os.Open(r.filename)
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
