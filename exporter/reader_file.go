package exporter

import (
	"net/url"
	"os"
	"time"

	"github.com/prometheus/common/log"
)

type fileStatsReader struct {
	filename string
}

func init() {
	StatsReaderCreators = append(StatsReaderCreators, newFileStatsReader)
}

func newFileStatsReader(u *url.URL, uri string, timeout time.Duration) StatsReader {
	if u.Scheme != "file" {
		return nil
	}

	return &fileStatsReader{
		filename: u.Path,
	}
}

func (reader *fileStatsReader) Read() (*UwsgiStats, error) {
	f, err := os.Open(reader.filename)
	if err != nil {
		log.Errorf("Cannot open file %s: %s", reader.filename, err)
		return nil, err
	}
	defer f.Close()

	uwsgiStats, err := parseUwsgiStatsFromIO(f)
	if err != nil {
		log.Errorf("Failed to unmarshal JSON: %s", err)
		return nil, err
	}

	return uwsgiStats, nil
}
