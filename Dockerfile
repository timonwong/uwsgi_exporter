FROM        quay.io/prometheus/busybox:latest
MAINTAINER  Timon Wong <timon86.wang@gmail.com>

COPY uwsgi_exporter /bin/uwsgi_exporter

EXPOSE      9107
ENTRYPOINT  [ "/bin/uwsgi_exporter" ]
