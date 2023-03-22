package exporter

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type httpStatsReader struct {
	uri string

	client *http.Client
}

func init() {
	registerStatsReaderFunc("http", newHTTPStatsReader)
	registerStatsReaderFunc("https", newHTTPStatsReader)
}

func newHTTPStatsReader(u *url.URL, timeout time.Duration) StatsReader {
	return &httpStatsReader{
		uri: u.String(),
		client: &http.Client{
			Timeout: timeout,
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
