package exporter

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-systemd/v22/journal"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestExporter_Collect_Journald(t *testing.T) {
	exporter, err := New(CollectorJournald, "postfix", "", "", "", log.NewNopLogger())
	if err == ErrUnsupportedCollector {
		t.Skip(err)
	}
	if err != nil {
		t.Fatalf("New() = _, %v; want nil", err)
	}
	in, err := os.Open("testdata/mail.log")
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	buf := bufio.NewReader(in)
	for {
		s, err := buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		s = strings.TrimSuffix(s, "\n")
		if r, err := parseRecord(s); err == nil {
			id := r.Program
			if r.Subprogram != "" {
				id += "/" + r.Subprogram
			}
			err = journal.Send(r.Text, journal.PriInfo, map[string]string{
				"SYSLOG_IDENTIFIER": id,
				"SYSLOG_TIMESTAMP":  r.Time.Format(timeFormat) + " ",
			})
			if err != nil {
				t.Fatal(err)
			}
		} else {
			if err = journal.Send(s, journal.PriInfo, nil); err != nil {
				t.Fatal(err)
			}
		}
	}
	time.Sleep(5 * time.Second)
	b, err := ioutil.ReadFile("testdata/metrics.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := testutil.CollectAndCompare(exporter, bytes.NewReader(b), testMetrics...); err != nil {
		t.Errorf("testutil.CollectAndCompare() = %v; want nil", err)
	}
}
