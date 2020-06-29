module github.com/mechiru/slog/example

go 1.14

replace github.com/mechiru/slog => ../

require (
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v0.2.0
	github.com/mechiru/slog v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v0.7.0
	golang.org/x/net v0.0.0-20200625001655-4c5254603344
)
