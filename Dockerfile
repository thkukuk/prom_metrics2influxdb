FROM registry.opensuse.org/opensuse/tumbleweed:latest AS build-stage
RUN zypper install --no-recommends --auto-agree-with-product-licenses -y git go make
#RUN git clone https://github.com/thkukuk/prom_metrics2influxdb
COPY . prom_metrics2influxdb
RUN cd prom_metrics2influxdb && make update && make tidy && make

FROM registry.opensuse.org/opensuse/busybox:latest
LABEL maintainer="Thorsten Kukuk <kukuk@thkukuk.de>"

ARG BUILDTIME=
ARG VERSION=unreleased
LABEL org.opencontainers.image.title="Prometheus Metrics Import to InfluxDB"
LABEL org.opencontainers.image.description="Imports Metrics for Prometheus and stores the values in InfluxDB"
LABEL org.opencontainers.image.created=$BUILDTIME
LABEL org.opencontainers.image.version=$VERSION

COPY --from=build-stage /prom_metrics2influxdb/bin/prom_metrics2influxdb /usr/local/bin
COPY entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/usr/local/bin/prom_metrics2influxdb"]
