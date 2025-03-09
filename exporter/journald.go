//go:build linux && cgo

package exporter

import (
	"cmp"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
)

// Journald collects Postfix logs from systemd journal.
type Journald struct {
	Path  string
	Unit  string
	Since time.Duration
	Test  bool

	r      *sdjournal.JournalReader
	closed bool
	done   chan time.Time
	wg     sync.WaitGroup
}

func (j *Journald) Collect(ch chan<- result) error {
	j.done = make(chan time.Time)
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
	d := cmp.Or(j.Since, -1)
	if d > 0 {
		d = -d
	}
	return sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
		Since:     d,
		Matches:   m,
		Path:      j.Path,
		Formatter: formatJournald,
	})
}

func (j *Journald) start(ch chan<- result) error {
	var err error
	j.r, err = j.open()
	if err != nil {
		return err
	}
	j.wg.Add(1)
	go func() {
		defer j.wg.Done()
		j.r.Follow(j.done, writerFunc(func(p []byte) (n int, err error) {
			var res result
			res.rec, res.err = parseRecord(string(p))
			select {
			case ch <- res:
			case <-j.done:
				return
			}
			return len(p), nil
		}))
	}()
	return nil
}

func (j *Journald) read(ch chan<- result) error {
	r, err := j.open()
	if err != nil {
		return err
	}
	j.wg.Add(1)
	go func() {
		defer j.wg.Done()
		defer r.Close()
		buf := make([]byte, 64<<10)
		for {
			if j.closed {
				return
			}
			n, err := r.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				return
			}
			if n > 0 {
				var res result
				res.rec, res.err = parseRecord(string(buf[:n]))
				select {
				case ch <- res:
				case <-j.done:
					return
				}
			}
		}
	}()
	return nil
}

func (j *Journald) Wait() {
	j.wg.Wait()
}

func (j *Journald) Close() error {
	j.closed = true
	close(j.done)
	var err error
	if j.r != nil {
		err = j.r.Close()
	}
	return err
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
