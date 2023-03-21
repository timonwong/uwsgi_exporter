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

// StatsReaderCreator is prototype for new stats reader
type StatsReaderCreator func(u *url.URL, uri string, timeout time.Duration) StatsReader

// statsReaderCreators is a response chain for stats reader creators.
var statsReaderCreators []StatsReaderCreator

// NewStatsReader creates a StatsReader according to uri.
func NewStatsReader(uri string, timeout time.Duration) (StatsReader, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uri: %w", err)
	}

	for _, statsReaderCreator := range statsReaderCreators {
		reader := statsReaderCreator(u, uri, timeout)
		if reader != nil {
			return reader, nil
		}
	}

	return nil, fmt.Errorf("incompatible uri %s", uri)
}

func setDeadLine(timeout time.Duration, conn net.Conn) error {
	if timeout == 0 {
		return nil
	}

	return conn.SetDeadline(time.Now().Add(timeout))
}
