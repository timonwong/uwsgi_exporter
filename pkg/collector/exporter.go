package collector

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "uwsgi"
	exporter  = "exporter"

	usDivider = float64(time.Second / time.Microsecond)
)

var subsystemDescriptors = SubsystemDescriptors{
	MainSubsystem: initDescriptors("", map[string]string{
		"listen_queue_length": "Length of listen queue.",
		"listen_queue_errors": "Number of listen queue errors.",
		"signal_queue_length": "Length of signal queue.",
		"workers":             "Number of workers.",
	}, nil),

	SocketSubsystem: initDescriptors("socket", map[string]string{
		"queue_length":     "Length of socket queue.",
		"max_queue_length": "Max length of socket queue.",
		"shared":           "Is shared socket?",
		"can_offload":      "Can socket offload?",
	}, []string{"name", "proto"}),

	WorkerSubsystem: initDescriptors("worker", map[string]string{
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
		"busy":  "Is core in busy?",
		"idle":  "Is core in idle?",
		"cheap": "Is core in cheap mode?",
	}, []string{"worker_id"}),

	WorkerAppSubsystem: initDescriptors("worker_app", map[string]string{
		"startup_time_seconds": "How long this app took to start.",

		"requests_total":   "Total number of requests.",
		"exceptions_total": "Total number of exceptions.",
	}, []string{"worker_id", "app_id", "mountpoint", "chdir"}),

	WorkerCoreSubsystem: initDescriptors("worker_core", map[string]string{
		"busy": "Is core busy",

		"requests_total":           "Total number of requests.",
		"static_requests_total":    "Total number of static requests.",
		"routed_requests_total":    "Total number of routed requests.",
		"offloaded_requests_total": "Total number of offloaded requests.",
		"write_errors_total":       "Total number of write errors.",
		"read_errors_total":        "Total number of read errors.",
	}, []string{"worker_id", "core_id"}),

	CacheSubsystem: initDescriptors("cache", map[string]string{
		"hits":      "Total number of hits.",
		"misses":    "Total number of misses.",
		"full":      "Total Number of times cache full was hit.",
		"items":     "Items in cache.",
		"max_items": "Max items for this cache.",
	}, []string{"name"}),
}

func initDescriptors(subsystem string, nameToHelp map[string]string, labels []string) Descriptors {
	descriptors := make(Descriptors, len(nameToHelp))
	for name, help := range nameToHelp {
		labels := append([]string{"stats_uri"}, labels...)
		descriptors[name] = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, name),
			help, labels, nil)
	}

	return descriptors
}

// Descriptors is a map for `prometheus.Desc` pointer.
type Descriptors map[string]*prometheus.Desc

type SubsystemDescriptors struct {
	MainSubsystem       Descriptors
	SocketSubsystem     Descriptors
	WorkerSubsystem     Descriptors
	WorkerAppSubsystem  Descriptors
	WorkerCoreSubsystem Descriptors
	CacheSubsystem      Descriptors
}

func (v *SubsystemDescriptors) Describe(ch chan<- *prometheus.Desc) {
	for _, descs := range []Descriptors{
		v.MainSubsystem,
		v.SocketSubsystem,
		v.WorkerSubsystem,
		v.WorkerAppSubsystem,
		v.WorkerCoreSubsystem,
		v.CacheSubsystem,
	} {
		for _, desc := range descs {
			ch <- desc
		}
	}
}

// Exporter collects uwsgi metrics for prometheus.
type Exporter struct {
	ctx         context.Context
	uri         string
	statsReader StatsReader
	metrics     Metrics
	ExporterOptions
}

type ExporterOptions struct {
	Logger       *slog.Logger
	CollectCores bool
}

// New creates a new uwsgi collector.
func New(ctx context.Context, uri string, statsReader StatsReader, metrics Metrics, options ExporterOptions) *Exporter {
	return &Exporter{
		ctx:             ctx,
		uri:             uri,
		statsReader:     statsReader,
		metrics:         metrics,
		ExporterOptions: options,
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

	subsystemDescriptors.Describe(ch)
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

func (e *Exporter) scrape(ctx context.Context, ch chan<- prometheus.Metric) {
	e.metrics.TotalScrapes.Inc()

	scrapeTime := time.Now()

	e.metrics.Up.Set(1)
	e.metrics.Error.Set(0)

	uwsgiStats, err := e.statsReader.Read(ctx)
	if err != nil {
		e.Logger.Error("Scrape failed", "error", err)

		e.metrics.ScrapeErrors.Inc()
		e.metrics.Up.Set(0)
		e.metrics.Error.Set(1)
		return
	}

	e.Logger.Debug("Scrape successful")
	e.metrics.ScrapeDurations.Observe(time.Since(scrapeTime).Seconds())

	// Collect metrics from stats
	e.collectMetrics(uwsgiStats, ch)
}

func (e *Exporter) mustNewGaugeMetric(desc *prometheus.Desc, value float64, labelValues ...string) prometheus.Metric {
	labelValues = append([]string{e.uri}, labelValues...)
	return prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, labelValues...)
}

func (e *Exporter) mustNewCounterMetric(desc *prometheus.Desc, value float64, labelValues ...string) prometheus.Metric {
	labelValues = append([]string{e.uri}, labelValues...)
	return prometheus.MustNewConstMetric(desc, prometheus.CounterValue, value, labelValues...)
}

var availableWorkerStatuses = []string{"busy", "idle", "cheap"}

func (e *Exporter) collectMetrics(stats *UwsgiStats, ch chan<- prometheus.Metric) {
	availableWorkers := make([]UwsgiWorker, 0, len(stats.Workers))
	// Filter workers (filter out "stand-by" workers, in uwsgi's adaptive process respawn mode)
	for _, workerStats := range stats.Workers {
		if workerStats.ID == 0 {
			continue
		}

		availableWorkers = append(availableWorkers, workerStats)
	}

	// Main
	mainDescs := subsystemDescriptors.MainSubsystem
	ch <- e.mustNewGaugeMetric(mainDescs["listen_queue_length"], float64(stats.ListenQueue))
	ch <- e.mustNewGaugeMetric(mainDescs["listen_queue_errors"], float64(stats.ListenQueueErrors))
	ch <- e.mustNewGaugeMetric(mainDescs["signal_queue_length"], float64(stats.SignalQueue))
	ch <- e.mustNewGaugeMetric(mainDescs["workers"], float64(len(availableWorkers)))

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

	socketDescs := subsystemDescriptors.SocketSubsystem
	for key, socket := range sockets {
		labelValues := []string{key.Name, key.Proto}

		ch <- e.mustNewGaugeMetric(socketDescs["queue_length"], float64(socket.Queue), labelValues...)
		ch <- e.mustNewGaugeMetric(socketDescs["max_queue_length"], float64(socket.MaxQueue), labelValues...)
		ch <- e.mustNewGaugeMetric(socketDescs["shared"], float64(socket.Shared), labelValues...)
		ch <- e.mustNewGaugeMetric(socketDescs["can_offload"], float64(socket.CanOffload), labelValues...)
	}

	// Workers
	workerDescs := subsystemDescriptors.WorkerSubsystem
	workerAppDescs := subsystemDescriptors.WorkerAppSubsystem
	workerCoreDescs := subsystemDescriptors.WorkerCoreSubsystem
	cacheDescs := subsystemDescriptors.CacheSubsystem

	for _, workerStats := range availableWorkers {
		labelValues := []string{strconv.Itoa(workerStats.ID)}

		ch <- e.mustNewGaugeMetric(workerDescs["accepting"], float64(workerStats.Accepting), labelValues...)
		ch <- e.mustNewGaugeMetric(workerDescs["delta_requests"], float64(workerStats.DeltaRequests), labelValues...)
		ch <- e.mustNewGaugeMetric(workerDescs["signal_queue_length"], float64(workerStats.SignalQueue), labelValues...)
		ch <- e.mustNewGaugeMetric(workerDescs["rss_bytes"], float64(workerStats.RSS), labelValues...)
		ch <- e.mustNewGaugeMetric(workerDescs["vsz_bytes"], float64(workerStats.VSZ), labelValues...)
		ch <- e.mustNewGaugeMetric(workerDescs["running_time_seconds"], float64(workerStats.RunningTime)/usDivider, labelValues...)
		ch <- e.mustNewGaugeMetric(workerDescs["last_spawn_time_seconds"], float64(workerStats.LastSpawn), labelValues...)
		ch <- e.mustNewGaugeMetric(workerDescs["average_response_time_seconds"], float64(workerStats.AvgRt)/usDivider, labelValues...)

		for _, st := range availableWorkerStatuses {
			v := float64(0)
			if workerStats.Status == st {
				v = 1.0
			}
			ch <- e.mustNewGaugeMetric(workerDescs[st], v, labelValues...)
		}

		ch <- e.mustNewCounterMetric(workerDescs["requests_total"], float64(workerStats.Requests), labelValues...)
		ch <- e.mustNewCounterMetric(workerDescs["exceptions_total"], float64(workerStats.Exceptions), labelValues...)
		ch <- e.mustNewCounterMetric(workerDescs["harakiri_count_total"], float64(workerStats.HarakiriCount), labelValues...)
		ch <- e.mustNewCounterMetric(workerDescs["signals_total"], float64(workerStats.Signals), labelValues...)
		ch <- e.mustNewCounterMetric(workerDescs["respawn_count_total"], float64(workerStats.RespawnCount), labelValues...)
		ch <- e.mustNewCounterMetric(workerDescs["transmitted_bytes_total"], float64(workerStats.TX), labelValues...)

		// Worker Apps
		ch <- e.mustNewGaugeMetric(workerDescs["apps"], float64(len(workerStats.Apps)), labelValues...)
		for _, appStats := range workerStats.Apps {
			labelValues := []string{strconv.Itoa(workerStats.ID), strconv.Itoa(appStats.ID), appStats.MountPoint, appStats.Chdir}
			ch <- e.mustNewGaugeMetric(workerAppDescs["startup_time_seconds"], float64(appStats.StartupTime), labelValues...)

			ch <- e.mustNewCounterMetric(workerAppDescs["requests_total"], float64(appStats.Requests), labelValues...)
			ch <- e.mustNewCounterMetric(workerAppDescs["exceptions_total"], float64(appStats.Exceptions), labelValues...)
		}

		// Worker Cores
		ch <- e.mustNewGaugeMetric(workerDescs["cores"], float64(len(workerStats.Cores)), labelValues...)
		if e.CollectCores {
			for _, coreStats := range workerStats.Cores {
				labelValues := []string{strconv.Itoa(workerStats.ID), strconv.Itoa(coreStats.ID)}
				ch <- e.mustNewGaugeMetric(workerCoreDescs["busy"], float64(coreStats.InRequest), labelValues...)

				ch <- e.mustNewCounterMetric(workerCoreDescs["requests_total"], float64(coreStats.Requests), labelValues...)
				ch <- e.mustNewCounterMetric(workerCoreDescs["static_requests_total"], float64(coreStats.StaticRequests), labelValues...)
				ch <- e.mustNewCounterMetric(workerCoreDescs["routed_requests_total"], float64(coreStats.RoutedRequests), labelValues...)
				ch <- e.mustNewCounterMetric(workerCoreDescs["offloaded_requests_total"], float64(coreStats.OffloadedRequests), labelValues...)
				ch <- e.mustNewCounterMetric(workerCoreDescs["write_errors_total"], float64(coreStats.WriteErrors), labelValues...)
				ch <- e.mustNewCounterMetric(workerCoreDescs["read_errors_total"], float64(coreStats.ReadErrors), labelValues...)
			}
		}
	}

	for _, cacheStats := range stats.Caches {
		labelValues := []string{cacheStats.Name}
		ch <- e.mustNewCounterMetric(cacheDescs["hits"], float64(cacheStats.Hits), labelValues...)
		ch <- e.mustNewCounterMetric(cacheDescs["misses"], float64(cacheStats.Misses), labelValues...)
		ch <- e.mustNewCounterMetric(cacheDescs["full"], float64(cacheStats.Full), labelValues...)

		ch <- e.mustNewGaugeMetric(cacheDescs["items"], float64(cacheStats.Items), labelValues...)
		ch <- e.mustNewGaugeMetric(cacheDescs["max_items"], float64(cacheStats.MaxItems), labelValues...)
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
