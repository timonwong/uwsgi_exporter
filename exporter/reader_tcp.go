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
	StatsReaderCreators = append(StatsReaderCreators, newTCPStatsReader)
}

func newTCPStatsReader(u *url.URL, uri string, timeout time.Duration) StatsReader {
	if u.Scheme != "tcp" {
		return nil
	}

	return &tcpStatsReader{
		host:    u.Host,
		timeout: timeout,
	}
}

func (reader *tcpStatsReader) Read() (*UwsgiStats, error) {
	conn, err := net.Dial("tcp", reader.host)
	if err != nil {
		return nil, fmt.Errorf("error reading stats from tcp: %w", err)
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
