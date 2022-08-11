package exporter

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

type httpStatsReader struct {
	uri string

	client *http.Client
}

func init() {
	StatsReaderCreators = append(StatsReaderCreators, newHTTPStatsReader)
}

func newHTTPStatsReader(u *url.URL, uri string, timeout time.Duration) StatsReader {
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil
	}

	return &httpStatsReader{
		uri: uri,
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
}

func (reader *httpStatsReader) Read() (*UwsgiStats, error) {
	resp, err := reader.client.Get(reader.uri)
	if err != nil {
		return nil, fmt.Errorf("error querying uwsgi stats: %w", err)
	}
	defer resp.Body.Close()

	uwsgiStats, err := parseUwsgiStatsFromIO(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return uwsgiStats, nil
}
