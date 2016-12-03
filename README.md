# uWSGI Exporter [![Build Status](https://travis-ci.org/timonwong/uwsgi_exporter.svg)][travis]

[![CircleCI](https://circleci.com/gh/timonwong/uwsgi_exporter/tree/master.svg?style=shield)][circleci]
[![Docker Repository on Quay](https://quay.io/repository/timonwong/uwsgi-exporter/status)][quay]
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
stats.uri                                  | URI for accessing uwsgi stats.
stats.timeout                              | Timeout for trying to get stats from uwsgi. (default 5s)
collect.cores                              | Whether to collect cores information per uwsgi worker. (default: true)
log.level                                  | Logging verbosity. (default: info)
web.listen-address                         | Address to listen on for web interface and telemetry. (default: ":9117")
web.telemetry-path                         | Path under which to expose metrics.
version                                    | Print the version information.

## Using Docker

You can deploy this exporter using the [timonwong/uwsgi-exporter](https://registry.hub.docker.com/u/timonwong/uwsgi-exporter/) Docker image.

For example:

```bash
docker pull timonwong/uwsgi-exporter

docker run -d -p 9117:9117 timonwong/uwsgi-exporter
```

[uWSGI]: https://uwsgi-docs.readthedocs.io
[circleci]: https://circleci.com/gh/timonwong/uwsgi_exporter
[hub]: https://hub.docker.com/r/timonwong/uwsgi-exporter/
[travis]: https://travis-ci.org/timonwong/uwsgi_exporter
[quay]: https://quay.io/repository/timonwong/uwsgi-exporter
