package exporter

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

var testMetrics = []string{
	"postfix_unsupported_total",
	"postfix_postscreen_actions_total",
	"postfix_connects_total",
	"postfix_disconnects_total",
	"postfix_lost_connections_total",
	"postfix_not_resolved_hostnames_total",
	"postfix_lmtp_statuses_total",
	"postfix_lmtp_delay_seconds",
	"postfix_smtp_statuses_total",
	"postfix_smtp_delay_seconds",
	"postfix_milter_actions_total",
	"postfix_login_failures_total",
	"postfix_qmgr_statuses_total",
	"postfix_logs_total",
}

func TestExporter_Collect_File(t *testing.T) {
	out, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		out.Close()
		os.Remove(out.Name())
	}()
	exporter, err := New(CollectorFile, "postfix", out.Name(), "", "", log.NewNopLogger())
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
		b, err := buf.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err = out.Write(b); err != nil {
			t.Fatal(err)
		}
		if err = out.Sync(); err != nil {
			t.Fatal(err)
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
