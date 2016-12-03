package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/timonwong/uwsgi_exporter/exporter"
)

func init() {
	prometheus.MustRegister(version.NewCollector("uwsgi_exporter"))
}

func main() {
	var (
		showVersion   = flag.Bool("version", false, "Print version information.")
		listenAddress = flag.String("web.listen-address", ":9117", "Address on which to expose metrics and web interfaces.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		statsURI      = flag.String("stats.uri", "", "URI for accessing uwsgi stats.")
		statsTimeout  = flag.Duration("stats.timeout", 5*time.Second, "Timeout for trying to get stats from uwsgi.")
		collectCores  = flag.Bool("collect.cores", true, "Collect cores information per uwsgi worker.")
	)
	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stdout, version.Print("uwsgi_exporter"))
		os.Exit(0)
	}

	log.Infoln("Starting uwsgi_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	uwsgiExporter := exporter.NewExporter(*statsURI, *statsTimeout, *collectCores)
	prometheus.MustRegister(uwsgiExporter)

	handler := prometheus.Handler()

	http.Handle(*metricsPath, handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>uWSGI Exporter</title></head>
			<body>
			<h1>uWSGI Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}
