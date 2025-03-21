# Postfix Exporter

[![tests](https://github.com/sergeymakinen/postfix_exporter/workflows/tests/badge.svg)](https://github.com/sergeymakinen/postfix_exporter/actions?query=workflow%3Atests)
[![Go Report Card](https://goreportcard.com/badge/github.com/sergeymakinen/postfix_exporter/v2)](https://goreportcard.com/report/github.com/sergeymakinen/postfix_exporter/v2)
[![codecov](https://codecov.io/gh/sergeymakinen/postfix_exporter/branch/main/graph/badge.svg)](https://codecov.io/gh/sergeymakinen/postfix_exporter)
[![Docker Pulls](https://img.shields.io/docker/pulls/sergeymakinen/postfix_exporter)](https://hub.docker.com/r/sergeymakinen/postfix_exporter)

Export Postfix stats from logs to Prometheus.

To run it:

```bash
make
./postfix_exporter [flags]
```

## Using Docker

You can deploy this exporter using
the [sergeymakinen/postfix_exporter](https://hub.docker.com/r/sergeymakinen/postfix_exporter) Docker image.

For example:

```bash
docker pull sergeymakinen/postfix_exporter

docker run -d -p 9907:9907 -v postfix_logs:/var/log/postfix sergeymakinen/postfix_exporter \
  --file.log /var/log/postfix/postfix.log
```

## Exported metrics

| Metric | Meaning | Labels
| --- | --- | ---
| postfix_errors_total | Total number of log records parsing resulted in an error. |
| postfix_foreign_total | Total number of foreign log records. |
| postfix_unsupported_total | Total number of unsupported log records. |
| postfix_postscreen_actions_total | Total number of times postscreen events were collected. | action
| postfix_connects_total | Total number of times connect events were collected. | subprogram
| postfix_disconnects_total | Total number of times disconnect events were collected. | subprogram
| postfix_lost_connections_total | Total number of times lost connection events were collected. | subprogram
| postfix_not_resolved_hostnames_total | Total number of times not resolved hostname events were collected. | subprogram
| postfix_statuses_total | Total number of times server message status change events were collected. | subprogram, status
| postfix_delay_seconds | Delay in seconds for a server to process a message. | subprogram, status
| postfix_status_replies_total | Total number of times server message status change event replies were collected. Requires [configuration](CONFIGURATION.md) to be present. | subprogram, status, code, enhanced_code, text
| postfix_smtp_replies_total | Total number of times SMTP server replies were collected. Requires [configuration](CONFIGURATION.md) to be present. | code, enhanced_code, text
| postfix_milter_actions_total | Total number of times milter events were collected. | subprogram, action
| postfix_login_failures_total | Total number of times login failure events were collected. | subprogram, method
| postfix_qmgr_statuses_total | Total number of times Postfix queue manager message status change events were collected. | status
| postfix_logs_total | Total number of log records processed. | subprogram, severity
| postfix_noqueue_reject_replies_total | Total number of times NOQUEUE: reject event replies were collected. Requires [configuration](CONFIGURATION.md) to be present. | subprogram, command, code, enhanced_code, text

## Flags

```bash
./postfix_exporter --help
```

* __`config.file`:__ Postfix exporter [configuration file](CONFIGURATION.md).
* __`config.check`:__ If true, validate the config file and then exit.
* __`collector`:__ Collector type to scrape metrics with. `file` or `journald`.
* __`postfix.instance`:__ Postfix instance name. `postfix` by default.
* __`file.log`:__ Path to a file containing Postfix logs. Example: `/var/log/mail.log`.
* __`journald.path`:__ Path where a systemd journal residing in. A local journal is being used by default.
* __`journald.unit`:__ Postfix systemd service name. `postfix@-.service` by default.
* __`journald.since`:__ Time since which to read from a systemd journal. Now by default.
* __`test`:__ If true, read logs, print metrics and then exit.
* __`web.listen-address`:__ Address to listen on for web interface and telemetry.
* __`web.telemetry-path`:__ Path under which to expose metrics.
* __`log.level`:__ Logging level. `info` by default.
* __`log.format`:__ Set the log target and format. Example: `logger:syslog?appname=bob&local=7`
  or `logger:stdout?json=true`.

### TLS and basic authentication

The postfix_exporter supports TLS and basic authentication.
To use TLS and/or basic authentication, you need to pass a configuration file
using the `--web.config.file` parameter. The format of the file is described
[in the exporter-toolkit repository](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md).
