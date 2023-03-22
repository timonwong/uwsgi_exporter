package main

import (
	"io"
	"net/http"
	_ "net/http/pprof" //#nosec
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"

	"github.com/timonwong/uwsgi_exporter/pkg/exporter"
)

func init() {
	prometheus.MustRegister(version.NewCollector("uwsgi_exporter"))
}

func main() {
	var (
		metricsPath  = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		statsURI     = kingpin.Flag("stats.uri", "URI for accessing uwsgi stats.").Default("").String()
		statsTimeout = kingpin.Flag("stats.timeout", "Timeout for trying to get stats from uwsgi.").Default("5s").Duration()
		collectCores = kingpin.Flag("collect.cores", "Collect cores information per uwsgi worker.").Default("false").Bool()
		webConfig    = webflag.AddFlags(kingpin.CommandLine, ":9117")
	)

	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)

	kingpin.Version(version.Print("uwsgi_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)
	level.Info(logger).Log("msg", "Starting uwsgi_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build", version.BuildContext())

	uwsgiExporter, err := exporter.NewExporter(logger, *statsURI, *statsTimeout, *collectCores)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to start logger", "error", err)
		os.Exit(1)
	}

	prometheus.MustRegister(uwsgiExporter)

	http.Handle(*metricsPath, promhttp.Handler())
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
	http.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, "ok")
	})

	srv := &http.Server{} //#nosec
	err = web.ListenAndServe(srv, webConfig, logger)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to listen address", "error", err)
		os.Exit(1)
	}
}
