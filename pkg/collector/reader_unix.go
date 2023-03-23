package collector

import (
	"context"
	"fmt"
	"net/url"
)

type unixStatsReader struct {
	filename string
}

func init() {
	registerStatsReaderFunc("unix", newUnixStatsReader)
}

func newUnixStatsReader(u *url.URL) StatsReader {
	return &unixStatsReader{
		filename: u.Path,
	}
}

func (r *unixStatsReader) Read(ctx context.Context) (*UwsgiStats, error) {
	d := newDialer()
	conn, err := d.DialContext(ctx, "unix", r.filename)
	if err != nil {
		return nil, fmt.Errorf("error reading stats from unix socket %s: %w", r.filename, err)
	}
	defer conn.Close()

	err = setDeadLine(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	uwsgiStats, err := parseUwsgiStatsFromIO(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return uwsgiStats, nil
}
