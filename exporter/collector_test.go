package exporter

import "testing"

func TestParseRecord_ErrorKeepsLine(t *testing.T) {
	for _, line := range []string{
		"",
		"garbage",
		"%unknown SYSLOG_TIMESTAMP% mail postfix[1]: postfix/postlog: stopping the Postfix mail system",
		"Oct  5 11:52:39 mail postfix/smtpd",
		"Oct  5 11:52:39 mail postfix/smtpd[abc]: connect from localhost[127.0.0.1]",
	} {
		r, err := parseRecord(line)
		if err == nil {
			t.Errorf("parseRecord(%q) = _, nil; want non-nil", line)
		}
		if got := r.String(); got != line {
			t.Errorf("parseRecord(%q).String() = %q; want %q", line, got, line)
		}
	}
}
