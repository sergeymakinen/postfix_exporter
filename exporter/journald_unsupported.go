//go:build !linux || !cgo

package exporter

import "time"

// Journald collects Postfix logs from journald.
type Journald struct {
	Path  string
	Unit  string
	Since time.Duration
	Test  bool
}

func (*Journald) Collect(chan<- result) error { return ErrUnsupportedCollector }

func (*Journald) Close() error { return nil }
