package slog

import (
	"bytes"
	"testing"
)

func TestSetup(t *testing.T) {
	for _, c := range []struct {
		want error
	}{
		{nil},
		{errInitialized},
	} {
		got := Setup("local")
		if got != c.want {
			t.Errorf("got=%v, want=%v", got, c.want)
		}
	}
}

func TestEnabled(t *testing.T) {
	severity = SeverityInfo

	for _, c := range []struct {
		in   Severity
		want bool
	}{
		{SeverityDebug, false},
		{SeverityInfo, true},
		{SeverityWarning, true},
		{SeverityError, true},
	} {
		got := Enabled(c.in)
		if got != c.want {
			t.Errorf("got=%v, want=%v", got, c.want)
		}
	}
}

func TestString(t *testing.T) {
	for _, c := range []struct {
		in   Severity
		want string
	}{
		{SeverityDebug, "DEBUG"},
		{SeverityInfo, "INFO"},
		{SeverityWarning, "WARNING"},
		{SeverityError, "ERROR"},
	} {
		got := c.in.String()
		if got != c.want {
			t.Errorf("got=%v, want=%v", got, c.want)
		}
	}
}

func TestLog(t *testing.T) {
	for _, c := range []struct {
		in   Entry
		want string
	}{
		{
			Entry{Severity: SeverityInfo.String(), Message: "hoge"},
			`{"severity":"INFO","message":"hoge"}` + "\n",
		},
		{
			Entry{Severity: SeverityInfo.String(), Trace: "trace", SpanID: "span-id", SourceLocation: &SourceLocation{File: "log_test.go", Line: 65, Function: "main.TestLog"}, Message: "fuga"},
			`{"severity":"INFO","logging.googleapis.com/trace":"trace","logging.googleapis.com/spanId":"span-id","logging.googleapis.com/sourceLocation":{"file":"log_test.go","line":65,"function":"main.TestLog"},"message":"fuga"}` + "\n",
		},
	} {
		var buf bytes.Buffer
		write(&buf, c.in)
		got := buf.String()
		if got != c.want {
			t.Errorf("got=%v, want=%v", got, c.want)
		}
	}
}
