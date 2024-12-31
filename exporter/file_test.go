package exporter

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/promslog"
	"github.com/sergeymakinen/postfix_exporter/v2/config"
)

var testMetrics = []string{
	"postfix_unsupported_total",
	"postfix_postscreen_actions_total",
	"postfix_connects_total",
	"postfix_disconnects_total",
	"postfix_lost_connections_total",
	"postfix_not_resolved_hostnames_total",
	"postfix_statuses_total",
	"postfix_delay_seconds",
	"postfix_status_replies_total",
	"postfix_smtp_replies_total",
	"postfix_milter_actions_total",
	"postfix_login_failures_total",
	"postfix_qmgr_statuses_total",
	"postfix_logs_total",
	"postfix_noqueue_reject_replies_total",
}

var tests = map[string]struct {
	Cfg     string
	Metrics string
}{
	"with config": {
		Cfg:     "testdata/postfix.yml",
		Metrics: "testdata/metrics-with-config.txt",
	},
	"without config": {
		Cfg:     "",
		Metrics: "testdata/metrics-without-config.txt",
	},
}

func TestExporter_Collect_File(t *testing.T) {
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			out, err := os.CreateTemp("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				out.Close()
				os.Remove(out.Name())
			}()
			var cfg *config.Config
			if test.Cfg != "" {
				cfg, err = config.Load(test.Cfg)
				if err != nil {
					t.Fatal(err)
				}
			}
			exporter, err := New(CollectorFile, "postfix", out.Name(), "", "", cfg, promslog.NewNopLogger())
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
			b, err := os.ReadFile(test.Metrics)
			if err != nil {
				t.Fatal(err)
			}
			if err := testutil.CollectAndCompare(exporter, bytes.NewReader(b), testMetrics...); err != nil {
				t.Errorf("testutil.CollectAndCompare() = %v; want nil", err)
			}
		})
	}

}
