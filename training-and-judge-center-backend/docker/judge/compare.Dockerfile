# Image for the default output checker. Unlike the language runners it carries
# no toolchain: it runs our own static binary, never contestant code. It still
# needs a shell and sleep, which the pool relies on to keep containers alive
# between execs, and it mirrors the runners' user and layout so the sandbox
# behaves identically.
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/compare ./cmd/compare

FROM alpine:3.21

RUN adduser -D -u 1000 judge \
    && mkdir /sandbox \
    && chown judge:judge /sandbox

COPY --from=builder /bin/compare /usr/local/bin/compare

USER judge
WORKDIR /sandbox
