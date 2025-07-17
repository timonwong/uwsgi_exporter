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
		return nil, fmt.Errorf("error querying uwsgi stats from %s: %w", r.uri, err)
	}
	defer resp.Body.Close()

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uwsgi stats endpoint returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// 检查 Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		return nil, fmt.Errorf("unexpected content type: %s, expected application/json", contentType)
	}

	uwsgiStats, err := parseUwsgiStatsFromIO(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uwsgi stats from %s: %w", r.uri, err)
	}
	return uwsgiStats, nil
}
