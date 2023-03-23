package collector

import (
	"context"
	"fmt"
	"net/url"
)

type tcpStatsReader struct {
	host string
}

func init() {
	registerStatsReaderFunc("tcp", newTCPStatsReader)
}

func newTCPStatsReader(u *url.URL) StatsReader {
	return &tcpStatsReader{
		host: u.Host,
	}
}

func (r *tcpStatsReader) Read(ctx context.Context) (*UwsgiStats, error) {
	d := newDialer()
	conn, err := d.DialContext(ctx, "tcp", r.host)
	if err != nil {
		return nil, fmt.Errorf("error reading stats from tcp: %w", err)
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
