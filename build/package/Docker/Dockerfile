FROM alpine:latest AS build1

RUN apk --no-cache add ca-certificates tzdata \
    build-base clang libssh2-dev libssh2-static nghttp2-dev nghttp2-static openssl-dev \
    openssl-libs-static zlib-static

ADD https://github.com/curl/curl/releases/download/curl-7_84_0/curl-7.84.0.tar.xz /tmp

ENV CC=clang
RUN cd /tmp && \
    tar xf curl-7.84.0.tar.xz && \
    cd curl-7.84.0 && \
    LDFLAGS="-static" PKG_CONFIG="pkg-config --static" ./configure --disable-shared \
    --enable-static \
    --disable-ldap \
    --enable-ipv6 \
    --enable-unix-sockets \
    --with-ssl \
    --with-libssh2 && \
    make LDFLAGS="-static -all-static" && \
    strip src/curl && \
    ldd src/curl && exit 1 || true && \
    mv src/curl /bin/curl

FROM golang:1.18-alpine AS build2

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -ldflags "-extldflags '-static'" -o /bin/reader cmd/reader/main.go && \
    CGO_ENABLED=0 go build -ldflags "-extldflags '-static'" -o /bin/account cmd/account/main.go

FROM scratch

COPY --from=build1 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build1 /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build1 /bin/curl /bin/curl

WORKDIR /app

COPY --from=build2 /bin/reader ./reader
COPY --from=build2 /bin/account ./account

HEALTHCHECK \
    CMD [ "/bin/curl", "-f", "http://localhost:3000/ping" ]

CMD [ "./reader" ]
