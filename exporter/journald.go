//go:build linux && cgo

package exporter

import (
	"cmp"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
)

// Journald collects Postfix logs from systemd journal.
type Journald struct {
	Path  string
	Unit  string
	Since time.Duration
	Test  bool

	r    *sdjournal.JournalReader
	done chan time.Time
}

func (j *Journald) Collect(ch chan<- result) error {
	if j.Test {
		return j.read(ch)
	}
	return j.start(ch)
}

func (j *Journald) open() (*sdjournal.JournalReader, error) {
	var m []sdjournal.Match
	if j.Unit != "" {
		m = append(m, sdjournal.Match{
			Field: sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT,
			Value: j.Unit,
		})
	}
	return sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
		Since:     cmp.Or(j.Since, -1),
		Matches:   m,
		Path:      j.Path,
		Formatter: formatJournald,
	})
}

func (j *Journald) start(ch chan<- result) error {
	r, err := j.open()
	if err != nil {
		return err
	}
	j.r = r
	j.done = make(chan time.Time)
	go j.r.Follow(j.done, writerFunc(func(p []byte) (n int, err error) {
		var res result
		res.rec, res.err = parseRecord(string(p))
		ch <- res
		return len(p), nil
	}))
	return nil
}

func (j *Journald) read(ch chan<- result) error {
	r, err := j.open()
	if err != nil {
		return err
	}
	defer r.Close()
	buf := make([]byte, 64<<10)
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if n > 0 {
			var res result
			res.rec, res.err = parseRecord(string(buf[:n]))
			ch <- res
		}
	}
	return nil
}

func (j *Journald) Close() error {
	if j.r == nil {
		return nil
	}
	j.done <- time.Now()
	close(j.done)
	return j.r.Close()
}

type writerFunc func(p []byte) (n int, err error)

func (f writerFunc) Write(p []byte) (n int, err error) {
	return f(p)
}

func formatJournald(entry *sdjournal.JournalEntry) (string, error) {
	severity := ""
	switch entry.Fields[sdjournal.SD_JOURNAL_FIELD_PRIORITY] {
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
		journaldField(entry, sdjournal.SD_JOURNAL_FIELD_HOSTNAME),
		journaldField(entry, sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER),
		journaldField(entry, sdjournal.SD_JOURNAL_FIELD_PID),
		severity,
		journaldField(entry, sdjournal.SD_JOURNAL_FIELD_MESSAGE),
	), nil
}

func journaldField(entry *sdjournal.JournalEntry, key string) string {
	if s, ok := entry.Fields[key]; ok {
		return s
	}
	return "%unknown " + key + "%"
}
