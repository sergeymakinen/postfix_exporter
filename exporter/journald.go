//go:build linux && cgo

package exporter

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
)

type journaldCollector struct {
	r    *sdjournal.JournalReader
	done chan time.Time
	f    func(r record, err error)
}

func collectFromJournald(path, unit string, f func(r record, err error)) (io.Closer, error) {
	var m []sdjournal.Match
	if unit != "" {
		m = append(m, sdjournal.Match{
			Field: sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT,
			Value: unit,
		})
	}
	r, err := sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
		Since:     -1,
		Matches:   m,
		Path:      path,
		Formatter: formatJournald,
	})
	if err != nil {
		return nil, err
	}
	c := &journaldCollector{
		r:    r,
		done: make(chan time.Time),
		f:    f,
	}
	go r.Follow(c.done, c)
	return c, nil
}

func (c *journaldCollector) Close() error {
	c.done <- time.Now()
	close(c.done)
	return c.r.Close()
}

func (c *journaldCollector) Write(p []byte) (n int, err error) {
	c.f(parseRecord(string(p)))
	return len(p), nil
}

func formatJournald(entry *sdjournal.JournalEntry) (string, error) {
	severity := ""
	switch entry.Fields["PRIORITY"] {
	case "4":
		severity = string(severityWarning)
	case "3":
		severity = string(severityError)
	case "1", "2":
		severity = string(severityFatal)
	case "0":
		severity = string(severityPanic)
	}
	if severity != "" {
		severity = ": " + severity
	}
	return fmt.Sprintf(
		"%s %s %s[%s]%s: %s",
		strings.TrimSuffix(journaldField(entry, "SYSLOG_TIMESTAMP"), " "),
		journaldField(entry, "_HOSTNAME"),
		journaldField(entry, "SYSLOG_IDENTIFIER"),
		journaldField(entry, "_PID"),
		severity,
		journaldField(entry, "MESSAGE"),
	), nil
}

func journaldField(entry *sdjournal.JournalEntry, key string) string {
	if s, ok := entry.Fields[key]; ok {
		return s
	}
	return "%unknown " + key + "%"
}
