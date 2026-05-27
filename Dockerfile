# Cross-compile from the native builder platform (no QEMU emulation).
# BUILDPLATFORM = whatever the runner is (amd64 on GH free runners).
# TARGETOS/TARGETARCH = what the final image runs on.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1020 \
 && /go/bin/templ generate \
 && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o /out/snibox ./cmd/snibox

FROM alpine:3.20
ARG VERSION=dev
LABEL org.opencontainers.image.title="snibox-next" \
      org.opencontainers.image.description="Self-hosted snippets/notes/links library — Go + templ/HTMX + SQLite." \
      org.opencontainers.image.source="https://github.com/darious/snibox-next" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}"
RUN apk add --no-cache ca-certificates tzdata wget \
 && addgroup -S snibox && adduser -S -G snibox -u 10001 snibox \
 && mkdir -p /data && chown snibox:snibox /data
USER snibox
WORKDIR /data
VOLUME ["/data"]
EXPOSE 8979
ENV SNIBOX_ADDR=0.0.0.0:8979 \
    SNIBOX_DB=/data/snibox.db \
    SNIBOX_TRUST_NETWORK=true
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8979/healthz || exit 1
COPY --from=build /out/snibox /usr/local/bin/snibox
ENTRYPOINT ["/usr/local/bin/snibox"]
