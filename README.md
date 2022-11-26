# uWSGI Exporter

[![CircleCI](https://circleci.com/gh/timonwong/uwsgi_exporter/tree/master.svg?style=shield)][circleci]
[![Docker Pulls](https://img.shields.io/docker/pulls/timonwong/uwsgi-exporter.svg?maxAge=604800)][hub]
[![Go Report Card](https://goreportcard.com/badge/github.com/timonwong/uwsgi_exporter)](https://goreportcard.com/report/github.com/timonwong/uwsgi_exporter)

Prometheus exporter for [uWSGI] metrics.

## Building and running

### Build

```bash
make
```

### Running

```bash
./uwsgi_exporter <flags>
```

### Flags

Name                                       | Description
-------------------------------------------|--------------------------------------------------------------------------------------------------
--stats.uri                                | **required** URI for accessing uwsgi stats (currently supports: "http", "https", "unix", "tcp").
--stats.timeout                            | Timeout for trying to get stats from uwsgi. (default 5s)
--collect.cores                            | Whether to collect cores information per uwsgi worker. **WARNING** may cause tremendous resource utilization when using gevent engine. (default: false)
--log.level                                | Logging verbosity. (default: info)
--web.listen-address                       | Address to listen on for web interface and telemetry. (default: ":9117")
--web.telemetry-path                       | Path under which to expose metrics.
--version                                  | Print the version information.

