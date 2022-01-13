# Postfix Exporter

[![tests](https://github.com/sergeymakinen/postfix_exporter/workflows/tests/badge.svg)](https://github.com/sergeymakinen/postfix_exporter/actions?query=workflow%3Atests)
[![Go Reference](https://pkg.go.dev/badge/github.com/sergeymakinen/postfix_exporter.svg)](https://pkg.go.dev/github.com/sergeymakinen/postfix_exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/sergeymakinen/postfix_exporter)](https://goreportcard.com/report/github.com/sergeymakinen/postfix_exporter)
[![codecov](https://codecov.io/gh/sergeymakinen/postfix_exporter/branch/main/graph/badge.svg)](https://codecov.io/gh/sergeymakinen/postfix_exporter)

Export Postfix stats from logs to Prometheus.

To run it:

```bash
make
./postfix_exporter [flags]
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
| postfix_lmtp_statuses_total | Total number of times a LMTP server message status change events were collected. | status
| postfix_lmtp_delay_seconds | Delay in seconds for a LMTP server to process a message. | status
| postfix_smtp_statuses_total | Total number of times a SMTP server message status change events were collected. | status
| postfix_smtp_delay_seconds | Delay in seconds for a SMTP server to process a message. | status
| postfix_milter_actions_total | Total number of times milter events were collected. | subprogram, action
| postfix_login_failures_total | Total number of times login failure events were collected. | subprogram, method

## Flags

```bash
./postfix_exporter --help
```

* __`collector`:__ Collector type to scrape metrics with. `file` or `journald`.
* __`postfix.instance`:__ Postfix instance name. `postfix` by default.
* __`file.log`:__ Path to a file containing Postfix logs. Example: `/var/log/mail.log`.
* __`journald.path`:__ Path where a systemd journal residing in. A local journal is being used by default.
* __`journald.unit`:__ Postfix systemd service name. `postfix@-.service` by default.
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
