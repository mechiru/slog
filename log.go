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
	"strings"
	"sync"

	"go.opentelemetry.io/otel/api/trace"
)

var (
	state              uint32 = stateUninitialized
	stateUninitialized uint32 = 0
	stateInitialized   uint32 = 1

	mu sync.Mutex

	projectID string
	severity  Severity = SeverityDebug
)

// An Option is an option for a slog package.
type Option func()

// WithSeverity returns an Option that specifies a severity.
// Default severity is SeverityDebug.
func WithSeverity(s Severity) Option {
	return func() { severity = s }
}

// WithLogLevel returns an Option that specifies a log level.
func WithLogLevel(lvl string) Option {
	return func() { severity = toSeverity(lvl) }
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
func Enabled(s Severity) bool { return s >= severity }

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

func toSeverity(lvl string) Severity {
	lvl = strings.ToUpper(lvl)
	for idx, s := range severities {
		if lvl == s {
			return Severity(idx)
		}
	}
	return SeverityDefault
}

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

func location(skip int) *SourceLocation {
	pc, file, line, ok := runtime.Caller(skip + 1)
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

func write(w io.Writer, entry Entry) error {
	return json.NewEncoder(w).Encode(entry)
}

func log(s Severity, msg string) error {
	return write(os.Stdout, Entry{
		Severity:       s.String(),
		SourceLocation: location(2),
		Message:        msg,
	})
}

// Debug logs a message at SeverityDebug.
func Debug(msg string) (err error) {
	if Enabled(SeverityDebug) {
		return log(SeverityDebug, msg)
	}
	return
}

// Debugf logs a message at SeverityDebug.
func Debugf(format string, a ...interface{}) (err error) {
	if Enabled(SeverityDebug) {
		return log(SeverityDebug, fmt.Sprintf(format, a...))
	}
	return
}

// Info logs a message at SeverityInfo.
func Info(msg string) (err error) {
	if Enabled(SeverityInfo) {
		return log(SeverityInfo, msg)
	}
	return
}

// Infof logs a message at SeverityInfo.
func Infof(format string, a ...interface{}) (err error) {
	if Enabled(SeverityInfo) {
		return log(SeverityInfo, fmt.Sprintf(format, a...))
	}
	return
}

// Warn logs a message at SeverityWarning.
func Warn(msg string) (err error) {
	if Enabled(SeverityWarning) {
		return log(SeverityWarning, msg)
	}
	return
}

// Warnf logs a message at SeverityWarning.
func Warnf(format string, a ...interface{}) (err error) {
	if Enabled(SeverityWarning) {
		return log(SeverityWarning, fmt.Sprintf(format, a...))
	}
	return
}

// Error logs a message at SeverityError.
func Error(msg string) (err error) {
	if Enabled(SeverityError) {
		return log(SeverityError, msg)
	}
	return
}

// Errorf logs a message at SeverityError.
func Errorf(format string, a ...interface{}) (err error) {
	if Enabled(SeverityError) {
		return log(SeverityError, fmt.Sprintf(format, a...))
	}
	return
}

// ReportError outputs a log with stacktrace so that error reporting can recognize the error.
func ReportError(msg string) (err error) {
	if Enabled(SeverityError) {
		return log(SeverityError, fmt.Sprintf("%s\n%s", msg, string(debug.Stack())))
	}
	return
}

// ReportErrorf outputs a log with stacktrace so that error reporting can recognize the error.
func ReportErrorf(format string, a ...interface{}) (err error) {
	if Enabled(SeverityError) {
		return log(SeverityError, fmt.Sprintf(format+"\n%s", append(a, string(debug.Stack()))))
	}
	return
}

func logWithSpan(s Severity, span trace.Span, msg string) error {
	if spanCtx := span.SpanContext(); span.IsRecording() && spanCtx.HasTraceID() && spanCtx.HasSpanID() {
		return write(os.Stdout, Entry{
			Severity:       s.String(),
			Trace:          fmt.Sprintf("projects/%s/traces/%s", projectID, spanCtx.TraceID.String()),
			SpanID:         spanCtx.SpanID.String(),
			SourceLocation: location(2),
			Message:        msg,
		})
	} else {
		return write(os.Stdout, Entry{
			Severity:       s.String(),
			SourceLocation: location(2),
			Message:        msg,
		})
	}
}

// DebugWithSpan logs a message at SeverityDebug.
func DebugWithSpan(span trace.Span, msg string) (err error) {
	if Enabled(SeverityDebug) {
		return logWithSpan(SeverityDebug, span, msg)
	}
	return
}

// DebugWithSpanf logs a message at SeverityDebug.
func DebugWithSpanf(span trace.Span, format string, a ...interface{}) (err error) {
	if Enabled(SeverityDebug) {
		return logWithSpan(SeverityDebug, span, fmt.Sprintf(format, a...))
	}
	return
}

// InfoWithSpan logs a message at SeverityInfo.
func InfoWithSpan(span trace.Span, msg string) (err error) {
	if Enabled(SeverityInfo) {
		return logWithSpan(SeverityInfo, span, msg)
	}
	return
}

// InfoWithSpanf logs a message at SeverityInfo.
func InfoWithSpanf(span trace.Span, format string, a ...interface{}) (err error) {
	if Enabled(SeverityInfo) {
		return logWithSpan(SeverityInfo, span, fmt.Sprintf(format, a...))
	}
	return
}

// WarnWithSpan logs a message at SeverityWarning.
func WarnWithSpan(span trace.Span, msg string) (err error) {
	if Enabled(SeverityWarning) {
		return logWithSpan(SeverityWarning, span, msg)
	}
	return
}

// WarnWithSpanf logs a message at SeverityWarning.
func WarnWithSpanf(span trace.Span, format string, a ...interface{}) (err error) {
	if Enabled(SeverityWarning) {
		return logWithSpan(SeverityWarning, span, fmt.Sprintf(format, a...))
	}
	return
}

// ErrorWithSpan logs a message at SeverityError.
func ErrorWithSpan(span trace.Span, msg string) (err error) {
	if Enabled(SeverityError) {
		return logWithSpan(SeverityError, span, msg)
	}
	return
}

// ErrorWithSpanf logs a message at SeverityError.
func ErrorWithSpanf(span trace.Span, format string, a ...interface{}) (err error) {
	if Enabled(SeverityError) {
		return logWithSpan(SeverityError, span, fmt.Sprintf(format, a...))
	}
	return
}

// ReportErrorWithSpan outputs a log with stacktrace so that error reporting can recognize the error.
func ReportErrorWithSpan(span trace.Span, msg string) (err error) {
	if Enabled(SeverityError) {
		return logWithSpan(SeverityError, span, fmt.Sprintf("%s\n%s", msg, string(debug.Stack())))
	}
	return
}

// ReportErrorWithSpanf outputs a log with stacktrace so that error reporting can recognize the error.
func ReportErrorWithSpanf(span trace.Span, format string, a ...interface{}) (err error) {
	if Enabled(SeverityError) {
		return logWithSpan(SeverityError, span, fmt.Sprintf(format+"\n%s", append(a, string(debug.Stack()))))
	}
	return
}

// DebugWithCtx logs a message at SeverityDebug.
func DebugWithCtx(ctx context.Context, msg string) (err error) {
	if Enabled(SeverityDebug) {
		return logWithSpan(SeverityDebug, trace.SpanFromContext(ctx), msg)
	}
	return
}

// DebugWithCtxf logs a message at SeverityDebug.
func DebugWithCtxf(ctx context.Context, format string, a ...interface{}) (err error) {
	if Enabled(SeverityDebug) {
		return logWithSpan(SeverityDebug, trace.SpanFromContext(ctx), fmt.Sprintf(format, a...))
	}
	return
}

// InfoWithCtx logs a message at SeverityInfo.
func InfoWithCtx(ctx context.Context, msg string) (err error) {
	if Enabled(SeverityInfo) {
		return logWithSpan(SeverityInfo, trace.SpanFromContext(ctx), msg)
	}
	return
}

// InfoWithCtxf logs a message at SeverityInfo.
func InfoWithCtxf(ctx context.Context, format string, a ...interface{}) (err error) {
	if Enabled(SeverityInfo) {
		return logWithSpan(SeverityInfo, trace.SpanFromContext(ctx), fmt.Sprintf(format, a...))
	}
	return
}

// WarnWithCtx logs a message at SeverityWarning.
func WarnWithCtx(ctx context.Context, msg string) (err error) {
	if Enabled(SeverityWarning) {
		return logWithSpan(SeverityWarning, trace.SpanFromContext(ctx), msg)
	}
	return
}

// WarnWithCtxf logs a message at SeverityWarning.
func WarnWithCtxf(ctx context.Context, format string, a ...interface{}) (err error) {
	if Enabled(SeverityWarning) {
		return logWithSpan(SeverityWarning, trace.SpanFromContext(ctx), fmt.Sprintf(format, a...))
	}
	return
}

// ErrorWithCtx logs a message at SeverityError.
func ErrorWithCtx(ctx context.Context, msg string) (err error) {
	if Enabled(SeverityError) {
		return logWithSpan(SeverityError, trace.SpanFromContext(ctx), msg)
	}
	return
}

// ErrorWithCtxf logs a message at SeverityError.
func ErrorWithCtxf(ctx context.Context, format string, a ...interface{}) (err error) {
	if Enabled(SeverityError) {
		return logWithSpan(SeverityError, trace.SpanFromContext(ctx), fmt.Sprintf(format, a...))
	}
	return
}

// ReportErrorWithCtx outputs a log with stacktrace so that error reporting can recognize the error.
func ReportErrorWithCtx(ctx context.Context, msg string) (err error) {
	if Enabled(SeverityError) {
		return logWithSpan(SeverityError, trace.SpanFromContext(ctx), fmt.Sprintf("%s\n%s", msg, string(debug.Stack())))
	}
	return
}

// ReportErrorWithCtx outputs a log with stacktrace so that error reporting can recognize the error.
func ReportErrorWithCtxf(ctx context.Context, format string, a ...interface{}) (err error) {
	if Enabled(SeverityError) {
		return logWithSpan(SeverityError, trace.SpanFromContext(ctx), fmt.Sprintf(format+"\n%s", append(a, string(debug.Stack()))))
	}
	return
}
