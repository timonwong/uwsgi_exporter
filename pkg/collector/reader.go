package collector

import (
	"context"
	"fmt"
	"net"
	"net/url"
)

// StatsReader reads uwsgi stats from specified uri.
type StatsReader interface {
	Read(ctx context.Context) (*UwsgiStats, error)
}

// StatsReaderFunc is prototype for new stats reader
type StatsReaderFunc func(u *url.URL) StatsReader

var statsReaderFuncRegistry = make(map[string]StatsReaderFunc)

func registerStatsReaderFunc(scheme string, creator StatsReaderFunc) {
	statsReaderFuncRegistry[scheme] = creator
}

// NewStatsReader creates a StatsReader according to uri.
func NewStatsReader(uri string) (StatsReader, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uri: %w", err)
	}

	fn := statsReaderFuncRegistry[u.Scheme]
	if fn == nil {
		return nil, fmt.Errorf("incompatible uri %s", uri)
	}

	return fn(u), nil
}

func setDeadLine(ctx context.Context, conn net.Conn) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil
	}

	return conn.SetDeadline(deadline)
}
