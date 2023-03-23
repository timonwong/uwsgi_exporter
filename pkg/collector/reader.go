package collector

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
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

type statsReaderOptions struct {
	requireSafeScheme bool
}

type StatsReaderOption func(*statsReaderOptions)

func WithRequireSafeScheme(safe bool) StatsReaderOption {
	return func(opts *statsReaderOptions) {
		opts.requireSafeScheme = safe
	}
}

// NewStatsReader creates a StatsReader according to uri.
func NewStatsReader(uri string, opts ...StatsReaderOption) (StatsReader, error) {
	options := &statsReaderOptions{}
	for _, opt := range opts {
		opt(options)
	}

	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uri: %w", err)
	}

	if !strings.Contains(uri, "://") && u.Host == "" {
		// Assume it's a http uri and parse again
		u, err = url.Parse("http://" + uri)
		if err != nil {
			return nil, fmt.Errorf("failed to parse uri (again): %w", err)
		}
	}

	// If in safe mode, we only accept tcp, http, and https
	if options.requireSafeScheme {
		switch u.Scheme {
		case "tcp", "http", "https":
		default:
			return nil, fmt.Errorf("unsafe scheme: %s", u.Scheme)
		}
	}

	fn := statsReaderFuncRegistry[u.Scheme]
	if fn == nil {
		return nil, fmt.Errorf("incompatible scheme: %s", u.Scheme)
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
