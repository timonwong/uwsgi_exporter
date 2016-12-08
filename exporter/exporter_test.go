package exporter

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	sampleUwsgiStatsFileName string
	sampleUwsgiStatsJSON     []byte
)

func init() {
	var err error

	if sampleUwsgiStatsFileName, err = filepath.Abs("../testdata/sample.json"); err != nil {
		panic(err)
	}

	if sampleUwsgiStatsJSON, err = ioutil.ReadFile("../testdata/sample.json"); err != nil {
		panic(err)
	}
}

func newUwsgiStatsServer(response []byte) *httptest.Server {
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Type", "application/json")
		w.Write(response)
	})
	s := httptest.NewServer(handlerFunc)
	return s
}

func TestNewExporter(t *testing.T) {
	s := newUwsgiStatsServer(sampleUwsgiStatsJSON)

	exporter := NewExporter(s.URL, someTimeout, false)
	ch := make(chan prometheus.Metric)

	go func() {
		defer close(ch)
		defer s.Close()
		exporter.Collect(ch)
	}()
}
