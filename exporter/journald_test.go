package exporter

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-systemd/v22/journal"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/promslog"
	"github.com/sergeymakinen/postfix_exporter/v2/config"
)

func TestExporter_Collect_Journald(t *testing.T) {
	cfg, err := config.Load("testdata/postfix.yml")
	if err != nil {
		t.Fatal(err)
	}
	exporter, err := New(CollectorJournald, "postfix", "", "", "", cfg, promslog.NewNopLogger())
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
			var severity string
			if r.Severity != severityInfo {
				severity = string(r.Severity) + ": "
			}
			err = journal.Send(severity+r.Text, journal.PriInfo, map[string]string{
				"SYSLOG_IDENTIFIER": id,
				"SYSLOG_TIMESTAMP":  r.Time.Format(bsdFormat) + " ",
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	time.Sleep(5 * time.Second)
	b, err := os.ReadFile("testdata/metrics.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := testutil.CollectAndCompare(exporter, bytes.NewReader(b), testMetrics...); err != nil {
		t.Errorf("testutil.CollectAndCompare() = %v; want nil", err)
	}
}
