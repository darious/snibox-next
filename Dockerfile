FROM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1020 \
 && /go/bin/templ generate \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/snibox ./cmd/snibox

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget \
 && addgroup -S snibox && adduser -S -G snibox -u 10001 snibox \
 && mkdir -p /data && chown snibox:snibox /data
USER snibox
WORKDIR /data
VOLUME ["/data"]
EXPOSE 8080
ENV SNIBOX_ADDR=0.0.0.0:8080 \
    SNIBOX_DB=/data/snibox.db \
    SNIBOX_TRUST_NETWORK=true
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --spider --quiet http://127.0.0.1:8080/healthz || exit 1
COPY --from=build /out/snibox /usr/local/bin/snibox
ENTRYPOINT ["/usr/local/bin/snibox"]
