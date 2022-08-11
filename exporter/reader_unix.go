package exporter

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

type unixStatsReader struct {
	filename string
	timeout  time.Duration
}

func init() {
	StatsReaderCreators = append(StatsReaderCreators, newUnixStatsReader)
}

func newUnixStatsReader(u *url.URL, uri string, timeout time.Duration) StatsReader {
	if u.Scheme != "unix" {
		return nil
	}

	return &unixStatsReader{
		filename: u.Path,
		timeout:  timeout,
	}
}

func (reader *unixStatsReader) Read() (*UwsgiStats, error) {
	conn, err := net.Dial("unix", reader.filename)
	if err != nil {
		return nil, fmt.Errorf("error reading stats from unix socket %s: %w", reader.filename, err)
	}
	defer conn.Close()

	err = conn.SetDeadline(time.Now().Add(reader.timeout))
	if err != nil {
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	uwsgiStats, err := parseUwsgiStatsFromIO(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return uwsgiStats, nil
}
