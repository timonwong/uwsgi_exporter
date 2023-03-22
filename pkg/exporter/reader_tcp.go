package exporter

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

type tcpStatsReader struct {
	host    string
	timeout time.Duration
}

func init() {
	registerStatsReaderFunc("tcp", newTCPStatsReader)
}

func newTCPStatsReader(u *url.URL, timeout time.Duration) StatsReader {
	return &tcpStatsReader{
		host:    u.Host,
		timeout: timeout,
	}
}

func (r *tcpStatsReader) Read() (*UwsgiStats, error) {
	var (
		err  error
		conn net.Conn
	)

	if r.timeout == 0 {
		conn, err = net.Dial("tcp", r.host)
	} else {
		conn, err = net.DialTimeout("tcp", r.host, r.timeout)
	}
	if err != nil {
		return nil, fmt.Errorf("error reading stats from tcp: %w", err)
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
