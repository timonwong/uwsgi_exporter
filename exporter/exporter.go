package exporter

import (
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// UwsgiExporter collects uwsgi metrics for prometheus.
type UwsgiExporter struct {
	mu sync.Mutex

	logger log.Logger

	statsReader  StatsReader
	uri          string
	collectCores bool

	uwsgiUp         prometheus.Gauge
	scrapeDurations *prometheus.SummaryVec
	descriptorsMap  DescriptorsMap
}

// Descriptors is a map for `prometheus.Desc` pointer.
type Descriptors map[string]*prometheus.Desc

// DescriptorsMap is a map for `Descriptors`.
type DescriptorsMap map[string]Descriptors

const (
	namespace           = "uwsgi"
	mainSubsystem       = ""
	socketSubsystem     = "socket"
	workerSubsystem     = "worker"
	workerAppSubsystem  = "worker_app"
	workerCoreSubsystem = "worker_core"
	cacheSubsystem      = "cache"

	usDivider = float64(time.Second / time.Microsecond)
)

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

			"busy":                    "Is busy",
			"requests_total":          "Total number of requests.",
			"exceptions_total":        "Total number of exceptions.",
			"harakiri_count_total":    "Total number of harakiri count.",
			"signals_total":           "Total number of signals.",
			"respawn_count_total":     "Total number of respawn count.",
			"transmitted_bytes_total": "Worker transmitted bytes.",
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

// NewExporter creates a new uwsgi exporter.
func NewExporter(logger log.Logger, uri string, timeout time.Duration, collectCores bool) (*UwsgiExporter, error) {
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

	statsReader, err := NewStatsReader(uri, timeout)
	if err != nil {
		return nil, err
	}

	return &UwsgiExporter{
		logger: logger,

		uri:          uri,
		collectCores: collectCores,
		statsReader:  statsReader,

		uwsgiUp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Whether the uwsgi server is up.",
		}),
		scrapeDurations: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrape_duration_seconds",
			Help:      "uwsgi_exporter: Duration of a scrape job.",
		}, []string{"result"}),
		descriptorsMap: descriptorsMap,
	}, nil
}

// Describe describes all the metrics ever exported by the exporter.
// It implements prometheus.Collector.
func (e *UwsgiExporter) Describe(ch chan<- *prometheus.Desc) {
	e.uwsgiUp.Describe(ch)
	e.scrapeDurations.Describe(ch)

	for _, descs := range e.descriptorsMap {
		for _, desc := range descs {
			ch <- desc
		}
	}
}

// Collect fetches the stats from configured uwsgi stats location and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *UwsgiExporter) Collect(ch chan<- prometheus.Metric) {
	startTime := time.Now()
	err := e.execute(ch)
	d := time.Since(startTime).Seconds()

	logger := log.With(e.logger, "duration", d)
	if err != nil {
		level.Error(logger).Log("msg", "Scrape failed", "error", err)
		e.uwsgiUp.Set(0)
		e.scrapeDurations.WithLabelValues("error").Observe(d)
	} else {
		level.Debug(logger).Log("msg", "Scrape successful")
		e.uwsgiUp.Set(1)
		e.scrapeDurations.WithLabelValues("success").Observe(d)
	}

	e.uwsgiUp.Collect(ch)
	e.scrapeDurations.Collect(ch)
}

func (e *UwsgiExporter) execute(ch chan<- prometheus.Metric) (err error) {
	e.mu.Lock() // To prevent stats reading from concurrent collects
	// Read (and parse) stats from uwsgi server
	uwsgiStats, err := e.statsReader.Read()
	if err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()

	// Collect metrics from stats
	e.collectMetrics(uwsgiStats, ch)
	return nil
}

func newGaugeMetric(desc *prometheus.Desc, value float64, labelValues ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, labelValues...)
}

func newCounterMetric(desc *prometheus.Desc, value float64, labelsValues ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(desc, prometheus.CounterValue, value, labelsValues...)
}

func (e *UwsgiExporter) collectMetrics(stats *UwsgiStats, ch chan<- prometheus.Metric) {
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
		if workerStats.Status == "busy" {
			ch <- newGaugeMetric(workerDescs["busy"], float64(1.0), labelValues...)
		} else {
			ch <- newGaugeMetric(workerDescs["busy"], float64(0.0), labelValues...)
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
