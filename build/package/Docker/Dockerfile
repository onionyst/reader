# syntax=docker/dockerfile:1

FROM golang:1.25-alpine3.23 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/reader  ./cmd/reader  && \
    go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/account ./cmd/account && \
    go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/probe   ./cmd/probe

FROM alpine:3.23 AS runtime-files
RUN apk add --no-cache ca-certificates tzdata

FROM scratch
WORKDIR /app

COPY --from=runtime-files /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=runtime-files /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /out/reader  /app/reader
COPY --from=build /out/account /app/account
COPY --from=build /out/probe   /app/probe

USER 65532:65532

HEALTHCHECK --interval=10s --timeout=2s --retries=3 \
    CMD ["/app/probe", "-url", "http://127.0.0.1:3000/healthz"]

ENTRYPOINT ["/app/reader"]
