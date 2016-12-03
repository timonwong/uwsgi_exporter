package exporter

import (
	"io/ioutil"
	"net/http"

	"github.com/prometheus/common/log"
)

// HTTPStatsReader reads uwsgi stats through HTTP(s).
type HTTPStatsReader struct {
	uri string

	client *http.Client
}

func (reader *HTTPStatsReader) Read() ([]byte, error) {
	resp, err := reader.client.Get(reader.uri)
	if err != nil {
		log.Errorf("Error while querying uwsgi stats: %s", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Failed to read stats body: %s", err)
		return nil, err
	}

	return body, nil
}
