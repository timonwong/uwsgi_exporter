package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof" //#nosec
	"os"
	"strconv"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"

	"github.com/timonwong/uwsgi_exporter/pkg/collector"
)

var (
	metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	statsURI      = kingpin.Flag("stats.uri", "URI for accessing uwsgi stats.").Default("").String()
	timeoutOffset = kingpin.Flag("timeout-offset", "Offset to subtract from timeout in seconds.").Default("0.25").Float64()
	_             = kingpin.Flag("stats.timeout", "Timeout for trying to get stats from uwsgi (deprecated).").Duration()
	collectCores  = kingpin.Flag("collect.cores", "Collect cores information per uwsgi worker.").Default("false").Bool()
	webConfig     = webflag.AddFlags(kingpin.CommandLine, ":9117")
)

func init() {
	prometheus.MustRegister(version.NewCollector("uwsgi_exporter"))
}

func main() {
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)

	kingpin.Version(version.Print("uwsgi_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)
	level.Info(logger).Log("msg", "Starting uwsgi_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build", version.BuildContext())

	http.Handle(*metricsPath, newHandler(collector.NewMetrics(), logger))
	if *metricsPath != "/" && *metricsPath != "" {
		landingConfig := web.LandingConfig{
			Name:        "uWSGI Exporter",
			Description: "Prometheus Exporter for uWSGI.",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: *metricsPath,
					Text:    "Metrics",
				},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			level.Error(logger).Log("err", err)
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}
	http.HandleFunc("/probe", handleProbe(collector.NewMetrics(), logger))
	http.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, "ok")
	})

	srv := &http.Server{} //#nosec
	err := web.ListenAndServe(srv, webConfig, logger)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to listen address", "error", err)
		os.Exit(1)
	}
}

func newHandler(metrics collector.Metrics, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Use request context for cancellation when connection gets closed.
		timeoutSeconds, err := getTimeout(r, *timeoutOffset, logger)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse timeout from Prometheus header: %s", err), http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSeconds*float64(time.Second)))
		defer cancel()
		r = r.WithContext(ctx)

		registry := prometheus.NewRegistry()
		registry.MustRegister(collector.New(ctx, *statsURI, metrics, *collectCores, logger))

		gatherers := prometheus.Gatherers{
			prometheus.DefaultGatherer,
			registry,
		}
		// Delegate http serving to Prometheus client library, which will call collector.Collect.
		h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

func handleProbe(metrics collector.Metrics, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		target := params.Get("target")
		if target == "" {
			http.Error(w, "target is required", http.StatusBadRequest)
			return
		}

		timeoutSeconds, err := getTimeout(r, *timeoutOffset, logger)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse timeout from Prometheus header: %s", err), http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSeconds*float64(time.Second)))
		defer cancel()
		r = r.WithContext(ctx)

		registry := prometheus.NewRegistry()
		registry.MustRegister(collector.New(ctx, target, metrics, *collectCores, logger))

		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

func getTimeout(r *http.Request, offset float64, logger log.Logger) (timeoutSeconds float64, err error) {
	// If a timeout is configured via the Prometheus header, add it to the request.
	if v := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); v != "" {
		var err error
		timeoutSeconds, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, err
		}
	}
	if timeoutSeconds == 0 {
		timeoutSeconds = 120
	}

	if offset >= timeoutSeconds {
		// Ignore timeout offset if it doesn't leave time to scrape.
		level.Error(logger).Log("msg", "Timeout offset should be lower than prometheus scrape timeout", "offset", offset, "prometheus_scrape_timeout", timeoutSeconds)
	} else {
		// Subtract timeout offset from timeout.
		timeoutSeconds -= offset
	}

	return timeoutSeconds, nil
}
