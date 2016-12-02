package exporter

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type Exporter struct {
	mutex sync.RWMutex

	uri         string
	timeout     time.Duration
	statsReader StatsReader

	scrapeDurations *prometheus.SummaryVec
	up              *prometheus.GaugeVec
	gaugesMap       GaugesMap
	countersMap     CountersMap
}

type Gauges map[string]*prometheus.GaugeVec
type Counters map[string]*prometheus.CounterVec

type GaugesMap map[string]Gauges
type CountersMap map[string]Counters

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
	gaugeMetricsMap = map[string]map[string]string{
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
		},

		workerAppSubsystem: map[string]string{
			"startup_time_seconds": "How long this app took to start.",
		},
	}

	counterMetricsMap = map[string]map[string]string{
		workerSubsystem: map[string]string{
			"requests_total":          "Total number of requests.",
			"exceptions_total":        "Total number of exceptions.",
			"harakiri_count_total":    "Total number of harakiri count.",
			"signals_total":           "Total number of signals.",
			"respawn_count_total":     "Total number of respawn count.",
			"transmitted_bytes_total": "Worker transmitted bytes.",
		},

		workerAppSubsystem: map[string]string{
			"requests_total":   "Total number of requests.",
			"exceptions_total": "Total number of exceptions.",
		},

		workerCoreSubsystem: map[string]string{
			"requests_total":           "Total number of requests.",
			"static_requests_total":    "Total number of static requests.",
			"routed_reqeusts_total":    "Total number of routed requests.",
			"offloaded_requests_total": "Total number of offloaded requests.",
			"write_errors_total":       "Total number of write errors.",
			"read_errors_total":        "Total number of read errors.",
			"in_requests_total":        "Total number of requests in.",
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
func NewExporter(uri string, timeout time.Duration) *Exporter {
	gaugesMap := GaugesMap{}
	countersMap := CountersMap{}
	constLabels := prometheus.Labels{"stats_uri": uri}

	for subsystem, gaugeMetrics := range gaugeMetricsMap {
		gauges := Gauges{}
		for name, help := range gaugeMetrics {
			gauges[name] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   subsystem,
				Name:        name,
				Help:        help,
				ConstLabels: constLabels,
			}, labelsMap[subsystem])
		}

		gaugesMap[subsystem] = gauges
	}

	for subsystem, counterMetrics := range counterMetricsMap {
		counters := Counters{}
		for name, help := range counterMetrics {
			counters[name] = prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   subsystem,
				Name:        name,
				Help:        help,
				ConstLabels: constLabels,
			}, labelsMap[subsystem])
		}

		countersMap[subsystem] = counters
	}

	statsReader, err := NewStatsReader(uri, timeout)
	if err != nil {
		log.Fatal(err)
	}

	return &Exporter{
		uri:         uri,
		timeout:     timeout,
		statsReader: statsReader,

		scrapeDurations: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrape_duration_seconds",
			Help:      "uwsgi_exporter: Duration of a scrape job.",
		}, []string{"result"}),
		up: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "up",
			Help:        "Was uwsgi stats query successful?",
			ConstLabels: constLabels,
		}, labelsMap[mainSubsystem]),
		gaugesMap:   gaugesMap,
		countersMap: countersMap,
	}
}

// Describe describes all the metrics ever exported by the exporter.
// It implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.scrapeDurations.Describe(ch)

	for _, vecs := range e.gaugesMap {
		for _, vec := range vecs {
			vec.Describe(ch)
		}
	}

	for _, vecs := range e.countersMap {
		for _, vec := range vecs {
			vec.Describe(ch)
		}
	}
}

// Collect fetches the stats from configured uwsgi stats location and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := e.execute(ch)
	duration := time.Since(begin)
	var result string

	if err != nil {
		log.Errorf("ERROR: collector failed after %fs: %s", duration.Seconds(), err)
		result = "error"
	} else {
		log.Debugf("OK: collector failed after %fs.", duration.Seconds())
		result = "success"
	}

	e.scrapeDurations.WithLabelValues(result).Observe(duration.Seconds())
}

func (e *Exporter) execute(ch chan<- prometheus.Metric) error {
	e.mutex.Lock() // To prevent metrics from concurrent collects.
	defer e.mutex.Unlock()

	// Reset metrics.
	for _, vecs := range e.gaugesMap {
		for _, vec := range vecs {
			vec.Reset()
		}
	}

	for _, vecs := range e.countersMap {
		for _, vec := range vecs {
			vec.Reset()
		}
	}

	// Read stats from uwsgi server
	body, err := e.statsReader.Read()
	if err != nil {
		return err
	}

	// Parse stats JSON data into struct
	var uwsgiStats UWSGIStats
	err = json.Unmarshal(body, uwsgiStats)
	if err != nil {
		log.Errorf("Failed to unmarshal JSON into struct: %s", err)
		e.setValueForUp(0)
		return err
	}

	// Parse stats into metrics
	e.parseStats(&uwsgiStats)

	// Report metrics
	for _, vecs := range e.gaugesMap {
		for _, vec := range vecs {
			vec.Collect(ch)
		}
	}

	for _, vecs := range e.countersMap {
		for _, vec := range vecs {
			vec.Collect(ch)
		}
	}

	return nil
}

func (e *Exporter) setValueForUp(v float64) {
	e.up.WithLabelValues(e.uri).Set(v)
}

func (e *Exporter) parseStats(stats *UWSGIStats) {
	// Main
	mainGauges := e.gaugesMap[mainSubsystem]
	mainGauges["listen_queue_length"].WithLabelValues().Set(float64(stats.ListenQueue))
	mainGauges["listen_queue_errors"].WithLabelValues().Set(float64(stats.ListenQueueErrors))
	mainGauges["signal_queue_length"].WithLabelValues().Set(float64(stats.SignalQueue))
	mainGauges["workers"].WithLabelValues().Set(float64(len(stats.Workers)))

	// Sockets
	socketGauges := e.gaugesMap[socketSubsystem]
	for _, socketStats := range stats.Sockets {
		labelValues := []string{socketStats.Name, socketStats.Proto}

		socketGauges["queue_length"].WithLabelValues(labelValues...).Set(float64(socketStats.Queue))
		socketGauges["max_queue_length"].WithLabelValues(labelValues...).Set(float64(socketStats.MaxQueue))
		socketGauges["shared"].WithLabelValues(labelValues...).Set(float64(socketStats.Shared))
		socketGauges["can_offload"].WithLabelValues(labelValues...).Set(float64(socketStats.CanOffload))
	}

	// Workers
	workerGauges := e.gaugesMap[workerSubsystem]
	workerCounters := e.countersMap[workerSubsystem]
	workerAppGauges := e.gaugesMap[workerAppSubsystem]
	workerAppCounters := e.countersMap[workerAppSubsystem]
	workerCoreCounters := e.gaugesMap[workerCoreSubsystem]

	for _, workerStats := range stats.Workers {
		labelValues := []string{strconv.Itoa(workerStats.ID), workerStats.Status}

		workerGauges["accepting"].WithLabelValues(labelValues...).Set(float64(workerStats.Accepting))
		workerGauges["delta_requests"].WithLabelValues(labelValues...).Set(float64(workerStats.DeltaRequests))
		workerGauges["signal_queue_length"].WithLabelValues(labelValues...).Set(float64(workerStats.SignalQueue))
		workerGauges["rss_bytes"].WithLabelValues(labelValues...).Set(float64(workerStats.RSS))
		workerGauges["vsz_bytes"].WithLabelValues(labelValues...).Set(float64(workerStats.VSZ))
		workerGauges["running_time_seconds"].WithLabelValues(labelValues...).Set(float64(workerStats.RunningTime) / usDivider)
		workerGauges["last_spawn_time_seconds"].WithLabelValues(labelValues...).Set(float64(workerStats.LastSpawn))
		workerGauges["average_response_time_seconds"].WithLabelValues(labelValues...).Set(float64(workerStats.AvgRt) / usDivider)

		workerCounters["requests_total"].WithLabelValues(labelValues...).Set(float64(workerStats.Requests))
		workerCounters["exceptions_total"].WithLabelValues(labelValues...).Set(float64(workerStats.Exceptions))
		workerCounters["harakiri_count_total"].WithLabelValues(labelValues...).Set(float64(workerStats.HarakiriCount))
		workerCounters["signals_total"].WithLabelValues(labelValues...).Set(float64(workerStats.Signals))
		workerCounters["respawn_count_total"].WithLabelValues(labelValues...).Set(float64(workerStats.RespawnCount))
		workerCounters["transmitted_bytes_total"].WithLabelValues(labelValues...).Set(float64(workerStats.TX))

		// Worker Apps
		for _, appStats := range workerStats.Apps {
			appLabelValues := append(labelValues, strconv.Itoa(appStats.ID), appStats.Mountpoint, appStats.Chdir)
			workerAppGauges["startup_time_seconds"].WithLabelValues(appLabelValues...).Set(float64(appStats.StartupTime))

			workerAppCounters["requests_total"].WithLabelValues(appLabelValues...).Set(float64(appStats.Requests))
			workerAppCounters["exceptions_total"].WithLabelValues(appLabelValues...).Set(float64(appStats.Exceptions))
		}

		// Worker Cores
		for _, coreStats := range workerStats.Cores {
			coreLabelValues := append(labelValues, strconv.Itoa(coreStats.ID))

			workerCoreCounters["requests_total"].WithLabelValues(coreLabelValues...).Set(float64(coreStats.Requests))
			workerCoreCounters["static_requests_total"].WithLabelValues(coreLabelValues...).Set(float64(coreStats.StaticRequests))
			workerCoreCounters["routed_reqeusts_total"].WithLabelValues(coreLabelValues...).Set(float64(coreStats.RoutedRequests))
			workerCoreCounters["offloaded_requests_total"].WithLabelValues(coreLabelValues...).Set(float64(coreStats.OffloadedRequests))
			workerCoreCounters["write_errors_total"].WithLabelValues(coreLabelValues...).Set(float64(coreStats.WriteErrors))
			workerCoreCounters["read_errors_total"].WithLabelValues(coreLabelValues...).Set(float64(coreStats.ReadErrors))
			workerCoreCounters["in_requests_total"].WithLabelValues(coreLabelValues...).Set(float64(coreStats.InRequests))
		}
	}
}
