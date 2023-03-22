package collector

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type httpStatsReader struct {
	uri string

	client *http.Client
}

func init() {
	registerStatsReaderFunc("http", newHTTPStatsReader)
	registerStatsReaderFunc("https", newHTTPStatsReader)
}

func newHTTPStatsReader(u *url.URL) StatsReader {
	return &httpStatsReader{
		uri:    u.String(),
		client: &http.Client{},
	}
}

func (r *httpStatsReader) Read(ctx context.Context) (*UwsgiStats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.uri, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error querying uwsgi stats: %w", err)
	}
	defer resp.Body.Close()

	uwsgiStats, err := parseUwsgiStatsFromIO(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uwsgi stats: %w", err)
	}
	return uwsgiStats, nil
}
