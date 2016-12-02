package exporter

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/common/log"
)

// StatsReader reads uwsgi stats from specified uri
type StatsReader interface {
	Read() ([]byte, error)
}

// NewStatsReader creates a StatsReader according to uri.
func NewStatsReader(uri string, timeout time.Duration) (StatsReader, error) {
	u, err := url.Parse(uri)
	if err != nil {
		log.Errorf("Failed to parse uri %s: %s", uri, err)
		return nil, err
	}

	switch u.Scheme {
	case "http":
	case "https":
		reader := &HTTPStatsReader{
			url: uri,
			client: &http.Client{
				Transport: &http.Transport{
					Dial: func(network, addr string) (net.Conn, error) {
						c, err := net.DialTimeout(network, addr, timeout)
						if err != nil {
							return nil, err
						}
						if err := c.SetDeadline(time.Now().Add(timeout)); err != nil {
							return nil, err
						}
						return c, nil
					},
				},
			},
		}
		return reader, nil
	}

	err = fmt.Errorf("Unknown scheme %s", u.Scheme)
	return nil, err
}
