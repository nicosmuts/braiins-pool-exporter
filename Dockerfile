# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26.4-bookworm AS builder

WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -buildvcs=false \
    -ldflags="-s -w \
      -X github.com/nicosmuts/braiins-pool-exporter/internal/version.Version=${VERSION} \
      -X github.com/nicosmuts/braiins-pool-exporter/internal/version.Commit=${COMMIT} \
      -X github.com/nicosmuts/braiins-pool-exporter/internal/version.BuildDate=${BUILD_DATE}" \
    -o /out/braiins-pool-exporter \
    ./cmd/braiins-pool-exporter

FROM alpine:3.22

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN apk add --no-cache ca-certificates wget \
    && addgroup -S -g 65532 exporter \
    && adduser -S -D -H -u 65532 -G exporter exporter

COPY --from=builder --chown=65532:65532 /out/braiins-pool-exporter /usr/local/bin/braiins-pool-exporter

LABEL org.opencontainers.image.title="Braiins Pool Exporter" \
      org.opencontainers.image.description="Prometheus exporter for the official Braiins Pool API" \
      org.opencontainers.image.url="https://github.com/nicosmuts/braiins-pool-exporter" \
      org.opencontainers.image.source="https://github.com/nicosmuts/braiins-pool-exporter" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}"

USER 65532:65532
EXPOSE 9108

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:9108/-/healthy || exit 1

ENTRYPOINT ["/usr/local/bin/braiins-pool-exporter"]
