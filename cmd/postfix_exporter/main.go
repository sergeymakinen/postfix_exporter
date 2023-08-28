package main

import (
	"net/http"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
	"github.com/sergeymakinen/postfix_exporter/exporter"
)

func main() {
	var (
		collector    = kingpin.Flag("collector", "Collector type to scrape metrics with. One of: [file, journald]").Default("file").Enum("file", "journald")
		instance     = kingpin.Flag("postfix.instance", "Postfix instance name.").Default("postfix").String()
		logPath      = kingpin.Flag("file.log", "Path to a file containing Postfix logs.").Default("/var/log/mail.log").String()
		journaldPath = kingpin.Flag("journald.path", "Path where a systemd journal residing in.").Default("").String()
		journaldUnit = kingpin.Flag("journald.unit", "Postfix systemd service name.").Default("postfix@-.service").String()
		toolkitFlags = webflag.AddFlags(kingpin.CommandLine, ":9907")
		metricsPath  = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	)
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Version(version.Print("postfix_exporter"))
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	level.Info(logger).Log("msg", "Starting postfix_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "context", version.BuildContext())

	prometheus.MustRegister(version.NewCollector("postfix_exporter"))
	collectorType := exporter.CollectorFile
	if *collector == "journald" {
		collectorType = exporter.CollectorJournald
	}
	exporter, err := exporter.New(collectorType, *instance, *logPath, *journaldPath, *journaldUnit, logger)
	if err != nil {
		level.Error(logger).Log("msg", "Error creating the exporter", "err", err)
		os.Exit(1)
	}
	defer exporter.Close()
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Postfix Exporter</title></head>
             <body>
             <h1>Postfix Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	srv := &http.Server{}
	if err := web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
		level.Error(logger).Log("msg", "Error running HTTP server", "err", err)
		os.Exit(1)
	}
}
