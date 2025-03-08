package main

import (
	"net/http"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
	"github.com/sergeymakinen/postfix_exporter/v2/config"
	"github.com/sergeymakinen/postfix_exporter/v2/exporter"
)

func main() {
	var (
		configFile    = kingpin.Flag("config.file", "Postfix Exporter configuration file.").String()
		configCheck   = kingpin.Flag("config.check", "If true, validate the config file and then exit.").Default().Bool()
		collectorType = kingpin.Flag("collector", "Collector type to scrape metrics with. One of: [file, journald]").Default("file").Enum("file", "journald")
		instance      = kingpin.Flag("postfix.instance", "Postfix instance name.").Default("postfix").String()
		logPath       = kingpin.Flag("file.log", "Path to a file containing Postfix logs.").Default("/var/log/mail.log").String()
		journaldPath  = kingpin.Flag("journald.path", "Path where a systemd journal residing in.").Default("").String()
		journaldUnit  = kingpin.Flag("journald.unit", "Postfix systemd service name.").Default("postfix@-.service").String()
		journaldSince = kingpin.Flag("journald.since", "Time since to read from a systemd journal.").Default("0s").Duration()
		test          = kingpin.Flag("test", "If true, read logs, print metrics and exit.").Default("false").Bool()
		toolkitFlags  = webflag.AddFlags(kingpin.CommandLine, ":9907")
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
	)
	promslogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promslogConfig)
	kingpin.Version(version.Print("postfix_exporter"))
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promslog.New(promslogConfig)

	logger.Info("Starting postfix_exporter", "version", version.Info())
	logger.Info("Build context", "context", version.BuildContext())

	var (
		cfg *config.Config
		err error
	)
	if *configFile != "" {
		cfg, err = config.Load(*configFile)
		if err != nil {
			logger.Error("Error loading config", "err", err)
			os.Exit(1)
		}
		if *configCheck {
			logger.Info("Config file is ok, exiting...")
			return
		}
		logger.Info("Loaded config file")
	}

	prometheus.MustRegister(versioncollector.NewCollector("postfix_exporter"))
	var collector exporter.Collector
	switch *collectorType {
	case "file":
		collector = &exporter.File{
			Path: *logPath,
			Test: *test,
		}
	case "journald":
		collector = &exporter.Journald{
			Path:  *journaldPath,
			Unit:  *journaldUnit,
			Since: *journaldSince,
			Test:  *test,
		}
	}
	exporter, err := exporter.New(collector, *instance, cfg, logger)
	if err != nil {
		logger.Error("Error creating the exporter", "err", err)
		os.Exit(1)
	}
	defer exporter.Close()
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
	if *metricsPath != "/" {
		landingConfig := web.LandingConfig{
			Name:        "Postfix Exporter",
			Description: "Prometheus Exporter for Postfix",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{
					Address: *metricsPath,
					Text:    "Metrics",
				},
			},
		}
		landingPage, err := web.NewLandingPage(landingConfig)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
		http.Handle("/", landingPage)
	}

	srv := &http.Server{}
	if err := web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
		logger.Error("Error running HTTP server", "err", err)
		os.Exit(1)
	}
}
