package exporter

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

// UWSGIExporter collects uwsgi metrics for prometheus.
type UWSGIExporter struct {
	mutex sync.RWMutex

	uri         string
	timeout     time.Duration
	statsReader StatsReader

	scrapeDurations *prometheus.SummaryVec
	descriptorsMap  DescriptorsMap
}

type Descriptors map[string]*prometheus.Desc
type DescriptorsMap map[string]Descriptors

const (
	namespace = "uwsgi"

	mainSubsystem       = ""
	socketSubsystem     = "socket"
	workerSubsystem     = "worker"
	workerAppSubsystem  = "worker_app"
	workerCoreSubsystem = "worker_core"

	usDivider = float64(time.Second / time.Microsecond)
)

var (
	metricsMap = map[string]map[string]string{
		mainSubsystem: map[string]string{
			"listen_queue_length": "Length of listen queue.",
			"listen_queue_errors": "Number of listen queue errors.",
			"signal_queue_length": "Length of signal queue.",
			"workers":             "Number of workers.",
		},

		socketSubsystem: map[string]string{
			"queue_length":     "Length of socket queue.",
			"max_queue_length": "Max length of socket queue.",
			"shared":           "Is shared socket?",
			"can_offload":      "Can socket offload?",
		},

		workerSubsystem: map[string]string{
			"accepting":                     "Is this worker accepting requests?",
			"delta_requests":                "Number of delta requests",
			"signal_queue_length":           "Length of signal queue.",
			"rss_bytes":                     "Worker RSS bytes.",
			"vsz_bytes":                     "Worker VSZ bytes.",
			"running_time_seconds":          "Worker running time in seconds.",
			"last_spawn_time_seconds":       "Last spawn time in seconds since epoch.",
			"average_response_time_seconds": "Average response time in seconds.",

			"requests_total":          "Total number of requests.",
			"exceptions_total":        "Total number of exceptions.",
			"harakiri_count_total":    "Total number of harakiri count.",
			"signals_total":           "Total number of signals.",
			"respawn_count_total":     "Total number of respawn count.",
			"transmitted_bytes_total": "Worker transmitted bytes.",
		},

		workerAppSubsystem: map[string]string{
			"startup_time_seconds": "How long this app took to start.",

			"requests_total":   "Total number of requests.",
			"exceptions_total": "Total number of exceptions.",
		},

		workerCoreSubsystem: map[string]string{
			"in_requests": "In requests?",

			"requests_total":           "Total number of requests.",
			"static_requests_total":    "Total number of static requests.",
			"routed_reqeusts_total":    "Total number of routed requests.",
			"offloaded_requests_total": "Total number of offloaded requests.",
			"write_errors_total":       "Total number of write errors.",
			"read_errors_total":        "Total number of read errors.",
		},
	}

	// Please note that stats_uri is omitted here, because it's const
	labelsMap = map[string][]string{
		mainSubsystem:       []string{},
		socketSubsystem:     []string{"name", "proto"},
		workerSubsystem:     []string{"worker_id", "status"},
		workerAppSubsystem:  []string{"worker_id", "status", "app_id", "mountpoint", "chdir"},
		workerCoreSubsystem: []string{"worker_id", "status", "core_id"},
	}
)

// NewExporter creates a new uwsgi exporter.
func NewExporter(uri string, timeout time.Duration) *UWSGIExporter {
	descriptorsMap := DescriptorsMap{}
	constLabels := prometheus.Labels{"stats_uri": uri}

	for subsystem, metrics := range metricsMap {
		descriptors := Descriptors{}
		for name, help := range metrics {
			descriptors[name] = prometheus.NewDesc(
				prometheus.BuildFQName(namespace, subsystem, name), help, labelsMap[subsystem], constLabels)
		}

		descriptorsMap[subsystem] = descriptors
	}

	statsReader, err := NewStatsReader(uri, timeout)
	if err != nil {
		log.Fatal(err)
	}

	return &UWSGIExporter{
		uri:         uri,
		timeout:     timeout,
		statsReader: statsReader,

		scrapeDurations: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrape_duration_seconds",
			Help:      "uwsgi_exporter: Duration of a scrape job.",
		}, []string{"result"}),
		descriptorsMap: descriptorsMap,
	}
}

// Describe describes all the metrics ever exported by the exporter.
// It implements prometheus.Collector.
func (e *UWSGIExporter) Describe(ch chan<- *prometheus.Desc) {
	e.scrapeDurations.Describe(ch)

	for _, descs := range e.descriptorsMap {
		for _, desc := range descs {
			ch <- desc
		}
	}
}

// Collect fetches the stats from configured uwsgi stats location and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *UWSGIExporter) Collect(ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := e.execute(ch)
	duration := time.Since(begin)
	var result string

	if err != nil {
		log.Errorf("ERROR: collector failed after %fs: %s", duration.Seconds(), err)
		result = "error"
	} else {
		log.Debugf("OK: collector successful after %fs.", duration.Seconds())
		result = "success"
	}

	e.scrapeDurations.WithLabelValues(result).Observe(duration.Seconds())
	e.scrapeDurations.Collect(ch)
}

func (e *UWSGIExporter) execute(ch chan<- prometheus.Metric) error {
	e.mutex.Lock() // To prevent metrics from concurrent collects.
	defer e.mutex.Unlock()

	// Read stats from uwsgi server
	body, err := e.statsReader.Read()
	if err != nil {
		return err
	}

	// Parse stats JSON data into struct
	var uwsgiStats UWSGIStats
	err = json.Unmarshal(body, &uwsgiStats)
	if err != nil {
		log.Errorf("Failed to unmarshal JSON into struct: %s", err)
		return err
	}

	// Parse stats into metrics
	e.collectStats(ch, &uwsgiStats)

	return nil
}

func (e *UWSGIExporter) collectStats(ch chan<- prometheus.Metric, stats *UWSGIStats) {
	// Main
	mainDescs := e.descriptorsMap[mainSubsystem]
	ch <- prometheus.MustNewConstMetric(mainDescs["listen_queue_length"], prometheus.GaugeValue, float64(stats.ListenQueue))
	ch <- prometheus.MustNewConstMetric(mainDescs["listen_queue_errors"], prometheus.GaugeValue, float64(stats.ListenQueueErrors))
	ch <- prometheus.MustNewConstMetric(mainDescs["signal_queue_length"], prometheus.GaugeValue, float64(stats.SignalQueue))
	ch <- prometheus.MustNewConstMetric(mainDescs["workers"], prometheus.GaugeValue, float64(len(stats.Workers)))

	// Sockets
	socketDescs := e.descriptorsMap[socketSubsystem]
	for _, socketStats := range stats.Sockets {
		labelValues := []string{socketStats.Name, socketStats.Proto}

		ch <- prometheus.MustNewConstMetric(socketDescs["queue_length"], prometheus.GaugeValue, float64(socketStats.Queue), labelValues...)
		ch <- prometheus.MustNewConstMetric(socketDescs["max_queue_length"], prometheus.GaugeValue, float64(socketStats.MaxQueue), labelValues...)
		ch <- prometheus.MustNewConstMetric(socketDescs["shared"], prometheus.GaugeValue, float64(socketStats.Shared), labelValues...)
		ch <- prometheus.MustNewConstMetric(socketDescs["can_offload"], prometheus.GaugeValue, float64(socketStats.CanOffload), labelValues...)
	}

	// Workers
	workerDescs := e.descriptorsMap[workerSubsystem]
	workerAppDescs := e.descriptorsMap[workerAppSubsystem]
	workerCoreDescs := e.descriptorsMap[workerCoreSubsystem]
	for _, workerStats := range stats.Workers {
		labelValues := []string{strconv.Itoa(workerStats.ID), workerStats.Status}

		ch <- prometheus.MustNewConstMetric(workerDescs["accepting"], prometheus.GaugeValue, float64(workerStats.Accepting), labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["delta_requests"], prometheus.GaugeValue, float64(workerStats.DeltaRequests), labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["signal_queue_length"], prometheus.GaugeValue, float64(workerStats.SignalQueue), labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["rss_bytes"], prometheus.GaugeValue, float64(workerStats.RSS), labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["vsz_bytes"], prometheus.GaugeValue, float64(workerStats.VSZ), labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["running_time_seconds"], prometheus.GaugeValue, float64(workerStats.RunningTime)/usDivider, labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["last_spawn_time_seconds"], prometheus.GaugeValue, float64(workerStats.LastSpawn), labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["average_response_time_seconds"], prometheus.GaugeValue, float64(workerStats.AvgRt)/usDivider, labelValues...)

		ch <- prometheus.MustNewConstMetric(workerDescs["requests_total"], prometheus.CounterValue, float64(workerStats.Requests), labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["exceptions_total"], prometheus.CounterValue, float64(workerStats.Exceptions), labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["harakiri_count_total"], prometheus.CounterValue, float64(workerStats.HarakiriCount), labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["signals_total"], prometheus.CounterValue, float64(workerStats.Signals), labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["respawn_count_total"], prometheus.CounterValue, float64(workerStats.RespawnCount), labelValues...)
		ch <- prometheus.MustNewConstMetric(workerDescs["transmitted_bytes_total"], prometheus.CounterValue, float64(workerStats.TX), labelValues...)

		// Worker Apps
		for _, appStats := range workerStats.Apps {
			labelValues := []string{strconv.Itoa(workerStats.ID), workerStats.Status, strconv.Itoa(appStats.ID), appStats.Mountpoint, appStats.Chdir}
			ch <- prometheus.MustNewConstMetric(workerAppDescs["startup_time_seconds"], prometheus.GaugeValue, float64(appStats.StartupTime), labelValues...)

			ch <- prometheus.MustNewConstMetric(workerAppDescs["requests_total"], prometheus.CounterValue, float64(appStats.Requests), labelValues...)
			ch <- prometheus.MustNewConstMetric(workerAppDescs["exceptions_total"], prometheus.CounterValue, float64(appStats.Exceptions), labelValues...)
		}

		// Worker Cores
		for _, coreStats := range workerStats.Cores {
			labelValues := []string{strconv.Itoa(workerStats.ID), workerStats.Status, strconv.Itoa(coreStats.ID)}
			ch <- prometheus.MustNewConstMetric(workerCoreDescs["in_requests"], prometheus.GaugeValue, float64(coreStats.InRequests), labelValues...)

			ch <- prometheus.MustNewConstMetric(workerCoreDescs["requests_total"], prometheus.CounterValue, float64(coreStats.Requests), labelValues...)
			ch <- prometheus.MustNewConstMetric(workerCoreDescs["static_requests_total"], prometheus.CounterValue, float64(coreStats.StaticRequests), labelValues...)
			ch <- prometheus.MustNewConstMetric(workerCoreDescs["routed_reqeusts_total"], prometheus.CounterValue, float64(coreStats.RoutedRequests), labelValues...)
			ch <- prometheus.MustNewConstMetric(workerCoreDescs["offloaded_requests_total"], prometheus.CounterValue, float64(coreStats.OffloadedRequests), labelValues...)
			ch <- prometheus.MustNewConstMetric(workerCoreDescs["write_errors_total"], prometheus.CounterValue, float64(coreStats.WriteErrors), labelValues...)
			ch <- prometheus.MustNewConstMetric(workerCoreDescs["read_errors_total"], prometheus.CounterValue, float64(coreStats.ReadErrors), labelValues...)
		}
	}
}
