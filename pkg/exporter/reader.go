package exporter

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

// StatsReader reads uwsgi stats from specified uri.
type StatsReader interface {
	Read() (*UwsgiStats, error)
}

// StatsReaderFunc is prototype for new stats reader
type StatsReaderFunc func(u *url.URL, timeout time.Duration) StatsReader

var statsReaderFuncRegistry = make(map[string]StatsReaderFunc)

func registerStatsReaderFunc(scheme string, creator StatsReaderFunc) {
	statsReaderFuncRegistry[scheme] = creator
}

// NewStatsReader creates a StatsReader according to uri.
func NewStatsReader(uri string, timeout time.Duration) (StatsReader, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uri: %w", err)
	}

	fn := statsReaderFuncRegistry[u.Scheme]
	if fn == nil {
		return nil, fmt.Errorf("incompatible uri %s", uri)
	}

	return fn(u, timeout), nil
}

func setDeadLine(timeout time.Duration, conn net.Conn) error {
	if timeout == 0 {
		return nil
	}

	return conn.SetDeadline(time.Now().Add(timeout))
}
