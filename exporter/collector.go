package exporter

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

type severity string

const (
	severityInfo    severity = "info"
	severityWarning severity = "warning"
	severityError   severity = "error"
	severityFatal   severity = "fatal"
	severityPanic   severity = "panic"
)

const bsdFormat = "Jan  2 15:04:05"

type record struct {
	Time       time.Time
	Hostname   string
	Program    string
	Subprogram string
	PID        int64
	Severity   severity
	Text       string

	line string
}

func (r record) String() string { return r.line }

func parseRecord(line string) (record, error) {
	s := line
	readUntil := func(substr string, n int) (string, error) {
		i := 0
		ss := s
		for ; n > 0; n-- {
			j := strings.Index(s, substr)
			if j == -1 {
				return "", errors.New("missing " + strconv.Quote(substr) + " in " + strconv.Quote(line))
			}
			if j == 0 {
				// Skip consecutive substrs.
				n++
			}
			s, i = s[j+len(substr):], i+j+len(substr)
		}
		return ss[:i-len(substr)], nil
	}
	ss, err := readUntil(" ", 1)
	if err != nil {
		return record{}, err
	}
	r := record{
		line: line,

		Severity: severityInfo,
	}
	if strings.Contains(ss, ":") {
		// RFC3339 timestamp.
		r.Time, err = time.Parse(time.RFC3339Nano, ss)
		if err != nil {
			return record{}, err
		}
	} else {
		// Classic BSD timestamp.
		ss2, err := readUntil(" ", 2)
		if err != nil {
			return record{}, err
		}
		ss += " " + ss2
		r.Time, err = time.Parse(bsdFormat, ss)
		if err != nil {
			return record{}, err
		}
	}
	r.Hostname, err = readUntil(" ", 1)
	if err != nil {
		return record{}, err
	}
	r.Program, err = readUntil("[", 1)
	if err != nil {
		return record{}, err
	}
	if parts := strings.SplitN(r.Program, "/", 2); len(parts) == 2 {
		r.Program, r.Subprogram = parts[0], parts[1]
	}
	ss, err = readUntil("]: ", 1)
	if err != nil {
		return record{}, err
	}
	r.PID, err = strconv.ParseInt(ss, 10, 64)
	if err != nil {
		return record{}, err
	}
	ss, err = readUntil(": ", 1)
	if err == nil {
		switch severity := severity(ss); severity {
		case severityWarning, severityError, severityFatal, severityPanic:
			r.Severity = severity
		default:
			// Unread ss which is not a severity.
			s = ss + ": " + s
		}
	}
	r.Text = s
	return r, nil
}
