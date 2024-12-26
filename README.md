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

#### Single exporter mode

```bash
./uwsgi_exporter <flags>
```

#### Multi-target support (BETA)

This exporter supports the [multi-target pattern](https://prometheus.io/docs/guides/multi-target-exporter/). This allows running a single instance of this exporter for multiple uWSGI targets.

To use the multi-target functionality, send an http request to the endpoint /probe?target=http://uwsgi1.example.com:5432 where target is set to the URI of the uWSGI instance to scrape metrics from.

On the prometheus side you can set a scrape config as follows

Please note that only `http`, `https`, and `tcp` targets are supported (`file` and `unix` targets are disabled for security reasons).

```yaml
scrape_configs:
  - job_name: uwsgi # To get metrics about the mysql exporter’s targets
    static_configs:
      - targets:
        # All uwsgi hostnames to monitor.
        - http://uwsgi1.example.com:5432
        - http://uwsgi2.example.com:5432
    metrics_path: "/probe"
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        # The uwsgi_exporter host:port
        replacement: localhost:9117
```

### Flags

| Name               | Description                                                                                                                                             |
|--------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| stats.uri          | **required** URI for accessing uwsgi stats (currently supports: "http", "https", "unix", "tcp").                                                        |
| stats.timeout      | Timeout for trying to get stats from uwsgi. (deprecated)                                                                                                |
| timeout-offset     | Offset to subtract from timeout in seconds. (default 0.25)                                                                                              |
| collect.cores      | Whether to collect cores information per uwsgi worker. **WARNING** may cause tremendous resource utilization when using gevent engine. (default: false) |
| log.level          | Logging verbosity. (default: info)                                                                                                                      |
| web.config.file    | Path to a [web configuration file](#tls-and-basic-authentication)                                                                                       |
| web.listen-address | Address to listen on for web interface and telemetry. (default: ":9117")                                                                                |
| web.telemetry-path | Path under which to expose metrics.                                                                                                                     |
| version            | Print the version information.                                                                                                                          |

## TLS and basic authentication

The uWSGI Exporter supports TLS and basic authentication.

To use TLS and/or basic authentication, you need to pass a configuration file
using the `--web.config.file` parameter. The format of the file is described
[in the exporter-toolkit repository](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md).

## Using Docker

You can deploy this exporter using the Docker image from following registry:

- [DockerHub]\: [timonwong/uwsgi-exporter](https://registry.hub.docker.com/u/timonwong/uwsgi-exporter/)

For example:

```bash
docker pull timonwong/uwsgi-exporter

docker run -d -p 9117:9117 timonwong/uwsgi-exporter --stats.uri localhost:8001
```

(uWSGI Stats Server port, 8001 in this example, is configured in `ini` uWSGI configuration files)

[uwsgi]: https://uwsgi-docs.readthedocs.io
[circleci]: https://circleci.com/gh/timonwong/uwsgi_exporter
[hub]: https://hub.docker.com/r/timonwong/uwsgi-exporter/
[travis]: https://travis-ci.org/timonwong/uwsgi_exporter
[dockerhub]: https://hub.docker.com
