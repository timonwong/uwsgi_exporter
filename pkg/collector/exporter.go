package collector

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace           = "uwsgi"
	exporter            = "exporter"
	mainSubsystem       = ""
	socketSubsystem     = "socket"
	workerSubsystem     = "worker"
	workerAppSubsystem  = "worker_app"
	workerCoreSubsystem = "worker_core"
	cacheSubsystem      = "cache"

	usDivider = float64(time.Second / time.Microsecond)
)

// Descriptors is a map for `prometheus.Desc` pointer.
type Descriptors map[string]*prometheus.Desc

// DescriptorsMap is a map for `Descriptors`.
type DescriptorsMap map[string]Descriptors

var (
	metricsMap = map[string]map[string]string{
		mainSubsystem: {
			"listen_queue_length": "Length of listen queue.",
			"listen_queue_errors": "Number of listen queue errors.",
			"signal_queue_length": "Length of signal queue.",
			"workers":             "Number of workers.",
		},

		socketSubsystem: {
			"queue_length":     "Length of socket queue.",
			"max_queue_length": "Max length of socket queue.",
			"shared":           "Is shared socket?",
			"can_offload":      "Can socket offload?",
		},

		workerSubsystem: {
			"accepting":                     "Is this worker accepting requests?",
			"delta_requests":                "Number of delta requests",
			"signal_queue_length":           "Length of signal queue.",
			"rss_bytes":                     "Worker RSS bytes.",
			"vsz_bytes":                     "Worker VSZ bytes.",
			"running_time_seconds":          "Worker running time in seconds.",
			"last_spawn_time_seconds":       "Last spawn time in seconds since epoch.",
			"average_response_time_seconds": "Average response time in seconds.",
			"apps":                          "Number of apps.",
			"cores":                         "Number of cores.",

			"requests_total":          "Total number of requests.",
			"exceptions_total":        "Total number of exceptions.",
			"harakiri_count_total":    "Total number of harakiri count.",
			"signals_total":           "Total number of signals.",
			"respawn_count_total":     "Total number of respawn count.",
			"transmitted_bytes_total": "Worker transmitted bytes.",

			// worker statuses (gauges)
			"busy":  "Is core in busy",
			"idle":  "Is core in idle",
			"cheap": "Is core in cheap mode",
		},

		workerAppSubsystem: {
			"startup_time_seconds": "How long this app took to start.",

			"requests_total":   "Total number of requests.",
			"exceptions_total": "Total number of exceptions.",
		},

		workerCoreSubsystem: {
			"busy": "Is core busy",

			"requests_total":           "Total number of requests.",
			"static_requests_total":    "Total number of static requests.",
			"routed_requests_total":    "Total number of routed requests.",
			"offloaded_requests_total": "Total number of offloaded requests.",
			"write_errors_total":       "Total number of write errors.",
			"read_errors_total":        "Total number of read errors.",
		},

		cacheSubsystem: {
			"hits":      "Total number of hits.",
			"misses":    "Total number of misses.",
			"full":      "Total Number of times cache full was hit.",
			"items":     "Items in cache.",
			"max_items": "Max items for this cache.",
		},
	}

	// Please note that stats_uri is omitted here, because it's const
	labelsMap = map[string][]string{
		mainSubsystem:       {},
		socketSubsystem:     {"name", "proto"},
		workerSubsystem:     {"worker_id"},
		workerAppSubsystem:  {"worker_id", "app_id", "mountpoint", "chdir"},
		workerCoreSubsystem: {"worker_id", "core_id"},
		cacheSubsystem:      {"name"},
	}
)

// Exporter collects uwsgi metrics for prometheus.
type Exporter struct {
	ctx            context.Context
	logger         log.Logger
	uri            string
	collectCores   bool
	descriptorsMap DescriptorsMap
	metrics        Metrics
}

// New creates a new uwsgi collector.
func New(ctx context.Context, uri string, metrics Metrics, collectCores bool, logger log.Logger) *Exporter {
	descriptorsMap := make(DescriptorsMap, len(metricsMap))
	constLabels := prometheus.Labels{"stats_uri": uri}

	for subsystem, metrics := range metricsMap {
		descriptors := make(Descriptors, len(metrics))
		for name, help := range metrics {
			descriptors[name] = prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, name), help, labelsMap[subsystem], constLabels)
		}

		descriptorsMap[subsystem] = descriptors
	}

	return &Exporter{
		ctx:            ctx,
		logger:         logger,
		uri:            uri,
		collectCores:   collectCores,
		descriptorsMap: descriptorsMap,
		metrics:        metrics,
	}
}

// Describe describes all the metrics ever exported by the exporter.
// It implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.metrics.TotalScrapes.Desc()
	ch <- e.metrics.Error.Desc()
	ch <- e.metrics.ScrapeDurations.Desc()
	ch <- e.metrics.ScrapeErrors.Desc()
	ch <- e.metrics.Up.Desc()

	for _, descs := range e.descriptorsMap {
		for _, desc := range descs {
			ch <- desc
		}
	}
}

// Collect fetches the stats from configured uwsgi stats location and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(e.ctx, ch)

	ch <- e.metrics.TotalScrapes
	ch <- e.metrics.Error
	ch <- e.metrics.ScrapeDurations
	ch <- e.metrics.ScrapeErrors
	ch <- e.metrics.Up
}

// readUwsgiStats reads (and parse) stats from uwsgi server
func (e *Exporter) readUwsgiStats(ctx context.Context) (*UwsgiStats, error) {
	statsReader, err := NewStatsReader(e.uri)
	if err != nil {
		return nil, fmt.Errorf("failed to create stats reader: %w", err)
	}

	return statsReader.Read(ctx)
}

func (e *Exporter) scrape(ctx context.Context, ch chan<- prometheus.Metric) {
	e.metrics.TotalScrapes.Inc()

	scrapeTime := time.Now()

	e.metrics.Up.Set(1)
	e.metrics.Error.Set(0)

	uwsgiStats, err := e.readUwsgiStats(ctx)
	if err != nil {
		level.Error(e.logger).Log("msg", "Scrape failed", "error", err)

		e.metrics.ScrapeErrors.Inc()
		e.metrics.Up.Set(0)
		e.metrics.Error.Set(1)
		return
	}

	level.Debug(e.logger).Log("msg", "Scrape successful")
	e.metrics.ScrapeDurations.Observe(time.Since(scrapeTime).Seconds())

	// Collect metrics from stats
	e.collectMetrics(uwsgiStats, ch)
}

func newGaugeMetric(desc *prometheus.Desc, value float64, labelValues ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, labelValues...)
}

func newCounterMetric(desc *prometheus.Desc, value float64, labelsValues ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(desc, prometheus.CounterValue, value, labelsValues...)
}

var availableWorkerStatuses = []string{"busy", "idle", "cheap"}

func (e *Exporter) collectMetrics(stats *UwsgiStats, ch chan<- prometheus.Metric) {
	// Main
	mainDescs := e.descriptorsMap[mainSubsystem]

	availableWorkers := make([]UwsgiWorker, 0, len(stats.Workers))
	// Filter workers (filter out "stand-by" workers, in uwsgi's adaptive process respawn mode)
	for _, workerStats := range stats.Workers {
		if workerStats.ID == 0 {
			continue
		}

		availableWorkers = append(availableWorkers, workerStats)
	}

	ch <- newGaugeMetric(mainDescs["listen_queue_length"], float64(stats.ListenQueue))
	ch <- newGaugeMetric(mainDescs["listen_queue_errors"], float64(stats.ListenQueueErrors))
	ch <- newGaugeMetric(mainDescs["signal_queue_length"], float64(stats.SignalQueue))
	ch <- newGaugeMetric(mainDescs["workers"], float64(len(availableWorkers)))

	// Sockets
	// NOTE(timonwong): Workaround bug #22
	type socketStatKey struct {
		Name  string
		Proto string
	}
	sockets := make(map[socketStatKey]UwsgiSocket, len(stats.Sockets))
	for _, socket := range stats.Sockets {
		key := socketStatKey{Name: socket.Name, Proto: socket.Proto}
		if _, ok := sockets[key]; !ok {
			// First one with the same key take precedence.
			sockets[key] = socket
		}
	}

	socketDescs := e.descriptorsMap[socketSubsystem]
	for key, socket := range sockets {
		labelValues := []string{key.Name, key.Proto}

		ch <- newGaugeMetric(socketDescs["queue_length"], float64(socket.Queue), labelValues...)
		ch <- newGaugeMetric(socketDescs["max_queue_length"], float64(socket.MaxQueue), labelValues...)
		ch <- newGaugeMetric(socketDescs["shared"], float64(socket.Shared), labelValues...)
		ch <- newGaugeMetric(socketDescs["can_offload"], float64(socket.CanOffload), labelValues...)
	}

	// Workers
	workerDescs := e.descriptorsMap[workerSubsystem]
	workerAppDescs := e.descriptorsMap[workerAppSubsystem]
	workerCoreDescs := e.descriptorsMap[workerCoreSubsystem]
	cacheDescs := e.descriptorsMap[cacheSubsystem]

	for _, workerStats := range availableWorkers {
		labelValues := []string{strconv.Itoa(workerStats.ID)}

		ch <- newGaugeMetric(workerDescs["accepting"], float64(workerStats.Accepting), labelValues...)
		ch <- newGaugeMetric(workerDescs["delta_requests"], float64(workerStats.DeltaRequests), labelValues...)
		ch <- newGaugeMetric(workerDescs["signal_queue_length"], float64(workerStats.SignalQueue), labelValues...)
		ch <- newGaugeMetric(workerDescs["rss_bytes"], float64(workerStats.RSS), labelValues...)
		ch <- newGaugeMetric(workerDescs["vsz_bytes"], float64(workerStats.VSZ), labelValues...)
		ch <- newGaugeMetric(workerDescs["running_time_seconds"], float64(workerStats.RunningTime)/usDivider, labelValues...)
		ch <- newGaugeMetric(workerDescs["last_spawn_time_seconds"], float64(workerStats.LastSpawn), labelValues...)
		ch <- newGaugeMetric(workerDescs["average_response_time_seconds"], float64(workerStats.AvgRt)/usDivider, labelValues...)

		for _, st := range availableWorkerStatuses {
			v := float64(0)
			if workerStats.Status == st {
				v = float64(1.0)
			}
			ch <- newGaugeMetric(workerDescs[st], v, labelValues...)
		}

		ch <- newCounterMetric(workerDescs["requests_total"], float64(workerStats.Requests), labelValues...)
		ch <- newCounterMetric(workerDescs["exceptions_total"], float64(workerStats.Exceptions), labelValues...)
		ch <- newCounterMetric(workerDescs["harakiri_count_total"], float64(workerStats.HarakiriCount), labelValues...)
		ch <- newCounterMetric(workerDescs["signals_total"], float64(workerStats.Signals), labelValues...)
		ch <- newCounterMetric(workerDescs["respawn_count_total"], float64(workerStats.RespawnCount), labelValues...)
		ch <- newCounterMetric(workerDescs["transmitted_bytes_total"], float64(workerStats.TX), labelValues...)

		// Worker Apps
		ch <- newGaugeMetric(workerDescs["apps"], float64(len(workerStats.Apps)), labelValues...)
		for _, appStats := range workerStats.Apps {
			labelValues := []string{strconv.Itoa(workerStats.ID), strconv.Itoa(appStats.ID), appStats.Mountpoint, appStats.Chdir}
			ch <- newGaugeMetric(workerAppDescs["startup_time_seconds"], float64(appStats.StartupTime), labelValues...)

			ch <- newCounterMetric(workerAppDescs["requests_total"], float64(appStats.Requests), labelValues...)
			ch <- newCounterMetric(workerAppDescs["exceptions_total"], float64(appStats.Exceptions), labelValues...)
		}

		// Worker Cores
		ch <- newGaugeMetric(workerDescs["cores"], float64(len(workerStats.Cores)), labelValues...)
		if e.collectCores {
			for _, coreStats := range workerStats.Cores {
				labelValues := []string{strconv.Itoa(workerStats.ID), strconv.Itoa(coreStats.ID)}
				ch <- newGaugeMetric(workerCoreDescs["busy"], float64(coreStats.InRequest), labelValues...)

				ch <- newCounterMetric(workerCoreDescs["requests_total"], float64(coreStats.Requests), labelValues...)
				ch <- newCounterMetric(workerCoreDescs["static_requests_total"], float64(coreStats.StaticRequests), labelValues...)
				ch <- newCounterMetric(workerCoreDescs["routed_requests_total"], float64(coreStats.RoutedRequests), labelValues...)
				ch <- newCounterMetric(workerCoreDescs["offloaded_requests_total"], float64(coreStats.OffloadedRequests), labelValues...)
				ch <- newCounterMetric(workerCoreDescs["write_errors_total"], float64(coreStats.WriteErrors), labelValues...)
				ch <- newCounterMetric(workerCoreDescs["read_errors_total"], float64(coreStats.ReadErrors), labelValues...)
			}
		}
	}

	for _, cacheStats := range stats.Caches {
		labelValues := []string{cacheStats.Name}
		ch <- newCounterMetric(cacheDescs["hits"], float64(cacheStats.Hits), labelValues...)
		ch <- newCounterMetric(cacheDescs["misses"], float64(cacheStats.Misses), labelValues...)
		ch <- newCounterMetric(cacheDescs["full"], float64(cacheStats.Full), labelValues...)

		ch <- newGaugeMetric(cacheDescs["items"], float64(cacheStats.Items), labelValues...)
		ch <- newGaugeMetric(cacheDescs["max_items"], float64(cacheStats.MaxItems), labelValues...)
	}
}

// Metrics represents exporter metrics which values can be carried between http requests.
type Metrics struct {
	TotalScrapes    prometheus.Counter
	ScrapeErrors    prometheus.Counter
	ScrapeDurations prometheus.Summary
	Error           prometheus.Gauge
	Up              prometheus.Gauge
}

// NewMetrics creates new Metrics instance.
func NewMetrics() Metrics {
	subsystem := exporter
	return Metrics{
		TotalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scrapes_total",
			Help:      "Total number of times uWSGI was scraped for metrics.",
		}),
		ScrapeErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scrape_errors_total",
			Help:      "Total number of times an error occurred scraping a uWSGI.",
		}),
		ScrapeDurations: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scrape_duration_seconds",
			Help:      "Duration of uWSGI scrape job.",
		}),
		Error: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "last_scrape_error",
			Help:      "Whether the last scrape of metrics from uWSGI resulted in an error (1 for error, 0 for success).",
		}),
		Up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Whether the uWSGI server is up.",
		}),
	}
}