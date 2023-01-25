# prom_metrics2influxdb
**Prometheus Metrics Import - fetches metrics and forwards them to InfluxDB**


For some use cases, like jobs running only once in 1-3 days, prometheus is not the best choice. My experience is, that InfluxDB is much better suited for this if you want to use grafana with the data.

So a simple tool, which scrapes the metrics provided by a service for prometheus and stores the values in InfluxDB was needed, and here it is.

## Assumptions about Metrics

Currently only metrics from type "gauge" are supported.

## Container

### Public Container Image

To run the public available image:

```bash
podman run --rm -v <path>/config.yaml:/config.yaml registry.opensuse.org/home/kukuk/containerfile/prom_metrics2influxdb
```

You can replace `podman` with `docker` without any further changes.

### Build locally

To build the container image with the `prom_metrics2influxdb` binary included run:

```bash
sudo podman build --rm --no-cache --build-arg VERSION=$(cat VERSION) --build-arg BUILDTIME=$(date +%Y-%m-%dT%TZ) -t prom_metrics2influxdb .
```

You can of cource replace `podman` with `docker`, no other arguments needs to be adjusted.

## Configuration

The prom_metrics2influxdb will be configured via command line and configuration file.

### Commandline

Available options are:
```plaintext
Usage:
  prom_metrics2influxdb [flags]

Flags:
  -c, --config string   configuration file (default "config.yaml")
  -h, --help            help for prom_metrics2influxdb
  -q, --quiet           don't print any informative messages
  -v, --verbose         become really verbose in printing messages
      --version         version for prom_metrics2influxdb
```

### Configuration File

By default `prom_metrics2influxdb` looks for the file `config.yaml` in the local directory. This can be overriden with the `--config` option.

Here is my configuration file:

```yaml
# Required: URL of the metrics to scrape
metrics: https://example.com/metrics
# Required: The measurement under which the values are stored in InfluxDB
measurement: prom_metrics
# Optional: if specified, the value of this metric is used as timestamp
# to store the data in the database, else the current time is used.
# Helpful for jobs not running regular
timestamp: last_successful_run
# Optional: if true, the metrics are only stored if the timestamp
# is newer
avoid_duplicate: true
# Optional, specifies the interval in which the metrics are fetched
# default is 1 hour
interval: 6h
influxdb:
  # machine on which influxdb runs on port 8086:
  server: <influxdb host>
  # Database or bucket or however it will be called in influxdb3...
  database: <my-db>
  # Optional, only used if a new database/bucket needs to be created
  # organization: <my-org>
  # If a token is required, you can specify it here (but be careful that you
  # don't commit it a public git repo or something similar! Or you can use
  # an environment variable 'INFLUXDB_TOKEN'
  # token: <token>
```
