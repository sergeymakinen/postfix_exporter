// +build !linux !cgo

package exporter

import "io"

func collectFromJournald(_, _ string, _ func(r record, err error)) (io.Closer, error) {
	return nil, ErrUnsupportedCollector
}
