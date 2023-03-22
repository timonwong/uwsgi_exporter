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
	registerStatsReaderFunc("unix", newUnixStatsReader)
}

func newUnixStatsReader(u *url.URL, timeout time.Duration) StatsReader {
	return &unixStatsReader{
		filename: u.Path,
		timeout:  timeout,
	}
}

func (r *unixStatsReader) Read() (*UwsgiStats, error) {
	conn, err := net.Dial("unix", r.filename)
	if err != nil {
		return nil, fmt.Errorf("error reading stats from unix socket %s: %w", r.filename, err)
	}
	defer conn.Close()

	err = setDeadLine(r.timeout, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	uwsgiStats, err := parseUwsgiStatsFromIO(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return uwsgiStats, nil
}
