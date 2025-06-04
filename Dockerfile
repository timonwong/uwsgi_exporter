ARG ARCH="amd64"
ARG OS="linux"

FROM --platform=${OS}/${ARCH} quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL maintainer="Timon Wong <timon86.wang@gmail.com>"

ARG ARCH="amd64"
ARG OS="linux"

COPY .build/${OS}-${ARCH}/uwsgi_exporter /bin/uwsgi_exporter

USER        nobody
EXPOSE      9117
ENTRYPOINT  [ "/bin/uwsgi_exporter" ]
