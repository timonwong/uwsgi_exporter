package collector

import (
	"context"
	_ "embed"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

var (
	sampleUwsgiStatsFileName string
	//go:embed testdata/sample.json
	sampleUwsgiStatsJSON []byte
	//go:embed testdata/wrong.json
	wrongUwsgiStatsJSON []byte
)

func init() {
	sampleUwsgiStatsFileName = lo.Must(filepath.Abs("testdata/sample.json"))
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

type MetricResult struct {
	labels     prometheus.Labels
	value      float64
	metricType dto.MetricType
}

func readMetric(m prometheus.Metric) MetricResult {
	pb := &dto.Metric{}
	m.Write(pb)
	labels := make(prometheus.Labels, len(pb.Label))
	for _, v := range pb.Label {
		labels[v.GetName()] = v.GetValue()
	}
	if pb.Gauge != nil {
		return MetricResult{labels: labels, value: pb.GetGauge().GetValue(), metricType: dto.MetricType_GAUGE}
	}
	if pb.Counter != nil {
		return MetricResult{labels: labels, value: pb.GetCounter().GetValue(), metricType: dto.MetricType_COUNTER}
	}
	if pb.Summary != nil {
		return MetricResult{labels: labels, value: 0, metricType: dto.MetricType_SUMMARY}
	}

	panic("Unsupported metric type")
}

func TestUwsgiExporter_CollectWrongJSON(t *testing.T) {
	s := newUwsgiStatsServer(wrongUwsgiStatsJSON)
	logger := log.NewLogfmtLogger(os.Stderr)
	ctx, cancel := context.WithTimeout(context.Background(), someTimeout)
	defer cancel()
	exporter := New(ctx, s.URL, NewMetrics(), ExporterOptions{
		Logger:            logger,
		CollectCores:      false,
		RequireSafeScheme: false,
	})

	ch := make(chan prometheus.Metric)

	go func() {
		defer close(ch)
		defer s.Close()
		exporter.Collect(ch)
	}()

	// total_scrapes
	expected := MetricResult{labels: prometheus.Labels{}, value: 1, metricType: dto.MetricType_COUNTER}
	got := readMetric(<-ch)
	assert.Equal(t, expected, got)

	// error
	expected = MetricResult{labels: prometheus.Labels{}, value: 1, metricType: dto.MetricType_GAUGE}
	got = readMetric(<-ch)
	assert.Equal(t, expected, got)

	// scrape duration
	expected = MetricResult{labels: prometheus.Labels{}, value: 0, metricType: dto.MetricType_SUMMARY}
	got = readMetric(<-ch)
	assert.Equal(t, expected, got)

	// scrape_errors
	expected = MetricResult{labels: prometheus.Labels{}, value: 1, metricType: dto.MetricType_COUNTER}
	got = readMetric(<-ch)
	assert.Equal(t, expected, got)

	// uwsgi_up
	expected = MetricResult{labels: prometheus.Labels{}, value: 0, metricType: dto.MetricType_GAUGE}
	got = readMetric(<-ch)
	assert.Equal(t, expected, got)
}

func TestUwsgiExporter_Collect(t *testing.T) {
	s := newUwsgiStatsServer(sampleUwsgiStatsJSON)
	logger := log.NewLogfmtLogger(os.Stderr)
	ctx, cancel := context.WithTimeout(context.Background(), someTimeout)
	defer cancel()
	exporter := New(ctx, s.URL, NewMetrics(), ExporterOptions{
		Logger:            logger,
		CollectCores:      true,
		RequireSafeScheme: false,
	})

	ch := make(chan prometheus.Metric)

	go func() {
		defer close(ch)
		defer s.Close()
		exporter.Collect(ch)
	}()

	// main
	labels := prometheus.Labels{"stats_uri": s.URL}
	mainMetricResults := []MetricResult{
		// listen_queue_length
		{labels: labels, value: 0, metricType: dto.MetricType_GAUGE},
		// listen_queue_errors
		{labels: labels, value: 0, metricType: dto.MetricType_GAUGE},
		// signal_queue_length
		{labels: labels, value: 0, metricType: dto.MetricType_GAUGE},
		// workers
		{labels: labels, value: 2, metricType: dto.MetricType_GAUGE},
	}
	for id, expect := range mainMetricResults {
		got := readMetric(<-ch)
		assert.Equal(t, expect, got, "Wrong main stats at id: %d", id)
	}

	// sockets
	labels = prometheus.Labels{"name": "127.0.0.1:36577", "proto": "uwsgi", "stats_uri": s.URL}
	socketMetricResults := []MetricResult{
		{labels: labels, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: labels, value: 100, metricType: dto.MetricType_GAUGE},
		{labels: labels, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: labels, value: 0, metricType: dto.MetricType_GAUGE},
	}
	for _, expect := range socketMetricResults {
		got := readMetric(<-ch)
		assert.Equal(t, expect, got, "Wrong socket stats")
	}

	// worker
	workerLabels := prometheus.Labels{"stats_uri": s.URL, "worker_id": "1"}
	workerMetricResults := []MetricResult{
		{labels: workerLabels, value: 1, metricType: dto.MetricType_GAUGE},
		{labels: workerLabels, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: workerLabels, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: workerLabels, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: workerLabels, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: workerLabels, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: workerLabels, value: 1457410597, metricType: dto.MetricType_GAUGE}, // last_spawn
		{labels: workerLabels, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: workerLabels, value: 0, metricType: dto.MetricType_GAUGE}, // busy
		{labels: workerLabels, value: 1, metricType: dto.MetricType_GAUGE}, // idle
		{labels: workerLabels, value: 0, metricType: dto.MetricType_GAUGE}, // cheap
		{labels: workerLabels, value: 0, metricType: dto.MetricType_COUNTER},
		{labels: workerLabels, value: 0, metricType: dto.MetricType_COUNTER},
		{labels: workerLabels, value: 0, metricType: dto.MetricType_COUNTER},
		{labels: workerLabels, value: 0, metricType: dto.MetricType_COUNTER},
		{labels: workerLabels, value: 1, metricType: dto.MetricType_COUNTER},
		{labels: workerLabels, value: 0, metricType: dto.MetricType_COUNTER},
	}
	for idx, expect := range workerMetricResults {
		got := readMetric(<-ch)
		assert.Equal(t, expect, got, "Wrong worker stats at idx %d", idx)
	}

	// worker apps
	labels = prometheus.Labels{"stats_uri": s.URL, "worker_id": "1", "chdir": "", "mountpoint": "", "app_id": "0"}
	workerAppMetricResults := []MetricResult{
		{labels: workerLabels, value: 1, metricType: dto.MetricType_GAUGE}, // app count

		{labels: labels, value: 0, metricType: dto.MetricType_GAUGE},

		{labels: labels, value: 0, metricType: dto.MetricType_COUNTER},
		{labels: labels, value: 0, metricType: dto.MetricType_COUNTER},
	}
	for _, expect := range workerAppMetricResults {
		got := readMetric(<-ch)
		assert.Equal(t, expect, got, "Wrong worker app stats")
	}

	// worker cores
	labels = prometheus.Labels{"stats_uri": s.URL, "worker_id": "1", "core_id": "0"}
	workerCoreMetricResults := []MetricResult{
		{labels: workerLabels, value: 1, metricType: dto.MetricType_GAUGE}, // core count

		{labels: labels, value: 0, metricType: dto.MetricType_GAUGE}, // in_requests

		{labels: labels, value: 0, metricType: dto.MetricType_COUNTER}, // requests_total
		{labels: labels, value: 0, metricType: dto.MetricType_COUNTER},
		{labels: labels, value: 0, metricType: dto.MetricType_COUNTER},
		{labels: labels, value: 0, metricType: dto.MetricType_COUNTER},
		{labels: labels, value: 0, metricType: dto.MetricType_COUNTER},
		{labels: labels, value: 0, metricType: dto.MetricType_COUNTER},
	}
	for _, expect := range workerCoreMetricResults {
		got := readMetric(<-ch)
		assert.Equal(t, expect, got, "Wrong worker core stats: %d")
	}

	// Drain
	for i := 0; i < len(workerMetricResults)+len(workerAppMetricResults)+len(workerCoreMetricResults); i++ {
		readMetric(<-ch)
	}

	// caches
	labels = prometheus.Labels{"stats_uri": s.URL, "name": "cache_1"}
	cacheMetricResults := []MetricResult{
		{labels: labels, value: 56614, metricType: dto.MetricType_COUNTER},   // hits
		{labels: labels, value: 4931570, metricType: dto.MetricType_COUNTER}, // misses
		{labels: labels, value: 0, metricType: dto.MetricType_COUNTER},       // full

		{labels: labels, value: 39, metricType: dto.MetricType_GAUGE},   // items
		{labels: labels, value: 2000, metricType: dto.MetricType_GAUGE}, // max_items
	}
	for _, expect := range cacheMetricResults {
		got := readMetric(<-ch)
		assert.Equal(t, expect, got, "Wrong cache stats: %d")
	}

	// Drain
	for i := 0; i < len(cacheMetricResults); i++ {
		readMetric(<-ch)
	}

	// total_scrapes
	expected := MetricResult{labels: prometheus.Labels{}, value: 1, metricType: dto.MetricType_COUNTER}
	got := readMetric(<-ch)
	assert.Equal(t, expected, got)

	// error
	expected = MetricResult{labels: prometheus.Labels{}, value: 0, metricType: dto.MetricType_GAUGE}
	got = readMetric(<-ch)
	assert.Equal(t, expected, got)

	// scrape duration
	expected = MetricResult{labels: prometheus.Labels{}, value: 0, metricType: dto.MetricType_SUMMARY}
	got = readMetric(<-ch)
	assert.Equal(t, expected, got)

	// scrape_errors
	expected = MetricResult{labels: prometheus.Labels{}, value: 0, metricType: dto.MetricType_COUNTER}
	got = readMetric(<-ch)
	assert.Equal(t, expected, got)

	// uwsgi_up
	expected = MetricResult{labels: prometheus.Labels{}, value: 1, metricType: dto.MetricType_GAUGE}
	got = readMetric(<-ch)
	assert.Equal(t, expected, got)
}
