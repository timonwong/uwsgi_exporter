package main

import (
	"io"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"

	"github.com/timonwong/uwsgi_exporter/exporter"
)

var (
	listenAddress    = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interfaces.").Default(":9117").String()
	metricsPath      = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	statsURI         = kingpin.Flag("stats.uri", "URI for accessing uwsgi stats.").Default("").String()
	statsTimeout     = kingpin.Flag("stats.timeout", "Timeout for trying to get stats from uwsgi.").Default("5s").Duration()
	collectCores     = kingpin.Flag("collect.cores", "Collect cores information per uwsgi worker.").Default("false").Bool()
	applicationLabel = kingpin.Flag("application.label", "Label for WSGI application.").Default("").String()
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

	uwsgiExporter, err := exporter.NewExporter(logger, *statsURI, *statsTimeout, *collectCores, *applicationLabel)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to start logger", "error", err)
		os.Exit(1)
	}

	prometheus.MustRegister(uwsgiExporter)

	http.Handle(*metricsPath, promhttp.Handler()) // nolint: staticcheck
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html>
			<head><title>uWSGI Exporter</title></head>
			<body>
			<h1>uWSGI Exporter</h1>
			<p><a href="`+*metricsPath+`">Metrics</a></p>
            <h2>Build</h2>
            <pre>`+version.Info()+` `+version.BuildContext()+`</pre>
			</body>
			</html>`)
	})
	http.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, "ok")
	})

	level.Info(logger).Log("msg", "Listening on", "addr", *listenAddress)

	err = http.ListenAndServe(*listenAddress, nil) //#nosec
	if err != nil {
		level.Error(logger).Log("msg", "Failed to listen address", "error", err)
		os.Exit(1)
	}
}
