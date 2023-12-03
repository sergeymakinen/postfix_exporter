// Package exporter provides a collector for Postfix stats.
package exporter

import (
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sergeymakinen/postfix_exporter/v2/config"
)

// Collector types.
const (
	CollectorFile = iota
	CollectorJournald
)

const namespace = "postfix"

// ErrUnsupportedCollector results from attempting to use a collector that
// is not currently supported.
var ErrUnsupportedCollector = errors.New("unsupported collector")

var (
	ipAddrPart = `[a-f0-9:.]+`

	psIPAddrPart       = `\[` + ipAddrPart + `]`
	rePsConnect        = regexp.MustCompile(`^CONNECT from ` + psIPAddrPart)
	rePsDNS            = regexp.MustCompile(`^DNSBL rank \d+ for ` + psIPAddrPart)
	rePsPregreet       = regexp.MustCompile(`^PREGREET \d+ after [\d.]+ from ` + psIPAddrPart)
	rePsPass           = regexp.MustCompile(`^PASS (OLD|NEW) ` + psIPAddrPart)
	rePsDisconnect     = regexp.MustCompile(`^DISCONNECT ` + psIPAddrPart)
	rePsHangup         = regexp.MustCompile(`^HANGUP after -?[\d.]+ from ` + psIPAddrPart)
	rePsNoqueueRcpt    = regexp.MustCompile(`^NOQUEUE: reject: RCPT from ` + psIPAddrPart)
	rePsData           = regexp.MustCompile(`^DATA without valid RCPT from ` + psIPAddrPart)
	rePsBdat           = regexp.MustCompile(`^BDAT without valid RCPT from ` + psIPAddrPart)
	rePsCmdTimeLimit   = regexp.MustCompile(`^COMMAND TIME LIMIT from ` + psIPAddrPart)
	rePsCmdLengthLimit = regexp.MustCompile(`^COMMAND LENGTH LIMIT from ` + psIPAddrPart)
	rePsBareNewline    = regexp.MustCompile(`^BARE NEWLINE from ` + psIPAddrPart)
	rePsNonSMTPCmd     = regexp.MustCompile(`^NON-SMTP COMMAND from ` + psIPAddrPart)
	rePsCmpPipelining  = regexp.MustCompile(`^COMMAND PIPELINING from ` + psIPAddrPart)
	rePsCmdCountLimit  = regexp.MustCompile(`^COMMAND COUNT LIMIT from ` + psIPAddrPart)
	rePsNoqueueConnect = regexp.MustCompile(`^NOQUEUE: reject: CONNECT from ` + psIPAddrPart)
	rePsListed         = regexp.MustCompile(`^(DENYLISTED|BLACKLISTED|ALLOWLISTED|WHITELISTED) ` + psIPAddrPart)
	rePsVeto           = regexp.MustCompile(`^(ALLOWLIST|WHITELIST) VETO ` + psIPAddrPart)

	hostnamePart           = `[a-zA-Z0-9-._]+`
	hostnameWithIPAddrPart = hostnamePart + `\[` + ipAddrPart + `]`
	reHostnameNotResolve   = regexp.MustCompile(`^hostname ` + hostnamePart + ` does not resolve to address ` + ipAddrPart)
	reConnect              = regexp.MustCompile(`^connect from ` + hostnameWithIPAddrPart)
	reDisconnect           = regexp.MustCompile(`^disconnect from ` + hostnameWithIPAddrPart)
	reLostConnection       = regexp.MustCompile(`^lost connection after (.+?) from ` + hostnameWithIPAddrPart)
	reMilter               = regexp.MustCompile(`^.+?: milter-([a-z-]+): .+? from ` + hostnameWithIPAddrPart)
	reLoginFailed          = regexp.MustCompile(`^` + hostnameWithIPAddrPart + `: SASL (.+?) authentication failed:`)
	reNoqueueReject        = regexp.MustCompile(`^NOQUEUE: reject: (\w+) from ` + hostnameWithIPAddrPart + `: (\d+) ([\d.]+) (<[^>]+>: )?([^;]+); `)

	reQueueStatus = regexp.MustCompile(`delay=(-?[\d.]+).+status=([a-z-]+) \(.+?\)$`)
	reQmgrStatus  = regexp.MustCompile(`status=([a-z-]+), .+?$`)

	hostReplyPart     = `host ` + hostnameWithIPAddrPart + ` said: (.+) \(in reply to \w+[\w /-]*\)`
	reHostReply       = regexp.MustCompile(hostReplyPart)
	reHostReplyStatus = regexp.MustCompile(`^(\d{3})(.{1,3}(\d\.\d\.\d)|[^ ]+) (.+)$`)
	reSmtpHostReply   = regexp.MustCompile(`^\w+: ` + hostReplyPart + `$`)
)

// Exporter collects Postfix stats from logs and exports them
// using the prometheus metrics package.
type Exporter struct {
	collector io.Closer
	instance  string
	logger    log.Logger
	config    *config.Config

	errors               prometheus.Counter
	foreign              prometheus.Counter
	unsupported          prometheus.Counter
	postscreen           *prometheus.CounterVec
	connects             *prometheus.CounterVec
	disconnects          *prometheus.CounterVec
	lostConnections      *prometheus.CounterVec
	hostnameNotResolved  *prometheus.CounterVec
	lmtpStatuses         *prometheus.CounterVec
	lmtpDelays           *prometheus.SummaryVec
	smtpStatuses         *prometheus.CounterVec
	smtpStatusReplies    *prometheus.CounterVec
	smtpReplies          *prometheus.CounterVec
	smtpDelays           *prometheus.SummaryVec
	milter               *prometheus.CounterVec
	loginFailed          *prometheus.CounterVec
	qmgrStatuses         *prometheus.CounterVec
	logs                 *prometheus.CounterVec
	noqueueRejectReplies *prometheus.CounterVec
}

// Close stops collecting new logs.
func (e *Exporter) Close() error {
	return e.collector.Close()
}

// Describe describes all the metrics exported by the Postfix exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.errors.Describe(ch)
	e.foreign.Describe(ch)
	e.unsupported.Describe(ch)
	e.postscreen.Describe(ch)
	e.connects.Describe(ch)
	e.disconnects.Describe(ch)
	e.lostConnections.Describe(ch)
	e.hostnameNotResolved.Describe(ch)
	e.lmtpStatuses.Describe(ch)
	e.lmtpDelays.Describe(ch)
	e.smtpStatuses.Describe(ch)
	e.smtpStatusReplies.Describe(ch)
	e.smtpReplies.Describe(ch)
	e.smtpDelays.Describe(ch)
	e.milter.Describe(ch)
	e.loginFailed.Describe(ch)
	e.qmgrStatuses.Describe(ch)
	e.logs.Describe(ch)
	e.noqueueRejectReplies.Describe(ch)
}

// Collect delivers collected Postfix statistics as Prometheus metrics.
// It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.errors.Collect(ch)
	e.foreign.Collect(ch)
	e.unsupported.Collect(ch)
	e.postscreen.Collect(ch)
	e.connects.Collect(ch)
	e.disconnects.Collect(ch)
	e.lostConnections.Collect(ch)
	e.hostnameNotResolved.Collect(ch)
	e.lmtpStatuses.Collect(ch)
	e.lmtpDelays.Collect(ch)
	e.smtpStatuses.Collect(ch)
	e.smtpStatusReplies.Collect(ch)
	e.smtpReplies.Collect(ch)
	e.smtpDelays.Collect(ch)
	e.milter.Collect(ch)
	e.loginFailed.Collect(ch)
	e.qmgrStatuses.Collect(ch)
	e.logs.Collect(ch)
	e.noqueueRejectReplies.Collect(ch)
}

func (e *Exporter) scrape(r record, err error) {
	if err != nil {
		e.errors.Inc()
		level.Warn(e.logger).Log("msg", "Error parsing log record", "err", err, "record", r)
		return
	}
	if r.Program != e.instance {
		e.foreign.Inc()
		level.Debug(e.logger).Log("msg", "Foreign log record", "record", r)
		return
	}
	e.logs.WithLabelValues(r.Subprogram, string(r.Severity)).Inc()
	found := true
	if r.Subprogram == "postscreen" {
		if matches := rePsConnect.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("CONNECT").Inc()
		} else if matches := rePsDNS.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("DNSBL").Inc()
		} else if matches := rePsPregreet.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("PREGREET").Inc()
		} else if matches := rePsPass.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("PASS " + matches[1]).Inc()
		} else if matches := rePsDisconnect.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("DISCONNECT").Inc()
		} else if matches := rePsHangup.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("HANGUP").Inc()
		} else if matches := rePsNoqueueRcpt.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("NOQUEUE: RCPT").Inc()
		} else if matches := rePsData.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("DATA").Inc()
		} else if matches := rePsBdat.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("BDAT").Inc()
		} else if matches := rePsCmdTimeLimit.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("COMMAND TIME LIMIT").Inc()
		} else if matches := rePsCmdLengthLimit.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("COMMAND LENGTH LIMIT").Inc()
		} else if matches := rePsBareNewline.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("BARE NEWLINE").Inc()
		} else if matches := rePsNonSMTPCmd.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("NON-SMTP COMMAND").Inc()
		} else if matches := rePsCmpPipelining.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("COMMAND PIPELINING").Inc()
		} else if matches := rePsCmdCountLimit.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("COMMAND COUNT LIMIT").Inc()
		} else if matches := rePsNoqueueConnect.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues("NOQUEUE: CONNECT").Inc()
		} else if matches := rePsListed.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues(matches[1]).Inc()
		} else if matches := rePsVeto.FindStringSubmatch(r.Text); matches != nil {
			e.postscreen.WithLabelValues(matches[1] + " VETO").Inc()
		} else {
			found = false
		}
	} else if r.Subprogram == "smtpd" || strings.HasSuffix(r.Subprogram, "/smtpd") {
		if strings.HasPrefix(r.Text, "NOQUEUE: reject:") {
			if matches := reNoqueueReject.FindStringSubmatch(r.Text); matches != nil {
				if cfg, m := findSubmatch(e.config.NoqueueRejectReplies, func(cfg config.NoqueueRejectReplyConfig) []int {
					return cfg.Regexp.FindStringSubmatchIndex(matches[5])
				}); m != nil {
					text := string(cfg.Regexp.ExpandString(nil, cfg.Text, matches[5], m))
					e.noqueueRejectReplies.WithLabelValues(r.Subprogram, matches[1], matches[2], matches[3], text).Inc()
				}
			} else {
				found = false
			}
		} else if matches := reConnect.FindStringSubmatch(r.Text); matches != nil {
			e.connects.WithLabelValues(r.Subprogram).Inc()
		} else if matches := reDisconnect.FindStringSubmatch(r.Text); matches != nil {
			e.disconnects.WithLabelValues(r.Subprogram).Inc()
		} else if matches := reLostConnection.FindStringSubmatch(r.Text); matches != nil {
			e.lostConnections.WithLabelValues(r.Subprogram).Inc()
		} else if matches := reHostnameNotResolve.FindStringSubmatch(r.Text); matches != nil {
			e.hostnameNotResolved.WithLabelValues(r.Subprogram).Inc()
		} else if matches := reMilter.FindStringSubmatch(r.Text); matches != nil {
			e.milter.WithLabelValues(r.Subprogram, matches[1]).Inc()
		} else if matches := reLoginFailed.FindStringSubmatch(r.Text); matches != nil {
			e.loginFailed.WithLabelValues(r.Subprogram, matches[1]).Inc()
		} else {
			found = false
		}
	} else if r.Subprogram == "smtp" {
		if matches := reQueueStatus.FindStringSubmatch(r.Text); matches != nil {
			e.smtpStatuses.WithLabelValues(matches[2]).Inc()
			f, _ := strconv.ParseFloat(matches[1], 64)
			e.smtpDelays.WithLabelValues(matches[2]).Observe(f)
			if m := reHostReply.FindStringSubmatch(r.Text); m != nil {
				reply, err := parseHostReply(m[1])
				if err == nil {
					if cfg, m := findSubmatch(e.config.HostReplies, func(cfg config.HostReplyConfig) []int {
						if cfg.Type != config.HostReplyAny && cfg.Type != config.HostReplyQueueStatus {
							return nil
						}
						return cfg.Regexp.FindStringSubmatchIndex(reply.Text)
					}); m != nil {
						text := string(cfg.Regexp.ExpandString(nil, cfg.Text, reply.Text, m))
						e.smtpStatusReplies.WithLabelValues(matches[2], reply.Code, reply.EnhancedCode, text).Inc()
					}
				} else {
					level.Warn(e.logger).Log("msg", "Error parsing host reply", "err", err, "record", r)
				}
			}
		} else if matches := reSmtpHostReply.FindStringSubmatch(r.Text); matches != nil {
			reply, err := parseHostReply(matches[1])
			if err == nil {
				if cfg, m := findSubmatch(e.config.HostReplies, func(cfg config.HostReplyConfig) []int {
					if cfg.Type != config.HostReplyAny && cfg.Type != config.HostReplyOther {
						return nil
					}
					return cfg.Regexp.FindStringSubmatchIndex(reply.Text)
				}); m != nil {
					text := string(cfg.Regexp.ExpandString(nil, cfg.Text, reply.Text, m))
					e.smtpReplies.WithLabelValues(reply.Code, reply.EnhancedCode, text).Inc()
				}
			} else {
				level.Warn(e.logger).Log("msg", "Error parsing host reply", "err", err, "record", r)
			}
		} else {
			found = false
		}
	} else if r.Subprogram == "lmtp" {
		if matches := reQueueStatus.FindStringSubmatch(r.Text); matches != nil {
			e.lmtpStatuses.WithLabelValues(matches[2]).Inc()
			f, _ := strconv.ParseFloat(matches[1], 64)
			e.lmtpDelays.WithLabelValues(matches[2]).Observe(f)
		} else {
			found = false
		}
	} else if r.Subprogram == "cleanup" {
		if matches := reMilter.FindStringSubmatch(r.Text); matches != nil {
			e.milter.WithLabelValues(r.Subprogram, matches[1]).Inc()
		} else {
			found = false
		}
	} else if r.Subprogram == "qmgr" {
		if matches := reQmgrStatus.FindStringSubmatch(r.Text); matches != nil {
			e.qmgrStatuses.WithLabelValues(matches[1]).Inc()
		} else {
			found = false
		}
	} else {
		found = false
	}
	if found {
		return
	}
	e.unsupported.Inc()
	level.Debug(e.logger).Log("msg", "Unsupported log record", "record", r)
}

// New returns an initialized exporter.
func New(collectorType int, instance, logPath, journaldPath, journaldUnit string, config *config.Config, logger log.Logger) (*Exporter, error) {
	quantiles := map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
	e := &Exporter{
		instance: instance,
		logger:   logger,
		config:   config,

		errors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "errors_total",
			Help:      "Total number of log records parsing resulted in an error.",
		}),
		foreign: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "foreign_total",
			Help:      "Total number of foreign log records.",
		}),
		unsupported: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "unsupported_total",
			Help:      "Total number of unsupported log records.",
		}),
		postscreen: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "postscreen_actions_total",
			Help:      "Total number of times postscreen events were collected.",
		}, []string{"action"}),
		connects: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "connects_total",
			Help:      "Total number of times connect events were collected.",
		}, []string{"subprogram"}),
		disconnects: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "disconnects_total",
			Help:      "Total number of times disconnect events were collected.",
		}, []string{"subprogram"}),
		lostConnections: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "lost_connections_total",
			Help:      "Total number of times lost connection events were collected.",
		}, []string{"subprogram"}),
		hostnameNotResolved: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "not_resolved_hostnames_total",
			Help:      "Total number of times not resolved hostname events were collected.",
		}, []string{"subprogram"}),
		lmtpStatuses: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "lmtp_statuses_total",
			Help:      "Total number of times LMTP server message status change events were collected.",
		}, []string{"status"}),
		lmtpDelays: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  namespace,
			Name:       "lmtp_delay_seconds",
			Help:       "Delay in seconds for a LMTP server to process a message.",
			Objectives: quantiles,
		}, []string{"status"}),
		smtpStatuses: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "smtp_statuses_total",
			Help:      "Total number of times SMTP server message status change events were collected.",
		}, []string{"status"}),
		smtpStatusReplies: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "smtp_status_replies_total",
			Help:      "Total number of times SMTP server message status change event replies were collected.",
		}, []string{"status", "code", "enhanced_code", "text"}),
		smtpReplies: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "smtp_replies_total",
			Help:      "Total number of times SMTP server replies were collected.",
		}, []string{"code", "enhanced_code", "text"}),
		smtpDelays: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  namespace,
			Name:       "smtp_delay_seconds",
			Help:       "Delay in seconds for a SMTP server to process a message.",
			Objectives: quantiles,
		}, []string{"status"}),
		milter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "milter_actions_total",
			Help:      "Total number of times milter events were collected.",
		}, []string{"subprogram", "action"}),
		loginFailed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "login_failures_total",
			Help:      "Total number of times login failure events were collected.",
		}, []string{"subprogram", "method"}),
		qmgrStatuses: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "qmgr_statuses_total",
			Help:      "Total number of times Postfix queue manager message status change events were collected.",
		}, []string{"status"}),
		logs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "logs_total",
			Help:      "Total number of log records processed.",
		}, []string{"subprogram", "severity"}),
		noqueueRejectReplies: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "noqueue_reject_replies_total",
			Help:      "Total number of times NOQUEUE: reject event replies were collected.",
		}, []string{"subprogram", "command", "code", "enhanced_code", "text"}),
	}
	var err error
	switch collectorType {
	case CollectorFile:
		e.collector, err = collectFromFile(logPath, e.scrape)
	case CollectorJournald:
		e.collector, err = collectFromJournald(journaldPath, journaldUnit, e.scrape)
	default:
		err = errors.New("unknown collector type " + strconv.Itoa(collectorType))
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

type hostReply struct {
	Code         string
	EnhancedCode string
	Text         string
}

func parseHostReply(s string) (*hostReply, error) {
	matches := reHostReplyStatus.FindStringSubmatch(s)
	if matches == nil {
		return nil, errors.New("failed to find host reply in " + strconv.Quote(s))
	}
	reply := &hostReply{
		Code: matches[1],
		Text: matches[len(matches)-1],
	}
	if len(matches) == 5 {
		reply.EnhancedCode = matches[3]
	}
	return reply, nil
}

func findSubmatch[S ~[]E, E any](slice S, f func(E) []int) (E, []int) {
	var zero E
	for _, e := range slice {
		if m := f(e); m != nil {
			return e, m
		}
	}
	return zero, nil
}
