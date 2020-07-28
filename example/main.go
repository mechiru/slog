package main

import (
	"os"
	"time"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/mechiru/slog"
	"go.opentelemetry.io/otel/api/global"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/net/context"
)

func main() {
	projectID := os.Getenv("PROJECT")

	slog.Setup(projectID /*, slog.WithSeverity(slog.SeverityWarning), slog.WithLogLevel("warning")*/)

	ctx := context.Background()
	setupExporter(ctx, projectID)

	tr := global.Tracer("tracer")

	ctx, span := tr.Start(ctx, "github.com/mechiru/slog/example/main")
	defer span.End()

	slog.Debug("Debug/debug message")
	slog.Debugf("Debugf/%s", "debug message")
	slog.DebugWithSpan(span, "DebugWithSpan/debug message")
	slog.DebugWithSpanf(span, "DebugWithSpanf/%s", "debug message")
	slog.DebugWithCtx(ctx, "DebugWithCtx/debug message")
	slog.DebugWithCtxf(ctx, "DebugWithCtxf/%s", "debug message")
	sleep()

	slog.Info("Info/info message")
	slog.Infof("Infof/%s", "info message")
	slog.InfoWithSpan(span, "InfoWithSpan/info message")
	slog.InfoWithSpanf(span, "InfoWithSpanf/%s", "info message")
	slog.InfoWithCtx(ctx, "InfoWithCtx/info message")
	slog.InfoWithCtxf(ctx, "InfoWithCtxf/%s", "info message")
	sleep()

	slog.Warn("Warn/warn message")
	slog.Warnf("Warnf/%s", "warn message")
	slog.WarnWithSpan(span, "WarnWithSpan/warn message")
	slog.WarnWithSpanf(span, "WarnWithSpanf/%s", "warn message")
	slog.WarnWithCtx(ctx, "WarnWithCtx/warn message")
	slog.WarnWithCtxf(ctx, "WarnWithCtx/%s", "warn message")
	sleep()

	slog.Error("Error/error message")
	slog.Errorf("Errorf/%s", "error message")
	slog.ErrorWithSpan(span, "ErrorWithSpan/error message")
	slog.ErrorWithSpanf(span, "ErrorWithSpanf/%s", "error message")
	slog.ErrorWithCtx(ctx, "ErrorWithCtx/error message")
	slog.ErrorWithCtxf(ctx, "ErrorWithCtxf/%s", "error message")
	sleep()

	slog.ReportError("ReportError/error message")
	slog.ReportErrorf("ReportErrorf/%s", "error message")
	slog.ReportErrorWithSpan(span, "ErrorWithSpan/error message")
	slog.ReportErrorWithSpanf(span, "ErrorWithSpanf/%s", "error message")
	slog.ReportErrorWithCtx(ctx, "ErrorWithCtx/error message")
	slog.ReportErrorWithCtxf(ctx, "ErrorWithCtxf/%s", "error message")
	sleep()
}

func sleep() { time.Sleep(500 * time.Millisecond) }

func setupExporter(ctx context.Context, projectID string) error {
	exporter, err := texporter.NewExporter(
		texporter.WithContext(ctx),
		texporter.WithProjectID(projectID),
	)
	if err != nil {
		return err
	}

	tp, err := sdktrace.NewProvider(sdktrace.WithSyncer(exporter))
	if err != nil {
		return err
	}

	global.SetTraceProvider(tp)
	return nil
}
