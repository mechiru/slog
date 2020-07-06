package slog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"sync"

	"go.opentelemetry.io/otel/api/trace"
)

var (
	state uint32 = 0

	stateUninitialized uint32 = 0
	stateInitialized   uint32 = 1

	mu sync.Mutex

	projectID string
	logLevel  Severity = SeverityDebug
)

// An Option is an option for a slog package.
type Option func()

// WithLogLevel returns an Option that specifies a log level.
// Default log level is SeverityDebug.
func WithLogLevel(lvl Severity) Option {
	return func() { logLevel = lvl }
}

var errInitialized = errors.New("slog is already initialized")

// Setup is setup function for slog package.
func Setup(traceProjectID string, opts ...Option) error {
	mu.Lock()
	defer mu.Unlock()

	if state != stateUninitialized {
		return errInitialized
	}

	projectID = traceProjectID
	for _, opt := range opts {
		opt()
	}
	state = stateInitialized
	return nil
}

// Enabled decides whether a given logging level is enabled when logging a message.
func Enabled(lvl Severity) bool { return lvl >= logLevel }

// Severity is implementation of https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#LogSeverity.
type Severity uint32

const (
	// SeverityDefault represents log entry has no assigned severity level.
	SeverityDefault Severity = 0
	// SeverityDebug represents debug or trace information.
	SeverityDebug Severity = 1
	// SeverityInfo represents routine information, such as ongoing status or performance.
	SeverityInfo Severity = 2
	// SeverityNotice represents normal but significant events, such as start up, shut down, or a configuration change.
	SeverityNotice Severity = 3
	// SeverityWarning represents warning events might cause problems.
	SeverityWarning Severity = 4
	// SeverityError represents error events are likely to cause problems.
	SeverityError Severity = 5
	// SeverityCritical represents critical events cause more severe problems or outages.
	SeverityCritical Severity = 6
	// SeverityAlert represents a person must take an action immediately.
	SeverityAlert Severity = 7
	// SeverityEmergency represents one or more systems are unusable.
	SeverityEmergency Severity = 8
)

var severities = []string{"DEFAULT", "DEBUG", "INFO", "NOTICE", "WARNING", "ERROR", "CRITICAL", "ALERT", "EMERGENCY"}

func (s Severity) String() string { return severities[int(s)] }

// Entry is a log entry.
// See https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry.
type Entry struct {
	// A severity as string.
	Severity string `json:"severity"`
	// Optional. Resource name of the trace associated with the log entry, if any.
	// If it contains a relative resource name, the name is assumed to be relative to //tracing.googleapis.com.
	// Example: projects/my-projectid/traces/06796866738c859f2f19b7cfb3214824
	Trace string `json:"logging.googleapis.com/trace,omitempty"`
	// Optional. The span ID within the trace associated with the log entry.
	// For Trace spans, this is the same format that the Trace API v2 uses: a 16-character hexadecimal encoding
	// of an 8-byte array, such as 000000000000004a.
	SpanID string `json:"logging.googleapis.com/spanId,omitempty"`
	// Optional. Source code location information associated with the log entry, if any.
	SourceLocation *SourceLocation `json:"logging.googleapis.com/sourceLocation,omitempty"`
	// The log entry payload, represented as a Unicode string (UTF-8).
	Message string `json:"message"`
}

// SourceLocation is additional information about the source code location that produced the log entry.
type SourceLocation struct {
	// Optional. Source file name. Depending on the runtime environment,
	// this might be a simple name or a fully-qualified name.
	File string `json:"file,omitempty"`
	// Optional. Line within the source file. 1-based; 0 indicates no line number available.
	Line int64 `json:"line,omitempty"`
	// Optional. Human-readable name of the function or method being invoked,
	// with optional context such as the class or package name.
	// This information may be used in contexts such as the logs viewer,
	// where a file and line number are less meaningful. The format can vary by language.
	// For example: qual.if.ied.Class.method (Java), dir/package.func (Go), function (Python).
	Function string `json:"function,omitempty"`
}

func newSourceLocation(pc uintptr, file string, line int, ok bool) *SourceLocation {
	if !ok {
		return nil
	}

	var function string
	if f := runtime.FuncForPC(pc); f != nil {
		function = f.Name()
	}

	return &SourceLocation{
		File:     file,
		Line:     int64(line),
		Function: function,
	}
}

func log(w io.Writer, entry Entry) {
	buf, err := json.Marshal(entry)
	if err == nil {
		w.Write(append(buf, '\n'))
	}
}

func logWithLevel(w io.Writer, s Severity, loc *SourceLocation, msg string) {
	if Enabled(s) {
		log(w, Entry{
			Severity:       s.String(),
			Message:        msg,
			SourceLocation: loc,
		})
	}
}

// Debug logs a message at SeverityDebug.
func Debug(msg string) {
	logWithLevel(os.Stdout, SeverityDebug, newSourceLocation(runtime.Caller(1)), msg)
}

// Info logs a message at SeverityInfo.
func Info(msg string) {
	logWithLevel(os.Stdout, SeverityInfo, newSourceLocation(runtime.Caller(1)), msg)
}

// Warn logs a message at SeverityWarning.
func Warn(msg string) {
	logWithLevel(os.Stdout, SeverityWarning, newSourceLocation(runtime.Caller(1)), msg)
}

// Error logs a message at SeverityError.
func Error(msg string) {
	logWithLevel(os.Stdout, SeverityError, newSourceLocation(runtime.Caller(1)), msg)
}

// ReportError outputs a log with stacktrace so that error reporting can recognize the error.
func ReportError(msg string) {
	logWithLevel(os.Stdout, SeverityError, newSourceLocation(runtime.Caller(1)), fmt.Sprintf("%s\n%s", msg, string(debug.Stack())))
}

func logWithSpan(w io.Writer, s Severity, span trace.Span, loc *SourceLocation, msg string) {
	if Enabled(s) {
		spanCtx := span.SpanContext()
		if span.IsRecording() && spanCtx.HasTraceID() && spanCtx.HasSpanID() {
			log(w, Entry{
				Severity:       s.String(),
				Trace:          fmt.Sprintf("projects/%s/traces/%s", projectID, spanCtx.TraceID.String()),
				SpanID:         spanCtx.SpanID.String(),
				SourceLocation: loc,
				Message:        msg,
			})
		} else {
			log(w, Entry{
				Severity:       s.String(),
				SourceLocation: loc,
				Message:        msg,
			})
		}
	}
}

// DebugWithSpan logs a message at SeverityDebug.
func DebugWithSpan(span trace.Span, msg string) {
	logWithSpan(os.Stdout, SeverityDebug, span, newSourceLocation(runtime.Caller(1)), msg)
}

// InfoWithSpan logs a message at SeverityInfo.
func InfoWithSpan(span trace.Span, msg string) {
	logWithSpan(os.Stdout, SeverityInfo, span, newSourceLocation(runtime.Caller(1)), msg)
}

// WarnWithSpan logs a message at SeverityWarning.
func WarnWithSpan(span trace.Span, msg string) {
	logWithSpan(os.Stdout, SeverityWarning, span, newSourceLocation(runtime.Caller(1)), msg)
}

// ErrorWithSpan logs a message at SeverityError.
func ErrorWithSpan(span trace.Span, msg string) {
	logWithSpan(os.Stdout, SeverityError, span, newSourceLocation(runtime.Caller(1)), msg)
}

// ReportErrorWithSpan outputs a log with stacktrace so that error reporting can recognize the error.
func ReportErrorWithSpan(span trace.Span, msg string) {
	logWithSpan(os.Stdout, SeverityError, span, newSourceLocation(runtime.Caller(1)), fmt.Sprintf("%s\n%s", msg, string(debug.Stack())))
}

// DebugWithCtx logs a message at SeverityDebug.
func DebugWithCtx(ctx context.Context, msg string) {
	logWithSpan(os.Stdout, SeverityDebug, trace.SpanFromContext(ctx), newSourceLocation(runtime.Caller(1)), msg)
}

// InfoWithCtx logs a message at SeverityInfo.
func InfoWithCtx(ctx context.Context, msg string) {
	logWithSpan(os.Stdout, SeverityInfo, trace.SpanFromContext(ctx), newSourceLocation(runtime.Caller(1)), msg)
}

// WarnWithCtx logs a message at SeverityWarning.
func WarnWithCtx(ctx context.Context, msg string) {
	logWithSpan(os.Stdout, SeverityWarning, trace.SpanFromContext(ctx), newSourceLocation(runtime.Caller(1)), msg)
}

// ErrorWithCtx logs a message at SeverityError.
func ErrorWithCtx(ctx context.Context, msg string) {
	logWithSpan(os.Stdout, SeverityError, trace.SpanFromContext(ctx), newSourceLocation(runtime.Caller(1)), msg)
}

// ReportErrorWithCtx outputs a log with stacktrace so that error reporting can recognize the error.
func ReportErrorWithCtx(ctx context.Context, msg string) {
	logWithSpan(os.Stdout, SeverityError, trace.SpanFromContext(ctx), newSourceLocation(runtime.Caller(1)), fmt.Sprintf("%s\n%s", msg, string(debug.Stack())))
}
