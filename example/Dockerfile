ARG GOVERSION=1.14
ARG OS=buster

FROM golang:${GOVERSION}-${OS} AS builder
WORKDIR /app
COPY ./ ./
RUN apt-get update -y \
	&& apt-get install ca-certificates -y \
	&& update-ca-certificates \
	&& GO111MODULE=on CGO_ENABLED=0 go build -v -mod=readonly -o=/app/main main.go

FROM debian:${OS} AS runtime
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app /app
RUN groupadd -g 1000 go \
	&& useradd -g go -u 1000 -M -s /bin/bash go \
	&& chown go:go /app/main
USER go
ENTRYPOINT ["./main"]
