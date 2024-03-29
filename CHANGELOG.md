## 1.3.0 / 2023-03-23

### Major changes

Now `--stats.timeout` is deprecated and takes no effect. The exporter now honors `X-Prometheus-Scrape-Timeout-Seconds` header
from Prometheus to determine the timeout. If the header is not set, the exporter will use the default timeout value of 120 seconds.

### Changes

* chore: Update readme about web.config file by @timonwong in https://github.com/timonwong/uwsgi_exporter/pull/65
* feat: Add support to multi-targets by @timonwong in https://github.com/timonwong/uwsgi_exporter/pull/66
* docs: add missing changelog by @timonwong in https://github.com/timonwong/uwsgi_exporter/pull/67
* chore: fail scrape if stats reader cannot be created by @timonwong in https://github.com/timonwong/uwsgi_exporter/pull/69
* feat: reduce sockets in TIME-WAIT state by @timonwong in https://github.com/timonwong/uwsgi_exporter/pull/70

## 1.2.0 / 2023-03-22

* build(deps): Bump github.com/prometheus/client_golang from 1.12.1 to 1.13.0 by @dependabot in https://github.com/timonwong/uwsgi_exporter/pull/47
* build(deps): Bump github.com/prometheus/common from 0.34.0 to 0.37.0 by @dependabot in https://github.com/timonwong/uwsgi_exporter/pull/46
* chore: bump github.com/prometheus/client_model from 0.2.0 to 0.3.0 by @dependabot in https://github.com/timonwong/uwsgi_exporter/pull/49
* Bump github.com/stretchr/testify from 1.8.0 to 1.8.1 by @dependabot in https://github.com/timonwong/uwsgi_exporter/pull/50
* feat: allow disable timeout by @timonwong in https://github.com/timonwong/uwsgi_exporter/pull/61
* feat: Add support for "idle" and "cheap" status by @timonwong in https://github.com/timonwong/uwsgi_exporter/pull/62
* ci: fix build by @timonwong in https://github.com/timonwong/uwsgi_exporter/pull/63
* chore: Use exporter toolkit by @timonwong in https://github.com/timonwong/uwsgi_exporter/pull/64

## 1.1.0 / 2022-08-11

* No other changes, just go version bump.

## 1.0.0 / 2019-12-18

*[CHANGE] Change default value `--collect.cores` to `false` #35
